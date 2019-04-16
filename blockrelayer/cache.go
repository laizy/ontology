package blockrelayer

import (
	"github.com/hashicorp/golang-lru"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
)

const (
	BLOCK_CAHE_SIZE   = 100   //Block cache size
	HEADER_CHACE_SIZE = 10000 //Transaction cache size
)

type DataCache struct {
	blockCache  *lru.ARCCache
	headerCache *lru.ARCCache
}

func NewDataCache() *DataCache {
	blockCache, _ := lru.NewARC(BLOCK_CAHE_SIZE)
	headerCache, _ := lru.NewARC(HEADER_CHACE_SIZE)
	return &DataCache{
		blockCache:  blockCache,
		headerCache: headerCache,
	}
}

func (self *DataCache) AddBlock(block *RawBlock) {
	blockHash := block.Hash
	self.blockCache.Add(blockHash, block)
}

func (this *DataCache) GetBlock(blockHash common.Uint256) *RawBlock {
	block, ok := this.blockCache.Get(blockHash)
	if !ok {
		return nil
	}
	return block.(*RawBlock)
}

func (this *DataCache) AddHeader(header *types.RawHeader) {
	this.headerCache.Add(header.Hash(), header)
}

func (this *DataCache) GetHeader(hash common.Uint256) *types.RawHeader {
	header, ok := this.headerCache.Get(hash)
	if !ok {
		return nil
	}
	return header.(*types.RawHeader)
}
