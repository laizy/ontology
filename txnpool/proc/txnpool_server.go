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

// Package proc provides functions for handle messages from
// consensus/ledger/net/http/validators
package proc

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	ethcomm "github.com/ethereum/go-ethereum/common"
	ethtype "github.com/ethereum/go-ethereum/core/types"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	txtypes "github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	msgpack "github.com/ontio/ontology/p2pserver/message/msg_pack"
	p2p "github.com/ontio/ontology/p2pserver/net/protocol"
	tc "github.com/ontio/ontology/txnpool/common"
	"github.com/ontio/ontology/validator/stateful"
	"github.com/ontio/ontology/validator/stateless"
	"github.com/ontio/ontology/validator/types"
)

type serverPendingTx struct {
	tx             *txtypes.Transaction // Pending
	sender         tc.SenderType        // Indicate which sender tx is from
	ch             chan *tc.TxResult    // channel to send tx result
	checkingStatus *tc.CheckingStatus
}

// TXPoolServer contains all api to external modules
type TXPoolServer struct {
	mu     sync.RWMutex   // Sync mutex
	wg     sync.WaitGroup // Worker sync
	txPool *tc.TXPool     // The tx pool that holds the valid transaction

	//restore for the evm tx only
	eipTxPool     map[common.Address]*txList // The tx pool that holds the valid transaction
	pendingEipTxs map[common.Address]*txList // The tx pool that holds the valid transaction
	pendingNonces *txNoncer

	allPendingTxs         map[common.Uint256]*serverPendingTx // The txs that server is processing
	actor                 *actor.PID
	Net                   p2p.P2P
	slots                 chan struct{} // The limited slots for the new transaction
	height                uint32        // The current block height
	gasPrice              uint64        // Gas price to enforce for acceptance into the pool
	disablePreExec        bool          // Disbale PreExecute a transaction
	disableBroadcastNetTx bool          // Disable broadcast tx from network

	stateless *stateless.ValidatorPool
	stateful  *stateful.ValidatorPool
	rspCh     chan *types.CheckResponse // The channel of verified response
	stopCh    chan bool                 // stop routine
}

// NewTxPoolServer creates a new tx pool server to schedule workers to
// handle and filter inbound transactions from the network, http, and consensus.
func NewTxPoolServer(disablePreExec, disableBroadcastNetTx bool) *TXPoolServer {
	s := &TXPoolServer{}
	// Initial txnPool
	s.txPool = tc.NewTxPool()
	s.allPendingTxs = make(map[common.Uint256]*serverPendingTx)

	//init queue
	s.eipTxPool = make(map[common.Address]*txList)
	s.pendingEipTxs = make(map[common.Address]*txList)
	s.pendingNonces = newTxNoncer(ledger.DefLedger)

	s.slots = make(chan struct{}, tc.MAX_LIMITATION)
	for i := 0; i < tc.MAX_LIMITATION; i++ {
		s.slots <- struct{}{}
	}

	s.gasPrice = getGasPriceConfig()
	log.Infof("tx pool: the current local gas price is %d", s.gasPrice)

	s.disablePreExec = disablePreExec
	s.disableBroadcastNetTx = disableBroadcastNetTx
	// Create the given concurrent workers
	s.stateless = stateless.NewValidatorPool(2)
	s.stateful = stateful.NewValidatorPool(1)
	s.rspCh = make(chan *types.CheckResponse, tc.MAX_PENDING_TXN)
	s.stopCh = make(chan bool)
	go s.start()

	return s
}

func (server *TXPoolServer) start() {
	for {
		select {
		case <-server.stopCh:
			return
		case rsp, ok := <-server.rspCh:
			if ok {
				server.handleRsp(rsp)
			}
		}
	}
}

// getPendingListSize return the length of the pending tx list.
func (s *TXPoolServer) getPendingListSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.allPendingTxs)
}

func (s *TXPoolServer) getHeight() uint32 {
	return atomic.LoadUint32(&s.height)
}

