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
