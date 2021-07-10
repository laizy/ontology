package main

import (
	"bytes"
	"math/big"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/serialization"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
	storage2 "github.com/ontio/ontology/smartcontract/storage"
)

type OngBalanceHandle struct{}

var ZERO = new(big.Int).SetUint64(0)

func (self OngBalanceHandle) SubBalance(cache *storage2.CacheDB, addr common.Address, val *big.Int) error {
	balance, err := self.GetBalance(cache, addr)
	if err != nil {
		return err
	}

	balance.Sub(balance, val)
	return self.SetBalance(cache, addr, balance)
}

func (self OngBalanceHandle) AddBalance(cache *storage2.CacheDB, addr common.Address, val *big.Int) error {
	balance, err := self.GetBalance(cache, addr)
	if err != nil {
		return err
	}

	balance.Add(balance, val)
	return self.SetBalance(cache, addr, balance)
}

func (self OngBalanceHandle) SetBalance(cache *storage2.CacheDB, addr common.Address, val *big.Int) error {
	balanceKey := ont.GenBalanceKey(utils.OngContractAddress, addr)
	result := val.Bytes()
	if ZERO.Cmp(val) == 0 {
		cache.Delete(balanceKey)
	} else {
		cache.Put(balanceKey, utils.GenVarBytesStorageItem(result).ToArray())
	}

	return nil
}

func (self OngBalanceHandle) GetBalance(cache *storage2.CacheDB, addr common.Address) (*big.Int, error) {
	balanceKey := ont.GenBalanceKey(utils.OngContractAddress, addr)
	item, err := utils.GetStorageItem(cache, balanceKey)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return ZERO, nil
	}
	v, err := serialization.ReadVarBytes(bytes.NewBuffer(item.Value))
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetBytes(v), nil
}
