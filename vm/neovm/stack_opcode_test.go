package neovm

import (
	"testing"

	"github.com/ontio/ontology/vm/neovm/types"
	"github.com/stretchr/testify/assert"
)

func checkStackOpCode(t *testing.T, code OpCode, origin, expected []int) {
	executor := NewExecutor([]byte{byte(code)})
	for _, val := range origin {
		executor.EvalStack.Push(types.VmValueFromInt64(int64(val)))
	}
	err := executor.Execute()
	assert.Nil(t, err)
	stack := executor.EvalStack
	assert.Equal(t, stack.Count(), len(expected))

	for i := 0; i < len(expected); i++ {
		val := expected[len(expected)-i-1]
		res, _ := stack.Pop()
		exp := types.VmValueFromInt64(int64(val))
		assert.True(t, res.Equals(exp))
	}
}

func TestStackOpCode(t *testing.T) {
	checkStackOpCode(t, SWAP, []int{1, 2}, []int{2, 1})
}

func TestArithmetic(t *testing.T) {
	checkStackOpCode(t, ADD, []int{1, 2}, []int{3})
	checkStackOpCode(t, SUB, []int{1, 2}, []int{-1})

	checkStackOpCode(t, MUL, []int{3, 2}, []int{6})

	checkStackOpCode(t, DIV, []int{3, 2}, []int{1})
	checkStackOpCode(t, DIV, []int{103, 2}, []int{51})

	checkStackOpCode(t, MAX, []int{3, 2}, []int{3})
	checkStackOpCode(t, MAX, []int{-3, 2}, []int{2})

	checkStackOpCode(t, MIN, []int{3, 2}, []int{2})
	checkStackOpCode(t, MIN, []int{-3, 2}, []int{-3})

	checkStackOpCode(t, SIGN, []int{3}, []int{1})
	checkStackOpCode(t, SIGN, []int{-3}, []int{-1})
	checkStackOpCode(t, SIGN, []int{0}, []int{0})

	checkStackOpCode(t, INC, []int{-10}, []int{-9})
	checkStackOpCode(t, DEC, []int{-10}, []int{-11})
	checkStackOpCode(t, NEGATE, []int{-10}, []int{10})
	checkStackOpCode(t, ABS, []int{-10}, []int{10})

	checkStackOpCode(t, NOT, []int{1}, []int{0})
	checkStackOpCode(t, NOT, []int{0}, []int{1})

	checkStackOpCode(t, NZ, []int{0}, []int{0})
	checkStackOpCode(t, NZ, []int{-10}, []int{1})
	checkStackOpCode(t, NZ, []int{10}, []int{1})

}