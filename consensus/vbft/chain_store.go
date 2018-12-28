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

type PendingBlock struct {
	block      *Block
	execResult *store.ExecuteResult
}
type ChainStore struct {
	db              *ledger.Ledger
	chainedBlockNum uint32
	pendingBlocks   map[uint32]*PendingBlock
	server          *Server
	needSubmitBlock bool
}

func OpenBlockStore(db *ledger.Ledger, server *Server) (*ChainStore, error) {
	return &ChainStore{
		db:              db,
		chainedBlockNum: db.GetCurrentBlockHeight(),
		pendingBlocks:   make(map[uint32]*PendingBlock),
		server:          server,
		needSubmitBlock: false,
	}, nil
}

func (self *ChainStore) Load() error {
	merkleRoot, err := self.db.GetStateMerkleRoot(self.chainedBlockNum)
	if err != nil {
		log.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", self.chainedBlockNum, err)
		return fmt.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", self.chainedBlockNum, err)
	}
	self.SetExecMerkleRoot(self.chainedBlockNum, merkleRoot)
	writeSet := overlaydb.NewMemDB(1, 1)
	self.SetExecWriteSet(self.chainedBlockNum, writeSet)
	self.needSubmitBlock = false
	return nil
}

func (self *ChainStore) close() {
	// TODO: any action on ledger actor??
}

func (self *ChainStore) GetChainedBlockNum() uint32 {
	return self.chainedBlockNum
}

func (self *ChainStore) SetExecMerkleRoot(blkNum uint32, merkleRoot common.Uint256) {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		blk.execResult.MerkleRoot = merkleRoot
	} else {
		log.Errorf("SetExecMerkleRoot failed blkNum:%d,merkleRoot:%s", blkNum, merkleRoot.ToHexString())
	}
}

func (self *ChainStore) GetExecMerkleRoot(blkNum uint32) common.Uint256 {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		return blk.execResult.MerkleRoot
	}
	merkleRoot, err := self.server.ledger.GetStateMerkleRoot(blkNum)
	if err != nil {
		log.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", blkNum, err)
		return common.Uint256{}
	} else {
		return merkleRoot
	}

}

func (self *ChainStore) SetExecWriteSet(blkNum uint32, memdb *overlaydb.MemDB) {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		blk.execResult.WriteSet = memdb
	} else {
		log.Errorf("SetExecWriteSet failed blkNum:%d", blkNum)
	}
}

func (self *ChainStore) GetExecWriteSet(blkNum uint32) *overlaydb.MemDB {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		return blk.execResult.WriteSet
	}
	return nil
}

func (self *ChainStore) ReloadFromLedger() {
	height := self.db.GetCurrentBlockHeight()
	if height > self.chainedBlockNum {
		// update chainstore height
		self.chainedBlockNum = height
		// remove persisted pending blocks
		newPending := make(map[uint32]*PendingBlock)
		for blkNum, blk := range self.pendingBlocks {
			if blkNum > height {
				newPending[blkNum] = blk
			}
		}
		// update pending blocks
		self.pendingBlocks = newPending
	}
}

func (self *ChainStore) AddBlock(block *Block) error {
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
	blkNum := self.GetChainedBlockNum() + 1
	for {
		var err error
		if self.needSubmitBlock {
			if submitBlk, present := self.pendingBlocks[blkNum-1]; submitBlk != nil && present {
				err := self.db.SubmitBlock(submitBlk.block.Block, *submitBlk.execResult)
				if err != nil && blkNum > self.GetChainedBlockNum() {
					return fmt.Errorf("ledger add submitBlk (%d, %d) failed: %s", blkNum, self.GetChainedBlockNum(), err)
				}
				if _, present := self.pendingBlocks[blkNum-2]; present {
					delete(self.pendingBlocks, blkNum-2)
				}
			} else {
				break
			}
		}
		execResult, err := self.db.ExecuteBlock(block.Block)
		if err != nil {
			log.Errorf("chainstore AddBlock GetBlockExecResult: %s", err)
			return fmt.Errorf("chainstore AddBlock GetBlockExecResult: %s", err)
		}
		self.pendingBlocks[blkNum] = &PendingBlock{block: block, execResult: &execResult}
		self.needSubmitBlock = true
		self.server.pid.Tell(
			&message.BlockConsensusComplete{
				Block: block.Block,
			})
		self.chainedBlockNum = blkNum
		blkNum++
		break
	}
	return nil
}

func (self *ChainStore) SetBlock(blkNum uint32, blk *PendingBlock) {
	self.pendingBlocks[blkNum] = blk
}

func (self *ChainStore) GetBlock(blockNum uint32) (*Block, error) {
	if blk, present := self.pendingBlocks[blockNum]; present {
		return blk.block, nil
	}
	block, err := self.db.GetBlockByHeight(uint32(blockNum))
	if err != nil {
		return nil, err
	}
	prevmerkleRoot := common.Uint256{}
	if blockNum > 1 {
		prevmerkleRoot, err = self.db.GetStateMerkleRoot(blockNum - 1)
		if err != nil {
			log.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", blockNum, err)
			return nil, fmt.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", blockNum, err)
		}
	}
	return initVbftBlock(block, prevmerkleRoot)
}