func (s *TXPoolServer) setHeight(height uint32) {
	if height == 0 {
		return
	}
	atomic.StoreUint32(&s.height, height)
}

// getGasPrice returns the current gas price enforced by the transaction pool
func (s *TXPoolServer) getGasPrice() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gasPrice
}

func (s *TXPoolServer) GetPendingTx(hash common.Uint256) *serverPendingTx {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.allPendingTxs[hash]
}

// removePendingTx removes a transaction from the pending list
// when it is handled. And if the submitter of the valid transaction
// is from http, broadcast it to the network. Meanwhile, check if it
// is in the block from consensus.
func (s *TXPoolServer) removePendingTx(hash common.Uint256, err errors.ErrCode) {
	s.mu.Lock()

	pt, ok := s.allPendingTxs[hash]
	if !ok {
		s.mu.Unlock()
		return
	}

	if err == errors.ErrNoError && ((pt.sender == tc.HttpSender) ||
		(pt.sender == tc.NetSender && !s.disableBroadcastNetTx)) {
		if s.Net != nil {
			msg := msgpack.NewTxn(pt.tx)
			go s.Net.Broadcast(msg)
		}
	}

	replyTxResult(pt.ch, hash, err, err.Error())

	delete(s.allPendingTxs, hash)

	if len(s.allPendingTxs) < tc.MAX_LIMITATION {
		select {
		case s.slots <- struct{}{}:
		default:
			log.Debug("removePendingTx: slots is full")
		}
	}

	s.mu.Unlock()
}

// adds a transaction to the pending list, if the
// transaction is already in the pending list, just return false.
func (s *TXPoolServer) setPendingTx(tx *txtypes.Transaction, sender tc.SenderType, txResultCh chan *tc.TxResult) (*serverPendingTx, common.Uint256, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	replacedTxHash := common.UINT256_EMPTY

	if ok := s.allPendingTxs[tx.Hash()]; ok != nil {
		log.Debugf("setPendingTx: transaction %x already in the verifying process", tx.Hash())
		return nil, replacedTxHash, fmt.Errorf("duplicated transaction input detected")
	}

	// replace the same nonce tx
	if tx.TxType == txtypes.EIP155 {
		old, err := s.addEipPendingTx(tx)
		if err != nil { //same nonce with higher gasprice existed in pending list
			log.Debugf("setPendingTx: transaction with same nonce %x already in the verifying process", tx.Hash())
			return nil, replacedTxHash, fmt.Errorf("duplicated EIP155 transaction nonce input detected")
		}
		if old != nil {
			//s.removePendingTx(old.Hash(), errors.ErrHigherNonceExist)
			replacedTxHash = old.Hash()
		} else {
			currentNonce := s.pendingNonces.get(tx.Payer)
			if currentNonce != uint64(tx.Nonce) {
				fmt.Printf("tx nonce is not correct:want:%d, get:%d\n", currentNonce, tx.Nonce)
				return nil, replacedTxHash, fmt.Errorf("tx nonce is not correct:want:%d, get:%d", currentNonce, tx.Nonce)
			}
		}
	}

	pt := &serverPendingTx{
		tx:     tx,
		sender: sender,
		ch:     txResultCh,
		checkingStatus: &tc.CheckingStatus{
			PassedStateless: 0,
			PassedStateful:  0,
			CheckHeight:     0,
		},
	}
	s.allPendingTxs[tx.Hash()] = pt
	return pt, replacedTxHash, nil
}

func (s *TXPoolServer) startTxVerify(tx *txtypes.Transaction, sender tc.SenderType, txResultCh chan *tc.TxResult) bool {
	pt, replaced, err := s.setPendingTx(tx, sender, txResultCh)
	if err != nil && pt == nil {
		replyTxResult(txResultCh, tx.Hash(), errors.ErrDuplicateInput, err.Error())
		return false
	}
	if replaced != common.UINT256_EMPTY {
		s.removePendingTx(replaced, errors.ErrHigherNonceExist)
	}

	if tx := s.getTransaction(tx.Hash()); tx != nil {
		log.Debugf("verifyTx: transaction %x already in the txn pool", tx.Hash())
		s.removePendingTx(tx.Hash(), errors.ErrDuplicateInput)
		return false
	}

	s.stateless.SubmitVerifyTask(tx, s.rspCh)
	s.stateful.SubmitVerifyTask(tx, s.rspCh)
	return true

}

