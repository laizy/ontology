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
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"runtime"
	"time"

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
	"github.com/ontio/ontology/events"
	common2 "github.com/ontio/ontology/http/base/common"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	utils2 "github.com/ontio/ontology/smartcontract/service/native/utils"
)

const (
	DefaultMultiCoreNum = 4
)

func init() {
	log.Init(log.PATH, log.Stdout)
	runtime.GOMAXPROCS(4)
}

var blockBuf *bytes.Buffer

func main() {
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

	blockBuf = bytes.NewBuffer(nil)
	TxGen(acct)

	ioutil.WriteFile("blocks.bin", blockBuf.Bytes(), 0777)
}

func GenAccounts(num int) []*account.Account {
	var accounts []*account.Account
	for i := 0; i < num; i++ {
		acc := account.NewAccount("")
		accounts = append(accounts, acc)
	}
	return accounts
}

func signTransaction(signer *account.Account, tx *types.Transaction) error {
	hash := tx.Hash()
	sign, _ := signature.Sign(signer, hash[:])
	tx.Sigs = append(tx.Sigs, &types.Sig{
		PubKeys: []keypair.PublicKey{signer.PublicKey},
		M:       1,
		SigData: [][]byte{sign},
	})
	return nil
}

func TxGen(issuer *account.Account) {
	// 生成1000个账户
	// 构造交易向这些账户转一个ont，每个区块10笔交易
	N := 1000 // 要小于max uint16
	accounts := GenAccounts(N)

	tsTx := make([]*types.Transaction, N)
	for i := 0; i < len(tsTx); i++ {
		tsTx[i] = NewTransferTransaction(utils2.OntContractAddress, issuer.Address, accounts[i].Address, 1, 0, 100000)
		if err := signTransaction(issuer, tsTx[i]); err != nil {
			fmt.Println("signTransaction error:", err)
			os.Exit(1)
		}
	}

	ont := uint64(constants.ONT_TOTAL_SUPPLY)
	ong := uint64(0)
	ongappove := uint64(0)
	for i := 0; i < 10; i++ {
		block, _ := makeBlock(issuer, tsTx[i*100:(i+1)*100])
		block.Serialize(blockBuf)
		err := ledger.DefLedger.AddBlock(block)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		state := getState(issuer.Address)
		ongappove += ont * 5
		ont -= 100

		checkEq(state["ont"], ont)
		checkEq(state["ong"], ong)
		checkEq(state["ongAppove"], ongappove)

		fmt.Println(state)
	}

	// 账户0 转账给自己，区块高度为11，预计ong approve 为 (11-1)*5
	{

		tx := NewTransferTransaction(utils2.OntContractAddress, accounts[0].Address, accounts[0].Address, 1, 0, 100000)
		if err := signTransaction(accounts[0], tx); err != nil {
			fmt.Println("signTransaction error:", err)
			os.Exit(1)
		}
		block, _ := makeBlock(issuer, []*types.Transaction{tx})
		block.Serialize(blockBuf)
		err := ledger.DefLedger.AddBlock(block)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		state := getState(accounts[0].Address)
		checkEq(state["ont"], 1)
		checkEq(state["ong"], 0)
		ongapp := uint64((11 - 1) * 5)
		checkEq(state["ongAppove"], ongapp)
		fmt.Println(state)
	}

	// step 3 : claim ong
	// 账户0 调用transferFrom自己，区块高度为12，预计ong为 (11-1)*5, ong appove 回到0
	{
		ongapp := uint64((11 - 1) * 5)

		tx := NewOngTransferFromTransaction(utils2.OntContractAddress, accounts[0].Address, accounts[0].Address, ongapp, 0, 100000)
		if err := signTransaction(accounts[0], tx); err != nil {
			fmt.Println("signTransaction error:", err)
			os.Exit(1)
		}
		block, _ := makeBlock(issuer, []*types.Transaction{tx})
		block.Serialize(blockBuf)
		err := ledger.DefLedger.AddBlock(block)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		state := getState(accounts[0].Address)
		fmt.Println(state)
		checkEq(state["ont"], 1)
		checkEq(state["ong"], ongapp)
		checkEq(state["ongAppove"], 0)
	}

	// step4 ong 转账
	// 账户0 将400 ong 转给 issuer， 预计 账户0 ong为400， issuer 的ong为400
	{
		issuerState := getState(issuer.Address)
		tx := NewTransferTransaction(utils2.OngContractAddress, accounts[0].Address, issuer.Address, 25, 0, 100000)
		if err := signTransaction(accounts[0], tx); err != nil {
			fmt.Println("signTransaction error:", err)
			os.Exit(1)
		}
		block, _ := makeBlock(issuer, []*types.Transaction{tx})
		block.Serialize(blockBuf)
		err := ledger.DefLedger.AddBlock(block)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		state := getState(accounts[0].Address)
		fmt.Println(state)
		checkEq(state["ont"], 1)
		checkEq(state["ong"], 25)
		checkEq(state["ongAppove"], 0)

		state = getState(issuer.Address)
		fmt.Println(state)
		checkEq(state["ont"], issuerState["ont"])
		checkEq(state["ong"], 25)
		checkEq(state["ongAppove"], issuerState["ongAppove"])
	}

}

func checkEq(a, b uint64) {
	if a != b {
		panic(fmt.Sprintf("not equal. a %d, b %d", a, b))
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

func NewOngTransferFromTransaction(from, to, sender common.Address, value, gasPrice, gasLimit uint64) *types.Transaction {
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
	tx := &types.Transaction{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		TxType:   types.Invoke,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  invokePayload,
		Sigs:     make([]*types.Sig, 0, 0),
	}

	return tx
}

func NewTransferTransaction(asset common.Address, from, to common.Address, value, gasPrice, gasLimit uint64) *types.Transaction {
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
	tx := &types.Transaction{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		TxType:   types.Invoke,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  invokePayload,
		Sigs:     make([]*types.Sig, 0, 0),
	}

	return tx
}
