package main

import (
	"encoding/hex"
	"encoding/json"
	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/store/ledgerstore"
	evm2 "github.com/ontio/ontology/smartcontract/service/evm"
	storage2 "github.com/ontio/ontology/smartcontract/storage"
	"github.com/ontio/ontology/vm/evm"
	"github.com/ontio/ontology/vm/evm/params"
	"math/big"
	"strconv"
)

var testBlockStore *ledgerstore.BlockStore
var testStateStore *ledgerstore.StateStore
var testLedgerStore *ledgerstore.LedgerStoreImp
var config *params.ChainConfig
var txDb *TxDb

func init() {
	log.InitLog(log.DebugLog)

	var err error
	testLedgerStore, err = ledgerstore.NewLedgerStore("test/ledger", 0)
	if err != nil {
		log.Errorf("NewLedgerStore error %s", err)
		return
	}

	testBlockDir := "test/block"
	testBlockStore, err = ledgerstore.NewBlockStore(testBlockDir, false)
	if err != nil {
		log.Errorf("NewBlockStore error %s", err)
		return
	}
	testStateDir := "test/state"
	merklePath := "test/" + ledgerstore.MerkleTreeStorePath
	testStateStore, err = ledgerstore.NewStateStore(testStateDir, merklePath, 1000)
	if err != nil {
		log.Errorf("NewStateStore error %s", err)
		return
	}

	txDb, err = NewTxDb("./sqlite3.db")
	if err != nil {
		log.Errorf("NewTxDb error %s", err)
		return
	}

	// TODO
	config = &params.ChainConfig{
		ChainID:        new(big.Int).SetUint64(1),
		HomesteadBlock: big.NewInt(1150000),
		DAOForkBlock:   big.NewInt(1920000),
		DAOForkSupport: false,
		EIP150Block:    big.NewInt(2463000),
		//EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(2675000),
		EIP158Block:         big.NewInt(2675000),
		ByzantiumBlock:      big.NewInt(4370000),
		ConstantinopleBlock: big.NewInt(7280000),
		PetersburgBlock:     big.NewInt(7280000),
		IstanbulBlock:       big.NewInt(9069000),
		MuirGlacierBlock:    big.NewInt(9200000),
		YoloV2Block:         nil,
	}
}

