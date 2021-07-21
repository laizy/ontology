package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"

	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/store/leveldbstore"
	"github.com/ontio/ontology/core/store/overlaydb"
	evm2 "github.com/ontio/ontology/smartcontract/service/evm"
	storage2 "github.com/ontio/ontology/smartcontract/storage"
	"github.com/ontio/ontology/vm/evm"
	"github.com/ontio/ontology/vm/evm/params"
)

func Ensure(err error) {
	if err != nil {
		panic(err)
	}
}

func EnsureTrue(b bool) {
	if b != true {
		panic("must be true")
	}
}

func NewStateDB(txHash, blockHash common2.Hash) (*storage2.CacheDB, *storage2.StateDB) {
	memback := leveldbstore.NewMemLevelDBStore()
	overlay := overlaydb.NewOverlayDB(memback)

	cache := storage2.NewCacheDB(overlay)
	state := storage2.NewStateDB(cache, txHash, blockHash, NewOngBalanceHandle())

	return cache, state
}
func GetBlock(n uint64) *types.Block {
	client, err := ethclient.Dial("http://172.168.3.21:7545")
	Ensure(err)
	block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(n)))
	Ensure(err)
	return block
}

func DoCheck(jsonTxStore string, config *params.ChainConfig) bool {
	txStore := new(TxStore)
	TxStoreBytes := []byte(jsonTxStore)
	err := json.Unmarshal(TxStoreBytes, txStore)
	Ensure(err)

	log.Infof("start checking tx: %s", txStore.TxHash)

	cache, db := NewStateDB(txStore.TxHash, txStore.BlockHash)
	for _, state := range txStore.StateObjectStore {
		ethAccount := storage2.EthAccount{
			Nonce:    state.OriginAccount.Nonce,
			CodeHash: common2.HexToHash(state.OriginAccount.CodeHash),
		}
		cache.PutEthAccount(state.Address, ethAccount)
		if state.OriginalCode != "" {
			code := common2.Hex2Bytes(state.OriginalCode)
			codeHash := crypto.Keccak256Hash(code)
			cache.PutEthCode(codeHash, code)
		}
		balance, _ := new(big.Int).SetString(state.OriginAccount.Balance, 10)
		if balance != nil {
			addr := state.Address
			err := db.OngBalanceHandle.SetBalance(cache, common.Address(addr), balance)
			Ensure(err)
		}
		for _, oriStore := range state.OriginStorage {
			db.SetState(state.Address, common2.HexToHash(oriStore.Key), common2.HexToHash(oriStore.Value))
		}
	}

	txHex, err := hex.DecodeString(txStore.RawTx)
	Ensure(err)
	tx := &types.Transaction{}
	err = rlp.DecodeBytes(txHex, tx)
	Ensure(err)
	msg := Tx2Msg(*tx, txStore.From)
	height, err := strconv.Atoi(txStore.Height)
	Ensure(err)
	coinbase := txStore.Coinbase
	blockContext := evm2.NewEVMBlockContext(uint32(height), uint32(txStore.TimeStamp), nil)
	blockContext.GasLimit = txStore.GasLimit
	difficulty, _ := big.NewInt(0).SetString(txStore.Difficulty, 10)
	blockContext.Difficulty = difficulty
	blockContext.Coinbase = txStore.Coinbase
	blockContext.GetHash = func(n uint64) common2.Hash {
		if n+1 == uint64(height) {
			return txStore.PreBlockHash
		}
		log.Warnf("get block %d, curr block height: %d", n, height)
		block := GetBlock(n)
		return block.Hash()
	}
	vmenv := evm.NewEVM(blockContext, evm.TxContext{}, db, config, evm.Config{})
	txContext := evm2.NewEVMTxContext(msg)
	vmenv.Reset(txContext, db)
	_, err = evm2.ApplyMessage(vmenv, msg, coinbase)
	Ensure(err)

	for _, state := range txStore.StateObjectStore {
		balance := db.GetBalance(state.Address)
		if state.CurrentAccount.Balance != balance.String() && txStore.Coinbase != state.Address && state.Address != txStore.From {
			log.Errorf("balance of address: %s,  %s != %s ", state.Address.String(), state.CurrentAccount.Balance, balance.String())
			return false
		}
		for _, curStore := range state.CurrentStorage {
			value := db.GetState(state.Address, common2.HexToHash(curStore.Key))
			if curStore.Value != value.Hex() {
				log.Errorf("state value of address: %s at key:  %s != %s ", state.Address.String(), curStore.Value, value.Hex())
				return false
			}
		}
	}

	return true
}

func main() {
	scanner := bufio.NewScanner(bufio.NewReader(os.Stdin))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		EnsureTrue(DoCheck(line, NewMainnetConfig()))
	}
}

func NewMainnetConfig() *params.ChainConfig {
	return &params.ChainConfig{
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

func main2() {
	log.InitLog(1, log.Stdout)
	txDb, err := NewTxDb("./sqlite3.db")
	if err != nil {
		log.Errorf("NewTxDb error %s", err)
		return
	}

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
			passed := DoCheck(unVerifiedTx.Tx, NewMainnetConfig())
			err = txDb.UpdateTx(unVerifiedTx.TxHash, true, passed)
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
	From             common2.Address    `json:"from"`
	BlockHash        common2.Hash       `json:"blockHash"`
	PreBlockHash     common2.Hash       `json:"preBlockHash"`
	Coinbase         common2.Address    `json:"coinbase"`
	Difficulty       string             `json:"difficulty"`
	GasLimit         uint64             `json:"gasLimit"`
	TimeStamp        uint64             `json:"timeStamp"`
	TxHash           common2.Hash       `json:"txHash"`
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
	Address        common2.Address `json:"address"`
	OriginalCode   string          `json:"originalCode"`
	CurrentCode    string          `json:"currentCode"`
	OriginAccount  AccountStore    `json:"originAccount"`
	CurrentAccount AccountStore    `json:"currentAccount"`
	OriginStorage  []storage       `json:"originStorage"`
	CurrentStorage []storage       `json:"currentStorage"`
}
