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

package statestore

import (
	"bytes"
	"fmt"

	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/states"
	"github.com/ontio/ontology/core/store/common"
	"github.com/syndtr/goleveldb/leveldb"
	"strings"
)

type StateBatch struct {
	store  common.PersistStore
	memory map[string]states.StateValue
}

func NewStateStoreBatch(store common.PersistStore) *StateBatch {
	return &StateBatch{
		store:  store,
		memory: make(map[string]states.StateValue),
	}
}

func (self *StateBatch) Find(prefix common.DataEntryPrefix, key []byte) ([]*common.StateItem, error) {
	var states []*common.StateItem
	bp := append([]byte{byte(prefix)}, key...)
	iter := self.store.NewIterator(bp)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		keyV := key[1:]
		if _, ok := self.memory[string(key)]; ok == false {
			value := iter.Value()
			state, err := decodeStateObject(prefix, value)
			if err != nil {
				return nil, err
			}
			states = append(states, &common.StateItem{Key: string(keyV), Value: state})
		}
	}
	kprefix := string(bp)
	for k, v := range self.memory {
		if v != nil && strings.HasPrefix(k, kprefix) {
			states = append(states, &common.StateItem{Key: k, Value: v})
		}
	}

	return states, nil
}

func (self *StateBatch) TryAdd(prefix common.DataEntryPrefix, key []byte, value states.StateValue) {
	k := string(append([]byte{byte(prefix)}, key...))
	self.memory[k] = value
}

func (self *StateBatch) TryGet(prefix common.DataEntryPrefix, key []byte) (states.StateValue, error) {
	k := string(append([]byte{byte(prefix)}, key...))
	if state, ok := self.memory[k]; ok {
		return state, nil
	}

	enc, err := self.store.Get([]byte(k))
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	if enc == nil {
		return nil, nil
	}
	stateVal, err := decodeStateObject(prefix, enc)
	if err != nil {
		return nil, err
	}
	return stateVal, nil
}

func (self *StateBatch) TryDelete(prefix common.DataEntryPrefix, key []byte) {
	k := string(append([]byte{byte(prefix)}, key...))
	self.memory[k] = nil
}

func (self *StateBatch) CommitTo() error {
	for k, v := range self.memory {
		if v == nil {
			self.store.BatchDelete([]byte(k))
		} else {
			data := new(bytes.Buffer)
			err := v.Serialize(data)
			if err != nil {
				return fmt.Errorf("error: key %v, value:%v", k, v)
			}
			self.store.BatchPut([]byte(k), data.Bytes())
		}
	}
	return nil
}

func decodeStateObject(prefix common.DataEntryPrefix, enc []byte) (states.StateValue, error) {
	reader := bytes.NewBuffer(enc)
	switch prefix {
	case common.ST_BOOKKEEPER:
		bookkeeper := new(payload.Bookkeeper)
		if err := bookkeeper.Deserialize(reader); err != nil {
			return nil, err
		}
		return bookkeeper, nil
	case common.ST_CONTRACT:
		contract := new(payload.DeployCode)
		if err := contract.Deserialize(reader); err != nil {
			return nil, err
		}
		return contract, nil
	case common.ST_STORAGE:
		storage := new(states.StorageItem)
		if err := storage.Deserialize(reader); err != nil {
			return nil, err
		}
		return storage, nil
	default:
		panic("[decodeStateObject] invalid state type!")
	}
}
