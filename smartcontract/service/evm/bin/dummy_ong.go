package main

import (
	"math/big"

	"github.com/ontio/ontology/common"
	storage2 "github.com/ontio/ontology/smartcontract/storage"
)

type OngBalanceHandle struct {
	AccountBalance map[common.Address]*big.Int
}

func NewOngBalanceHandle() *OngBalanceHandle {
	return &OngBalanceHandle{
		AccountBalance: make(map[common.Address]*big.Int),
	}
}

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
	self.AccountBalance[addr] = val
	return nil
}

func (self OngBalanceHandle) GetBalance(cache *storage2.CacheDB, addr common.Address) (*big.Int, error) {
	balance := self.AccountBalance[addr]
	if balance == nil {
		return ZERO, nil
	}
	return balance, nil
}
