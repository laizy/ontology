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

package ongx

import (
	"encoding/binary"
	"fmt"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/constants"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

func InitOngx() {
	native.Contracts[utils.XXXXContractAddress] = RegisterOngxContract
}

func RegisterOngxContract(native *native.NativeService) {
	native.Register(ont.TRANSFER_NAME, OngTransfer)
}

func OngTransfer(native *native.NativeService) ([]byte, error) {
	var transfers ont.Transfers
	source := common.NewZeroCopySource(native.Input)
	if err := transfers.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewDetailErr(err, errors.ErrNoCode, "[OngTransfer] Transfers deserialize error!")
	}
	contract := native.ContextRef.CurrentContext().ContractAddress
	for _, v := range transfers.States {
		if v.Value == 0 {
			continue
		}
		if v.Value > constants.ONG_TOTAL_SUPPLY {
			return utils.BYTE_FALSE, fmt.Errorf("transfer ong amount:%d over totalSupply:%d", v.Value, constants.ONG_TOTAL_SUPPLY)
		}

		fromKey := ont.GenBalanceKey(contract, v.From)
		if v.From == common.ADDRESS_EMPTY {
			var buf [9]byte
			binary.LittleEndian.PutUint64(buf[1:], v.Value)
			native.CacheDB.Put(fromKey, buf[:])
		} else if !native.ContextRef.CheckWitness(v.From) {
			return utils.BYTE_FALSE, errors.NewErr("authentication failed!")
		}

		var fromBalance uint64
		b, err := native.CacheDB.Get(fromKey)
		if err != nil {
			return utils.BYTE_FALSE, errors.NewErr("authentication failed!")
		}
		if b == nil {
			fromBalance = 0
		} else {
			fromBalance = binary.LittleEndian.Uint64(b[1:])
		}
		if fromBalance < v.Value {
			return utils.BYTE_FALSE, errors.NewErr("authentication failed!")
		}
		fromBalance -= v.Value

		toKey := ont.GenBalanceKey(contract, v.To)
		var toBalance uint64
		b, err = native.CacheDB.Get(toKey)
		if err != nil {
			return utils.BYTE_FALSE, errors.NewErr("authentication failed!")
		}
		if b == nil {
			toBalance = 0
		} else {
			toBalance = binary.LittleEndian.Uint64(b[1:])
		}
		toBalance += v.Value

		var buf [9]byte
		binary.LittleEndian.PutUint64(buf[1:], fromBalance)
		native.CacheDB.Put(fromKey, buf[:])
		binary.LittleEndian.PutUint64(buf[1:], toBalance)
		native.CacheDB.Put(toKey, buf[:])
	}
	return utils.BYTE_TRUE, nil
}

func OngBalanceOf(native *native.NativeService) ([]byte, error) {
	return ont.GetBalanceValue(native, ont.TRANSFER_FLAG)
}
