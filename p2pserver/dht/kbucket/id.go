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

package kbucket

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/big"
	"math/bits"

	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
)

var Difficulty = 18 //bit

type KadId struct {
	val common.Address
}

func (self KadId) Serialization(sink *common.ZeroCopySink) {
	sink.WriteAddress(self.val)
}

func (self KadId) Deserialization(source *common.ZeroCopySource) error {
	val, eof := source.NextAddress()
	if eof {
		return io.ErrUnexpectedEOF
	}
	self.val = val
	return nil
}

func (self *KadId) ToHexString() string {
	return self.val.ToHexString()
}

type KadKeyId struct {
	PublicKey keypair.PublicKey

	Id KadId
}

func (self KadId) GenRandKadId(prefix uint) KadId {
	var kad KadId
	if prefix > uint(len(self.val[:])) {
		prefix = uint(len(self.val[:]))
	}
	_, _ = rand.Read(kad.val[:])
	copy(kad.val[:prefix], self.val[:prefix])
	return kad
}

func (self KadId) ToUint64() uint64 {
	if self.isPseudoKadId() {
		nonce := binary.LittleEndian.Uint64(self.val[:8])
		return nonce
	}
	kid := new(big.Int).SetBytes(self.val[:])
	uint64Max := new(big.Int).SetUint64(math.MaxUint64)
	res := kid.Mod(kid, uint64Max)
	return res.Uint64()
}

func (self KadId) isPseudoKadId() bool {
	for i := 8; i < len(self.val); i++ {
		if self.val[i] != 0 {
			return false
		}
	}
	return true
}

func PseudoKadIdFromUint64(data uint64) KadId {
	id := common.ADDRESS_EMPTY
	binary.LittleEndian.PutUint64(id[:], data)
	return KadId{
		val: id,
	}
}

func (this *KadKeyId) Serialization(sink *common.ZeroCopySink) {
	sink.WriteVarBytes(keypair.SerializePublicKey(this.PublicKey))
	this.Id.Serialization(sink)
}

func (this *KadKeyId) Deserialization(source *common.ZeroCopySource) error {
	data, _, irregular, eof := source.NextVarBytes()
	if irregular {
		return common.ErrIrregularData
	}
	if eof {
		return io.ErrUnexpectedEOF
	}
	pub, err := keypair.DeserializePublicKey(data)
	if err != nil {
		return err
	}
	if !validatePublicKey(pub) {
		return errors.New("invalid kad public key")
	}
	this.PublicKey = pub
	id := KadId{}
	err = id.Deserialization(source)
	if err != nil {
		return err
	}
	this.Id = id
	return nil
}

func kadIdFromPubkey(pubKey keypair.PublicKey) KadId {
	return KadId{val: types.AddressFromPubKey(pubKey)}
}

func RandKadKeyId() *KadKeyId {
	var acc *account.Account
	for {
		acc = account.NewAccount("")
		if validatePublicKey(acc.PublicKey) {
			break
		}
	}
	kid := kadIdFromPubkey(acc.PublicKey)
	return &KadKeyId{
		PublicKey: acc.PublicKey,
		Id:        kid,
	}
}

func validatePublicKey(pubKey keypair.PublicKey) bool {
	pub := keypair.SerializePublicKey(pubKey)
	res := sha256.Sum256(pub)
	hash := sha256.Sum256(res[:])
	limit := Difficulty >> 3
	for i := 0; i < limit; i++ {
		if hash[i] != 0 {
			return false
		}
	}
	diff := Difficulty - limit*8
	if diff != 0 {
		x := hash[limit] >> uint8(8-diff)
		if x != 0 {
			return false
		}
	}
	return true
}

func (self KadId) distance(b KadId) KadId {
	var c KadId
	for i := 0; i < len(self.val); i++ {
		c.val[i] = self.val[i] ^ b.val[i]
	}

	return c
}

// Closer returns true if a is closer to self than b is
func (self KadId) Closer(a, b KadId) bool {
	adist := self.distance(a)
	bdist := self.distance(b)

	return bytes.Compare(adist.val[:], bdist.val[:]) < 0
}

// CommonPrefixLen(cpl) calculate two ID's xor prefix 0
func CommonPrefixLen(a, b KadId) int {
	dis := a.distance(b)
	return zeroPrefixLen(dis.val[:])
}

// ZeroPrefixLen returns the number of consecutive zeroes in a byte slice.
func zeroPrefixLen(id []byte) int {
	for i, b := range id {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}

	return len(id) * 8
}
