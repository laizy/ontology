/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	"encoding/json"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/constants"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/genesis"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/signature"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/core/validation"
	"github.com/ontio/ontology/events"
	common2 "github.com/ontio/ontology/http/base/common"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	utils2 "github.com/ontio/ontology/smartcontract/service/native/utils"
)

func init() {
	log.Init(log.PATH, log.Stdout)
	runtime.GOMAXPROCS(4)
}

//var blockBuf *bytes.Buffer

func main() {
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	datadir := "testdata"
	os.RemoveAll(datadir)
	log.Trace("Node version: ", config.Version)

	acct := account.NewAccount("")
	buf := keypair.SerializePublicKey(acct.PublicKey)
	config.DefConfig.Genesis.ConsensusType = "solo"
	config.DefConfig.Genesis.SOLO.GenBlockTime = 3
	config.DefConfig.Genesis.SOLO.Bookkeepers = []string{hex.EncodeToString(buf)}

	log.Debug("The Node's PublicKey ", acct.PublicKey)

	bookkeepers := []keypair.PublicKey{acct.PublicKey}
	//Init event hub
	events.Init()

	log.Info("1. Loading the Ledger")
	var err error
	ledger.DefLedger, err = ledger.NewLedger(datadir)
	if err != nil {
		log.Fatalf("NewLedger error %s", err)
		os.Exit(1)
	}
	genblock, err := genesis.BuildGenesisBlock(bookkeepers, config.DefConfig.Genesis)
	if err != nil {
		log.Error(err)
		return
	}
	err = ledger.DefLedger.Init(bookkeepers, genblock)
	if err != nil {
		log.Fatalf("DefLedger.Init error %s", err)
		os.Exit(1)
	}

	//blockBuf = bytes.NewBuffer(nil)
	BlockGen(acct)

	//ioutil.WriteFile("blocks.bin", blockBuf.Bytes(), 0777)
}

func GenAccounts(num int) []*account.Account {
	var accounts []*account.Account
	for i := 0; i < num; i++ {
		acc := account.NewAccount("")
		accounts = append(accounts, acc)
	}
	return accounts
}

func signTransaction(signer *account.Account, tx *types.MutableTransaction) error {
	hash := tx.Hash()
	sign, _ := signature.Sign(signer, hash[:])
	tx.Sigs = append(tx.Sigs, types.Sig{
		PubKeys: []keypair.PublicKey{signer.PublicKey},
		M:       1,
		SigData: [][]byte{sign},
	})
	return nil
}

func BlockGen(issuer *account.Account) {
	// 生成N个账户
	// 构造交易向这些账户转一个ont
	N := 10000 // 要小于max uint16
	M := 10000
	H := 100
	accounts := GenAccounts(N)
	balance := make(map[int]uint64)

	// setup
	tsTx := make([]*types.Transaction, N)
	for i := 0; i < len(tsTx); i++ {
		ont := uint64(100)
		balance[i] = ont
		mutable := NewTransferTransaction(utils2.OntContractAddress, issuer.Address, accounts[i].Address, ont, 0, 100000)
		if err := signTransaction(issuer, mutable); err != nil {
			fmt.Println("signTransaction error:", err)
			os.Exit(1)
		}
		tx, _ := mutable.IntoImmutable()
		validation.VerifyTransaction(tx)
		tsTx[i] = tx
	}

	numBlock := 1
	for i := 0; i < numBlock; i++ {
		start := i * N / numBlock
		end := (i + 1) * N / numBlock
		if end > N {
			end = N
		}

		block, _ := makeBlock(issuer, tsTx[start:end])
		//block.Serialize(blockBuf)
		err := ledger.DefLedger.AddBlock(block)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}
	}

	for h := 0; h < H; h++ {
		tsTx := make([]*types.Transaction, M)
		for i := 0; i < M; i++ {
			from := rand.Int() % N
			to := rand.Int() % N
			if balance[from] > 0 {
				balance[from] -= 1
				balance[to] += 1
				mutable := NewTransferTransaction(utils2.OntContractAddress, accounts[from].Address,
					accounts[to].Address, 1, 0, 100000)
				if err := signTransaction(accounts[from], mutable); err != nil {
					fmt.Println("signTransaction error:", err)
					os.Exit(1)
				}
				tx, _ := mutable.IntoImmutable()
				validation.VerifyTransaction(tx)
				tsTx[i] = tx
			}
		}
		block, _ := makeBlock(issuer, tsTx)
		//block.Serialize(blockBuf)
		tstart := time.Now()

		err := ledger.DefLedger.AddBlock(block)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}
		tend := time.Now()

		fmt.Println("current block ", ledger.DefLedger.GetCurrentBlockHeight())

		fmt.Println("execute time:", tend.Sub(tstart).Nanoseconds()/int64(time.Millisecond))
		mem := runtime.MemStats{}
		runtime.ReadMemStats(&mem)
		rs, _ := json.Marshal(mem)
		fmt.Printf("current mem stats %v\n", string(rs))
	}

	// check result
	for i := 0; i < N; i++ {
		state := getState(accounts[i].Address)
		if state["ont"] != balance[i] {
			fmt.Printf("execution error , balance unmarched. expected:%d, got: %d\n", balance[i], state["ont"])
		}
	}

}