// GetPID returns an actor pid with the actor type, If the type
// doesn't exist, return nil.
func (s *TXPoolServer) GetPID() *actor.PID {
	return s.actor
}

// registers an actor with the actor type and pid.
func (s *TXPoolServer) RegisterActor(pid *actor.PID) {
	s.actor = pid
}

// Stop stops server and workers.
func (s *TXPoolServer) Stop() {
	if s.actor != nil {
		s.actor.Stop()
	}
	close(s.rspCh)
	close(s.stopCh)
	close(s.slots)
}

// getTransaction returns a transaction with the transaction hash.

func (s *TXPoolServer) getTransaction(hash common.Uint256) *txtypes.Transaction {
	return s.txPool.GetTransaction(hash)
}

// getTxPool returns a tx list for consensus.
func (s *TXPoolServer) getTxPool(byCount bool, height uint32) []*tc.VerifiedTx {
	s.setHeight(height)

	avlTxList, oldTxList := s.txPool.GetTxPool(byCount, height)

	for _, t := range oldTxList {
		s.delTransaction(t)
		s.reVerifyStateful(t, tc.NilSender)
	}

	return avlTxList
}

// getTxCount returns current tx count, including pending and verified
func (s *TXPoolServer) getTxCount() []uint32 {
	ret := make([]uint32, 0)
	ret = append(ret, uint32(s.txPool.GetTransactionCount()))
	ret = append(ret, uint32(s.getPendingListSize()))
	return ret
}

// getTxHashList returns a currently pending tx hash list
func (s *TXPoolServer) getTxHashList() []common.Uint256 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	txHashPool := s.txPool.GetTransactionHashList()
	ret := make([]common.Uint256, 0, len(s.allPendingTxs)+len(txHashPool))
	existedTxHash := make(map[common.Uint256]bool)
	for _, hash := range txHashPool {
		ret = append(ret, hash)
		existedTxHash[hash] = true
	}
	for _, v := range s.allPendingTxs {
		hash := v.tx.Hash()
		if !existedTxHash[hash] {
			ret = append(ret, hash)
			existedTxHash[hash] = true
		}
	}
	return ret
}

//clean the EIP txpool and eip pending txpool under the tx nonce
func (s *TXPoolServer) cleanEipTxPool(txs []*txtypes.Transaction) {
	for _, tx := range txs {
		if tx.TxType == txtypes.EIP155 {
			if _, ok := s.eipTxPool[tx.Payer]; ok {
				s.eipTxPool[tx.Payer].Forward(uint64(tx.Nonce))
				if s.eipTxPool[tx.Payer].Len() == 0 {
					delete(s.eipTxPool, tx.Payer)
				}
			}
			if _, ok := s.pendingEipTxs[tx.Payer]; ok {
				s.pendingEipTxs[tx.Payer].Forward(uint64(tx.Nonce))
				if s.pendingEipTxs[tx.Payer].Len() == 0 {
					delete(s.pendingEipTxs, tx.Payer)
				}
			}
		}
	}
}
func (s *TXPoolServer) cleanPendingNonce() {
	s.pendingNonces.clean()
}