func main() {
	log.InitLog(1, log.Stdout)
	for {
		unVerifiedTxes, err := txDb.SelectBatchUnverifiedTx()
		if err != nil {
			log.Errorf("SelectBatchUnverifiedTx error %s", err)
			return
		}
		for _, unVerifiedTx := range unVerifiedTxes {
			// TestNet
			//config = &params.ChainConfig{
			//	ChainID:        new(big.Int).SetUint64(5),
			//	HomesteadBlock: big.NewInt(0),
			//	DAOForkBlock:   nil,
			//	DAOForkSupport: false,
			//	EIP150Block:    big.NewInt(0),
			//	//EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
			//	EIP155Block:         big.NewInt(0),
			//	EIP158Block:         big.NewInt(0),
			//	ByzantiumBlock:      big.NewInt(0),
			//	ConstantinopleBlock: big.NewInt(0),
			//	PetersburgBlock:     big.NewInt(0),
			//	IstanbulBlock:       big.NewInt(0),
			//	MuirGlacierBlock:    big.NewInt(0),
			//	YoloV2Block:         nil,
			//}
			// main
			config = &params.ChainConfig{
				ChainID:        new(big.Int).SetUint64(1),
				HomesteadBlock: big.NewInt(1150000),
				DAOForkBlock:   big.NewInt(1920000),
				DAOForkSupport: false,
				EIP150Block:    big.NewInt(2463000),
				//EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
				EIP155Block:         big.NewInt(2675000),
				EIP158Block:         big.NewInt(2675000),
				ByzantiumBlock:      big.NewInt(4370000),
				ConstantinopleBlock: big.NewInt(7280000),
				PetersburgBlock:     big.NewInt(7280000),
				IstanbulBlock:       big.NewInt(9069000),
				MuirGlacierBlock:    big.NewInt(9200000),
				YoloV2Block:         nil,
			}

			txStore := new(TxStore)
			TxStoreBytes := []byte(unVerifiedTx.Tx)
			if err != nil {
				log.Errorf("TxStoreBytes HexToBytes error %s", err)
				err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
				if err != nil {
					log.Errorf("UpdateTx error %s", err)
					return
				}
				continue
			}
			err = json.Unmarshal(TxStoreBytes, txStore)
			if err != nil {
				log.Errorf("json.Unmarshal error %s", err)
				err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
				if err != nil {
					log.Errorf("json.Unmarshal UpdateTx error %s", err)
					return
				}
				continue
			}

			cache := testLedgerStore.GetCacheDB()
			db := storage2.NewStateDB(cache, common2.HexToHash(txStore.TxHash), common2.HexToHash(txStore.BlockHash), OngBalanceHandle{})

			for _, state := range txStore.StateObjectStore {
				ethAccount := storage2.EthAccount{
					Nonce:    state.OriginAccount.Nonce,
					CodeHash: common2.HexToHash(state.OriginAccount.CodeHash),
				}
				cache.PutEthAccount(common2.HexToAddress(state.Address), ethAccount)
				if state.OriginalCode != "" {
					code := common2.Hex2Bytes(state.OriginalCode)
					codeHash := crypto.Keccak256Hash(code)
					cache.PutEthCode(codeHash, code)
				}
				balance, _ := new(big.Int).SetString(state.OriginAccount.Balance, 10)
				if balance != nil {
					addr := common2.HexToAddress(state.Address)
					err := OngBalanceHandle{}.SetBalance(cache, common.Address(addr), balance)
					if err != nil {
						log.Errorf("ong.OngBalanceHandle{}.SetBalance error %s", err)
						err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
						if err != nil {
							log.Errorf("UpdateTx error %s", err)
							return
						}
						continue
					}
				}
				for _, oriStore := range state.OriginStorage {
					db.SetState(common2.HexToAddress(state.Address), common2.HexToHash(oriStore.Key), common2.HexToHash(oriStore.Value))
				}
			}
			//err = db.Commit()
			if err != nil {
				log.Errorf("db.Commit() error %s", err)
				err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
				if err != nil {
					log.Errorf("UpdateTx error %s", err)
					return
				}
				continue
			}
			txHex, err := hex.DecodeString(txStore.RawTx)
			if err != nil {
				log.Errorf("DecodeString txHex error %s", err)
				err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
				if err != nil {
					log.Errorf("UpdateTx error %s", err)
					return
				}
				continue
			}
			tx := &types.Transaction{}
			err = rlp.DecodeBytes(txHex, tx)
			if err != nil {
				log.Errorf("DecodeString txHex error %s", err)
				err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
				if err != nil {
					log.Errorf("UpdateTx error %s", err)
					return
				}
				continue
			}
			if tx == nil {
				continue
			}
			msg := Tx2Msg(*tx, common2.HexToAddress(txStore.From))
			height, err := strconv.Atoi(txStore.Height)
			coinbase := common.Address(common2.HexToAddress(txStore.Coinbase))
			blockContext := evm2.NewEVMBlockContext(uint32(height), uint32(txStore.TimeStamp), testLedgerStore)
			vmenv := evm.NewEVM(blockContext, evm.TxContext{}, db, config, evm.Config{})
			txContext := evm2.NewEVMTxContext(msg)
			vmenv.Reset(txContext, db)
			_, err = evm2.ApplyMessage(vmenv, msg, common2.Address(coinbase))
			if err != nil {
				log.Errorf("ApplyMessage error %s", err)
				err := txDb.UpdateTx(unVerifiedTx.TxHash, true, false)
				if err != nil {
					log.Errorf("UpdateTx error %s", err)
					return
				}
				continue
			}
			flag := true
			for _, state := range txStore.StateObjectStore {
				balance := db.GetBalance(common2.HexToAddress(state.Address))
				if state.CurrentAccount.Balance != balance.String() && txStore.Coinbase != state.Address {
					flag = false
					break
				}
				for _, curStore := range state.CurrentStorage {
					value := db.GetState(common2.HexToAddress(state.Address), common2.HexToHash(curStore.Key))
					if curStore.Value != value.Hex() {
						flag = false
						break
					}
				}
			}
			err = txDb.UpdateTx(unVerifiedTx.TxHash, true, flag)
			if err != nil {
				log.Errorf("Finish UpdateTx error %s", err)
			}
		}
	}
}

func Tx2Msg(tx types.Transaction, from common2.Address) types.Message {
	return types.NewMessage(from, tx.To(), tx.Nonce(), tx.Value(), tx.Gas(), tx.GasPrice(), tx.Data(), true)
}

type TxStore struct {
	Height           string             `json:"height"`
	From             string             `json:"from"`
	BlockHash        string             `json:"blockHash"`
	Coinbase         string             `json:"coinbase"`
	TimeStamp        uint64             `json:"timeStamp"`
	TxHash           string             `json:"txHash"`
	TxIndex          int                `json:"txIndex"`
	RawTx            string             `json:"rawTx"`
	StateObjectStore []stateObjectStore `json:"stateObjectStore"`
}

type AccountStore struct {
	Nonce    uint64 `json:"nonce"`
	Balance  string `json:"balance"`
	CodeHash string `json:"codeHash"`
}

type storage struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type stateObjectStore struct {
	Address        string       `json:"address"`
	OriginalCode   string       `json:"originalCode"`
	CurrentCode    string       `json:"currentCode"`
	OriginAccount  AccountStore `json:"originAccount"`
	CurrentAccount AccountStore `json:"currentAccount"`
	OriginStorage  []storage    `json:"originStorage"`
	CurrentStorage []storage    `json:"currentStorage"`
}
