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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runAltStackOpCodeNew(code []byte) (stacks [2]*ValueStack, err error) {
	executor := NewExecutor(code)
	err = executor.Execute()
	if err != nil {
		return
	}

	stacks = [2]*ValueStack{executor.EvalStack, executor.AltStack}
	return
}

func TestRandomOpcode(t *testing.T) {
	const N = 100000
	buf := make([]byte, 10)
	for i := 0; i < N; i++ {
		_, _ = rand.Read(buf)
		for i := 0; i < len(buf); i++ {
			switch OpCode(buf[i]) {
			case APPCALL, SYSCALL, VERIFY:
				buf[i] = 0
			}
		}

		buf, _ = hex.DecodeString("5100730314db0afee719")

		hexCode := fmt.Sprintf("opcode: %x", buf)
		fmt.Println(hexCode)

		old, err1 := runAltStackOpCodeOld(buf)
		stacks, err2 := runAltStackOpCodeNew(buf)
		if err1 != nil || err2 != nil {
			if err1 == nil {
				panic(fmt.Sprintf("err1 == nil: %s", hexCode))
			}
			if err2 == nil {
				panic(fmt.Sprintf("err2 == nil: %s", hexCode))
			}
			continue
		}

		assert.Equal(t, old[0].Count(), stacks[0].Count())
		assert.Equal(t, old[1].Count(), stacks[1].Count())

		for s, stack := range stacks {
			expect := old[s]
			count := expect.Count()
			for i := 0; i < count; i++ {
				val := expect.Pop()
				res, _ := stack.Pop()
				oldv := oldValue2json(t, val)
				newv := value2json(t, res)

				if oldv != newv {
					panic(fmt.Sprintf("value not equal:%s != %s, opcode:%s", oldv, newv, hexCode))
				}
			}
		}

	}
}

func runAltStackOpCodeOld(code []byte) (stacks [2]*RandomAccessStack, err error) {
	executor := NewExecutionEngine()
	context := NewExecutionContext(code)
	executor.PushContext(context)
	err = executor.Execute()
	if err != nil {
		return
	}
	stacks = [2]*RandomAccessStack{executor.EvaluationStack, executor.AltStack}
	return
}