func getState(addr common.Address) map[string]uint64 {
	ont := new(big.Int)
	ong := new(big.Int)
	appove := new(big.Int)

	ontBalance, _ := ledger.DefLedger.GetStorageItem(utils2.OntContractAddress, addr[:])
	if ontBalance != nil {
		ont = common.BigIntFromNeoBytes(ontBalance)
	}
	ongBalance, _ := ledger.DefLedger.GetStorageItem(utils2.OngContractAddress, addr[:])
	if ongBalance != nil {
		ong = common.BigIntFromNeoBytes(ongBalance)
	}

	appoveKey := append(utils2.OntContractAddress[:], addr[:]...)
	ongappove, _ := ledger.DefLedger.GetStorageItem(utils2.OngContractAddress, appoveKey[:])
	if ongappove != nil {
		appove = common.BigIntFromNeoBytes(ongappove)
	}

	rsp := make(map[string]uint64)
	rsp["ont"] = ont.Uint64()
	rsp["ong"] = ong.Uint64()
	rsp["ongAppove"] = appove.Uint64()

	return rsp
}

func makeBlock(acc *account.Account, txs []*types.Transaction) (*types.Block, error) {
	nextBookkeeper, err := types.AddressFromBookkeepers([]keypair.PublicKey{acc.PublicKey})
	if err != nil {
		return nil, fmt.Errorf("GetBookkeeperAddress error:%s", err)
	}
	prevHash := ledger.DefLedger.GetCurrentBlockHash()
	height := ledger.DefLedger.GetCurrentBlockHeight()

	nonce := uint64(height)
	txHash := []common.Uint256{}
	for _, t := range txs {
		txHash = append(txHash, t.Hash())
	}

	txRoot := common.ComputeMerkleRoot(txHash)
	if err != nil {
		return nil, fmt.Errorf("ComputeRoot error:%s", err)
	}

	blockRoot := ledger.DefLedger.GetBlockRootWithNewTxRoot(txRoot)
	header := &types.Header{
		Version:          0,
		PrevBlockHash:    prevHash,
		TransactionsRoot: txRoot,
		BlockRoot:        blockRoot,
		Timestamp:        constants.GENESIS_BLOCK_TIMESTAMP + height + 1,
		Height:           height + 1,
		ConsensusData:    nonce,
		NextBookkeeper:   nextBookkeeper,
	}
	block := &types.Block{
		Header:       header,
		Transactions: txs,
	}

	blockHash := block.Hash()

	sig, err := signature.Sign(acc, blockHash[:])
	if err != nil {
		return nil, fmt.Errorf("[Signature],Sign error:%s.", err)
	}

	block.Header.Bookkeepers = []keypair.PublicKey{acc.PublicKey}
	block.Header.SigData = [][]byte{sig}
	return block, nil
}

func NewOngTransferFromTransaction(from, to, sender common.Address, value, gasPrice, gasLimit uint64) *types.MutableTransaction {
	sts := &ont.TransferFrom{
		From:   from,
		To:     to,
		Sender: sender,
		Value:  value,
	}

	invokeCode, _ := common2.BuildNativeInvokeCode(utils2.OngContractAddress, 0, "transferFrom", []interface{}{sts})
	invokePayload := &payload.InvokeCode{
		Code: invokeCode,
	}
	tx := &types.MutableTransaction{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		TxType:   types.Invoke,
		Payer:    sender,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  invokePayload,
		Sigs:     nil,
	}

	return tx
}

func NewTransferTransaction(asset common.Address, from, to common.Address, value, gasPrice, gasLimit uint64) *types.MutableTransaction {
	var sts []*ont.State
	sts = append(sts, &ont.State{
		From:  from,
		To:    to,
		Value: value,
	})
	invokeCode, _ := common2.BuildNativeInvokeCode(asset, 0, "transfer", []interface{}{sts})
	invokePayload := &payload.InvokeCode{
		Code: invokeCode,
	}
	tx := &types.MutableTransaction{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		TxType:   types.Invoke,
		Payer:    from,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  invokePayload,
		Sigs:     nil,
	}

	return tx
}
