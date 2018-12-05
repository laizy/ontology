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

package vbft

import (
	"fmt"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/core/store"
	"github.com/ontio/ontology/core/store/overlaydb"
	"github.com/ontio/ontology/events/message"
)

type ChainStore struct {
	db              *ledger.Ledger
	chainedBlockNum uint32
	pendingBlocks   map[uint32]*Block
	execResult      *store.ExecuteResult
	needSubmitBlock bool
}

func OpenBlockStore(db *ledger.Ledger) (*ChainStore, error) {
	return &ChainStore{
		db:              db,
		chainedBlockNum: db.GetCurrentBlockHeight(),
		pendingBlocks:   make(map[uint32]*Block),
		execResult:      &store.ExecuteResult{},
		needSubmitBlock: false,
	}, nil
}

func (self *ChainStore) close() {
	// TODO: any action on ledger actor??
}

func (self *ChainStore) GetChainedBlockNum() uint32 {
	return self.chainedBlockNum
}

func (self *ChainStore) SetExecMerkeRoot(merkleRoot common.Uint256) {
	self.execResult.MerkleRoot = merkleRoot
}

func (self *ChainStore) GetExecMerkeRoot() common.Uint256 {
	return self.execResult.MerkleRoot
}

func (self *ChainStore) SetExecWriteSet(memdb *overlaydb.MemDB) {
	self.execResult.WriteSet = memdb
}

func (self *ChainStore) GetExecWriteSet() *overlaydb.MemDB {
	return self.execResult.WriteSet
}

func (self *ChainStore) ReloadFromLedger() {
	height := self.db.GetCurrentBlockHeight()
	if height > self.chainedBlockNum {
		// update chainstore height
		self.chainedBlockNum = height
		// remove persisted pending blocks
		newPending := make(map[uint32]*Block)
		for blkNum, blk := range self.pendingBlocks {
			if blkNum > height {
				newPending[blkNum] = blk
			}
		}
		// update pending blocks
		self.pendingBlocks = newPending
	}
}

func (self *ChainStore) AddBlock(block *Block, server *Server) error {
	if block == nil {
		return fmt.Errorf("try add nil block")
	}

	if block.getBlockNum() <= self.GetChainedBlockNum() {
		log.Warnf("chain store adding chained block(%d, %d)", block.getBlockNum(), self.GetChainedBlockNum())
		return nil
	}

	if block.Block.Header == nil {
		panic("nil block header")
	}
	self.pendingBlocks[block.getBlockNum()] = block

	blkNum := self.GetChainedBlockNum() + 1
	for {
		if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
			log.Infof("ledger adding chained block (%d, %d)", blkNum, self.GetChainedBlockNum())

			var err error
			if self.needSubmitBlock {
				if submitBlk, present := self.pendingBlocks[blkNum-1]; submitBlk != nil && present {
					err := self.db.SubmitBlock(submitBlk.Block, *self.execResult)
					if err != nil && blkNum > self.GetChainedBlockNum() {
						return fmt.Errorf("ledger add submitBlk (%d, %d) failed: %s", blkNum, self.GetChainedBlockNum(), err)
					}
					delete(self.pendingBlocks, blkNum-1)
				} else {
					break
				}
			}
			execResult, err := self.db.ExecuteBlock(blk.Block)
			if err != nil {
				log.Errorf("chainstore AddBlock GetBlockExecResult: %s", err)
				return fmt.Errorf("GetBlockExecResult: %s", err)
			}
			self.execResult = &execResult
			self.needSubmitBlock = true
			server.pid.Tell(
				&message.NotifyBlockCompleteMsg{
					Block: blk.Block,
				})
			self.chainedBlockNum = blkNum
			/*
				if blkNum != self.db.GetCurrentBlockHeight() {
					log.Errorf("!!! chain store added chained block (%d, %d): %s",
						blkNum, self.db.GetCurrentBlockHeight(), err)
				}
			*/
			blkNum++
		} else {
			break
		}
	}

	return nil
}

func (self *ChainStore) GetBlock(blockNum uint32) (*Block, error) {

	if blk, present := self.pendingBlocks[blockNum]; present {
		return blk, nil
	}

	block, err := self.db.GetBlockByHeight(uint32(blockNum))
	if err != nil {
		return nil, err
	}

	return initVbftBlock(block)
}
