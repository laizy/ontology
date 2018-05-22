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

package types

import (
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
)

// VerifyType of validator
type VerifyType uint8

const (
	Stateless VerifyType = iota
	Stateful  VerifyType = iota
)

// message
type RegisterValidatorReq struct {
	Validator *actor.PID
	Type      VerifyType
	Id        string
}

type UnRegisterValidatorReq struct {
	Id         string
	VerifyType VerifyType
}

type UnRegisterValidatorRsp struct {
	Id         string
	VerifyType VerifyType
}

type VerifyTxReq struct {
	Tx types.Transaction
}

type VerifyTxRsp struct {
	VerifyType VerifyType
	Hash       common.Uint256
	Height     uint32
	ErrCode    errors.ErrCode
}

// Validator wraps validator actor's pid
type Validator interface {
	// Register send a register message to poolId
	Register(poolId *actor.PID)
	// UnRegister send an unregister message to poolId
	UnRegister(poolId *actor.PID)
	// VerifyType returns the type of validator
	VerifyType() VerifyType
}

type ValidatorActor struct {
	Pid *actor.PID
	Id  string
}
