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

package neovm

import (
	"testing"

	vtypes "github.com/ontio/ontology/vm/neovm/types"
)

func TestOpRemove(t *testing.T) {
	var e1 ExecutionEngine
	e1.EvaluationStack = NewRandAccessStack()

	m1 := vtypes.NewMap()

	m1.Add(vtypes.NewByteArray([]byte("aaa")), vtypes.NewByteArray([]byte("aaa")))
	m1.Add(vtypes.NewByteArray([]byte("bbb")), vtypes.NewByteArray([]byte("bbb")))
	m1.Add(vtypes.NewByteArray([]byte("ccc")), vtypes.NewByteArray([]byte("ccc")))

	PushData(&e1, m1)
	opDup(&e1)
	PushData(&e1, vtypes.NewByteArray([]byte("aaa")))
	opRemove(&e1)

	mm := e1.EvaluationStack.Peek(0)

	v := mm.(*vtypes.Map).TryGetValue(vtypes.NewByteArray([]byte("aaa")))

	if v != nil {
		t.Fatal("NeoVM OpRemove remove map failed.")
	}
}

func TestStruct_Clone(t *testing.T) {
	var e1 ExecutionEngine
	e1.EvaluationStack = NewRandAccessStack()
	a := vtypes.NewStruct(nil)
	for i := 0; i < 1024; i++ {
		a.Add(vtypes.NewStruct(nil))
	}
	b := vtypes.NewStruct(nil)
	for i := 0; i < 1024; i++ {
		b.Add(a)
	}
	PushData(&e1, b)
	for i := 0; i < 1024; i++ {
		opDup(&e1)
		opDup(&e1)
		opAppend(&e1)
	}

}