// cleanTransactionList cleans the txs in the block from the ledger
func (s *TXPoolServer) cleanTransactionList(txs []*txtypes.Transaction, height uint32) {
	s.txPool.CleanTransactionList(txs)
	s.cleanEipTxPool(txs)
	s.cleanPendingNonce()

	// Check whether to update the gas price and remove txs below the threshold
	if height%tc.UPDATE_FREQUENCY == 0 {
		gasPrice := getGasPriceConfig()
		s.mu.Lock()
		oldGasPrice := s.gasPrice
		s.gasPrice = gasPrice
		s.mu.Unlock()
		if oldGasPrice != gasPrice {
			log.Infof("Transaction pool price threshold updated from %d to %d",
				oldGasPrice, gasPrice)
		}

		if oldGasPrice < gasPrice {
			s.txPool.RemoveTxsBelowGasPrice(gasPrice)
		}
	}
	// Cleanup tx pool
	if !s.disablePreExec {
		remain := s.txPool.Remain()
		for _, t := range remain {
			if ok, _ := preExecCheck(t); !ok {
				log.Debugf("cleanTransactionList: preExecCheck tx %x failed", t.Hash())
				continue
			}
			s.reVerifyStateful(t, tc.NilSender)
		}
	}
}

// delTransaction deletes a transaction in the tx pool.
func (s *TXPoolServer) delTransaction(t *txtypes.Transaction) {
	s.txPool.DelTxList(t)
}

// addTxList adds a valid transaction to the tx pool.
func (s *TXPoolServer) addTxList(txEntry *tc.VerifiedTx) bool { //solve the EIP155
	eipFlag := false
	if txEntry.Tx.TxType == txtypes.EIP155 {
		eipFlag = true
		pendingNonce := s.Nonce(txEntry.Tx.Payer)
		ledgerNonce := s.CurrentNonce(txEntry.Tx.Payer)

		if pendingNonce < ledgerNonce {
			pendingNonce = ledgerNonce
		}
		if uint64(txEntry.Tx.Nonce) != pendingNonce {
			log.Errorf("tx nonce is not correct .want :%d, get :%d", pendingNonce, txEntry.Tx.Nonce)
			return false
		}
	}
	ret := s.txPool.AddTxList(txEntry)
	if eipFlag && ret {
		s.pendingNonces.set(txEntry.Tx.Payer, uint64(txEntry.Tx.Nonce+1))
	}
	return ret
}

func (s *TXPoolServer) addEIPTxPool(trans *txtypes.Transaction) error {
	if trans.TxType != txtypes.EIP155 {
		return fmt.Errorf("not an EIP155 tx")
	}
	if _, ok := s.eipTxPool[trans.Payer]; !ok {
		s.eipTxPool[trans.Payer] = newTxList(true)
	}

	//does the same nonce exist?
	old := s.eipTxPool[trans.Payer].txs.Get(uint64(trans.Nonce))
	if old == nil {
		s.eipTxPool[trans.Payer].txs.Put(trans)
	} else {
		//this should not happend
		hash := old.Hash()
		log.Errorf("an same nonce EIP155 tx hash:%s,nonce:%d exist!", hash.ToHexString(), old.Nonce)
		return fmt.Errorf("same nonce exist")
	}
	return nil
}

//return the replace tx if exist
func (s *TXPoolServer) addEipPendingTx(tx *txtypes.Transaction) (*txtypes.Transaction, error) {
	if tx.TxType != txtypes.EIP155 {
		return nil, fmt.Errorf("not an EIP155 tx")
	}

	if _, ok := s.pendingEipTxs[tx.Payer]; !ok {
		s.pendingEipTxs[tx.Payer] = newTxList(true)
	}

	old := s.pendingEipTxs[tx.Payer].txs.Get(uint64(tx.Nonce))
	if old == nil {
		s.pendingEipTxs[tx.Payer].txs.Put(tx)
	} else {
		if old.GasPrice < tx.GasPrice {
			s.pendingEipTxs[tx.Payer].txs.Remove(uint64(old.Nonce))
			s.pendingEipTxs[tx.Payer].txs.Put(tx)
		} else {
			return nil, fmt.Errorf("same nonce existed")
		}
	}
	return old, nil
}

