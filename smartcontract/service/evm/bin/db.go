package main

import (
	_ "github.com/mattn/go-sqlite3"

	"database/sql"
)

type TxDb struct {
	db       *sql.DB
	tx       *sql.Tx
	stmt     *sql.Stmt
	cache    uint64
	numCalls uint64
}

var insertStatement = `
INSERT INTO diffs
    (txHash, tx, verified, pass)
    VALUES
    ($1, $2, $3, $4)
`

var updateStatement = `
UPDATE diffs
SET verified=?, pass=?
WHERE txHash=?
`

var createStmt = `
CREATE TABLE IF NOT EXISTS diffs (
    "txHash" STRING NOT NULL PRIMARY KEY,
    "tx" STRING,
    "verified" BOOL,
    "pass" BOOL
)
`
var selectStmt = `
SELECT * from diffs WHERE verified = $1 LIMIT 1000
`

var selectTx = `
SELECT count(*) from diffs WHERE txHash = $1
`

func (txDb *TxDb) UpdateTx(txHash string, verified bool, pass bool) error {
	_, err := txDb.stmt.Exec(verified, pass, txHash)
	if err != nil {
		return err
	}
	txDb.numCalls += 1

	// if we had enough calls, commit it
	if txDb.numCalls >= txDb.cache {
		if err := txDb.ForceCommit(); err != nil {
			return err
		}
	}
	return nil
}

type TxInfo struct {
	TxHash   string
	Tx       string
	Verified bool
	Pass     bool
}

func (txDb *TxDb) SelectBatchUnverifiedTx() (c []*TxInfo, err error) {
	var rows *sql.Rows
	if rows, err = txDb.db.Query(selectStmt, false); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		r := new(TxInfo)
		if err = rows.Scan(&r.TxHash, &r.Tx, &r.Verified, &r.Pass); err != nil {
			c = nil
			return
		}
		c = append(c, r)
	}
	err = rows.Err()
	return
}

func (txDb *TxDb) ForceCommit() error {
	if err := txDb.tx.Commit(); err != nil {
		return err
	}
	return txDb.resetTx()
}

func (txDb *TxDb) resetTx() error {
	txDb.numCalls = 0

	tx, err := txDb.db.Begin()
	if err != nil {
		return err
	}
	txDb.tx = tx

	stmt, err := txDb.tx.Prepare(updateStatement)
	if err != nil {
		return err
	}
	txDb.stmt = stmt

	return nil
}

func (diff *TxDb) Close() error {
	return diff.db.Close()
}

func NewTxDb(path string) (*TxDb, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createStmt)
	if err != nil {
		return nil, err
	}

	txDb := &TxDb{db: db, cache: 1000}

	// initialize the transaction
	if err := txDb.resetTx(); err != nil {
		return nil, err
	}
	return txDb, nil
}