// checkTx checks whether a transaction is in the pending list or
// the transacton pool
func (s *TXPoolServer) checkTx(hash common.Uint256) bool {
	// Check if the tx is in pending list
	s.mu.RLock()
	if ok := s.allPendingTxs[hash]; ok != nil {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()

	// Check if the tx is in txn pool
	if res := s.txPool.GetTransaction(hash); res != nil {
		return true
	}

	return false
}

// getTxStatusReq returns a transaction's status with the transaction hash.
func (s *TXPoolServer) getTxStatusReq(hash common.Uint256) *tc.TxStatus {
	if ret := s.GetPendingTx(hash); ret != nil {
		return &tc.TxStatus{
			Hash:  hash,
			Attrs: ret.checkingStatus.GetTxAttr(),
		}
	}

	return s.txPool.GetTxStatus(hash)
}

// getTransactionCount returns the tx size of the transaction pool.
func (s *TXPoolServer) getTransactionCount() int {
	return s.txPool.GetTransactionCount()
}

// re-verify a transaction's stateful data.
func (s *TXPoolServer) reVerifyStateful(tx *txtypes.Transaction, sender tc.SenderType) {
	pt, replaced, err := s.setPendingTx(tx, sender, nil)
	if pt == nil && err != nil {
		return
	}
	if replaced != common.UINT256_EMPTY {
		s.removePendingTx(replaced, errors.ErrHigherNonceExist)
	}
	s.removePendingTx(pt.tx.Hash(), errors.ErrHigherNonceExist)

	pt.checkingStatus.SetStateless()
	s.stateful.SubmitVerifyTask(tx, s.rspCh)
}

// verifies the block from consensus.
// There are three cases to handle.
// 1, for those unverified txs, assign them to the available worker;
// 2, for those verified txs whose height >= block's height, nothing to do;
// 3, for those verified txs whose height < block's height, re-verify their
// stateful data.
func (s *TXPoolServer) verifyBlock(req *tc.VerifyBlockReq, sender *actor.PID) {
	if req == nil || len(req.Txs) == 0 {
		return
	}

	s.setHeight(req.Height)

	processedTxs := make([]*tc.VerifyTxResult, len(req.Txs))

	// Check whether a tx's gas price is lower than the required, if yes, just return error
	txs := make(map[common.Uint256]*txtypes.Transaction, len(req.Txs))
	for _, t := range req.Txs {
		if t.GasPrice < s.gasPrice {
			entry := &tc.VerifyTxResult{
				Height:  req.Height,
				Tx:      t,
				ErrCode: errors.ErrGasPrice,
			}
			processedTxs = append(processedTxs, entry)
			sender.Tell(&tc.VerifyBlockRsp{TxnPool: processedTxs})
			return
		}

		// Check whether double spent
		if _, ok := txs[t.Hash()]; ok {
			entry := &tc.VerifyTxResult{
				Height:  req.Height,
				Tx:      t,
				ErrCode: errors.ErrDoubleSpend,
			}
			processedTxs = append(processedTxs, entry)
			sender.Tell(&tc.VerifyBlockRsp{TxnPool: processedTxs})
			return
		}
		txs[t.Hash()] = t
	}

	checkBlkResult := s.txPool.GetUnverifiedTxs(req.Txs, req.Height)

	if len(checkBlkResult.UnverifiedTxs) > 0 {
		ch := make(chan *types.CheckResponse, len(checkBlkResult.UnverifiedTxs))
		validator := stateless.NewValidatorPool(5)
		for _, t := range checkBlkResult.UnverifiedTxs {
			validator.SubmitVerifyTask(t, ch)
		}
		for i := 0; i < len(checkBlkResult.UnverifiedTxs); i++ {
			response := <-ch
			if response.ErrCode != errors.ErrNoError {
				processedTxs = append(processedTxs, &tc.VerifyTxResult{
					Height:  req.Height,
					Tx:      txs[response.Hash],
					ErrCode: response.ErrCode,
				})
				sender.Tell(&tc.VerifyBlockRsp{TxnPool: processedTxs})
				return
			}
		}
	}

	lenStateFul := len(checkBlkResult.UnverifiedTxs) + len(checkBlkResult.OldTxs)
	if lenStateFul > 0 {
		currHeight := ledger.DefLedger.GetCurrentBlockHeight()
		for currHeight < req.Height {
			// wait ledger sync up
			log.Warnf("ledger need sync up for tx verification, curr height: %d, expected:%d", currHeight, req.Height)
			time.Sleep(time.Second)
			currHeight = ledger.DefLedger.GetCurrentBlockHeight()
		}

		ch := make(chan *types.CheckResponse, lenStateFul)
		validator := stateful.NewValidatorPool(1)
		for _, tx := range checkBlkResult.UnverifiedTxs {
			validator.SubmitVerifyTask(tx, ch)
		}
		for _, tx := range checkBlkResult.OldTxs {
			validator.SubmitVerifyTask(tx, ch)
		}
		for i := 0; i < lenStateFul; i++ {
			resp := <-ch
			processedTxs = append(processedTxs, &tc.VerifyTxResult{
				Height:  resp.Height,
				Tx:      txs[resp.Hash],
				ErrCode: resp.ErrCode,
			})
			if resp.ErrCode != errors.ErrNoError {
				sender.Tell(&tc.VerifyBlockRsp{TxnPool: processedTxs})
				return
			}
		}
	}

	processedTxs = append(processedTxs, checkBlkResult.VerifiedTxs...)
	sender.Tell(&tc.VerifyBlockRsp{TxnPool: processedTxs})
}

// handles the verified response from the validator and if
// the tx is valid, add it to the tx pool, or remove it from the pending
// list
func (server *TXPoolServer) handleRsp(rsp *types.CheckResponse) {
	pt := server.GetPendingTx(rsp.Hash)
	if pt == nil {
		return
	}
	if rsp.ErrCode != errors.ErrNoError {
		//Verify fail
		log.Debugf("handleRsp: validator %d transaction %x invalid: %s", rsp.Type, rsp.Hash, rsp.ErrCode.Error())
		server.removePendingTx(rsp.Hash, rsp.ErrCode)
		return
	}
	if rsp.Type == types.Stateful && rsp.Height < server.getHeight() {
		// If validator's height is less than the required one, re-validate it.
		server.stateful.SubmitVerifyTask(rsp.Tx, server.rspCh)
		return
	}
	switch rsp.Type {
	case types.Stateful:
		pt.checkingStatus.SetStateful(rsp.Height)
	case types.Stateless:
		pt.checkingStatus.SetStateless()
	}

	if pt.checkingStatus.GetStateless() && pt.checkingStatus.GetStateful() {
		txEntry := &tc.VerifiedTx{
			Tx:             pt.tx,
			VerifiedHeight: pt.checkingStatus.CheckHeight,
		}
		f := server.addTxList(txEntry)
		if f {
			server.addEIPTxPool(pt.tx)
		}
		server.removePendingTx(pt.tx.Hash(), errors.ErrNoError)
		server.removeEIPPendingTx(pt.tx)

	}
}

func (s *TXPoolServer) CurrentNonce(addr common.Address) uint64 {
	ethacct, err := ledger.DefLedger.GetEthAccount(ethcomm.Address(addr))
	if err != nil {
		return 0
	}
	return ethacct.Nonce

}

func (s *TXPoolServer) Nonce(addr common.Address) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.pendingNonces.get(addr)
}

func (s *TXPoolServer) removeEIPPendingTx(tx *txtypes.Transaction) {
	if _, ok := s.pendingEipTxs[tx.Payer]; ok {
		s.pendingEipTxs[tx.Payer].txs.Remove(uint64(tx.Nonce))
	}
}

func (s *TXPoolServer) PendingEIPTransactions() map[ethcomm.Address]map[uint64]*ethtype.Transaction {
	ret := make(map[ethcomm.Address]map[uint64]*ethtype.Transaction, 0)
	for k, v := range s.pendingEipTxs {
		m := make(map[uint64]*ethtype.Transaction, 0)
		for kt, vt := range v.txs.items {
			ethTx, err := vt.GetEIP155Tx()
			if err != nil {
				log.Errorf("error GetEIP155Tx:%s", err)
			}
			m[kt] = ethTx
		}
		ret[ethcomm.BytesToAddress(k[:])] = m
	}

	return ret
}
