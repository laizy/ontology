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

func TestDUPFROMALTSTACK(t *testing.T) {
	code := DUPFROMALTSTACK
	executor := NewExecutor([]byte{byte(code)})
	executor.AltStack.PushInt64(9999)
	executor.EvalStack.PushInt64(8888)
	executor.Execute()
	val,err := executor.EvalStack.Pop()
	assert.Equal(t, err, nil)
	i, err := val.AsInt64()
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(9999),i)
}

func TestTOALTSTACK(t *testing.T){
	code := TOALTSTACK
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushInt64(8888)
	executor.Execute()
	val,err := executor.AltStack.Pop()
	i, err := val.AsInt64()
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(8888),i)
}

func TestFROMALTSTACK(t *testing.T) {
	code := FROMALTSTACK
	executor := NewExecutor([]byte{byte(code)})
	executor.AltStack.PushInt64(8888)
	executor.Execute()
	val,err := executor.EvalStack.Pop()
	i, err := val.AsInt64()
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(8888),i)
}

func TestCAT(t *testing.T) {
	code := CAT
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushBytes([]byte("test"))
	executor.EvalStack.PushBytes([]byte("hello"))
	executor.Execute()

	val,err := executor.EvalStack.Pop()
	bs, err := val.AsBytes()
	assert.Equal(t, err, nil)
	assert.Equal(t, bs, []byte("testhello"))
}

func TestSUBSTR(t *testing.T) {
	code := SUBSTR
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushBytes([]byte("testhello"))
	executor.EvalStack.PushInt64(1)
	executor.EvalStack.PushInt64(3)
	executor.Execute()

	val,err := executor.EvalStack.Pop()
	bs, err := val.AsBytes()
	assert.Equal(t, err, nil)
	bs2 := []byte("testhello")
	assert.Equal(t, bs, bs2[1:4])
}
func TestLEFT(t *testing.T){
	code := LEFT
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushBytes([]byte("testhello"))
	executor.EvalStack.PushInt64(3)
	executor.Execute()

	val,err := executor.EvalStack.Pop()
	bs, err := val.AsBytes()
	assert.Equal(t, err, nil)
	bs2 := []byte("testhello")
	assert.Equal(t, bs, bs2[:3])
}
func TestRIGHT(t *testing.T){
	code := RIGHT
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushBytes([]byte("testhello"))
	executor.EvalStack.PushInt64(3)
	executor.Execute()

	val,err := executor.EvalStack.Pop()
	bs, err := val.AsBytes()
	assert.Equal(t, err, nil)
	bs2 := []byte("testhello")
	assert.Equal(t, bs, bs2[len(bs2)-3:])
}

func TestSIZE(t *testing.T) {
	code := SIZE
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushBytes([]byte("testhello"))
	executor.Execute()

	val,err := executor.EvalStack.Pop()
	i, err := val.AsInt64()
	assert.Equal(t, err, nil)
	bs2 := []byte("testhello")
	assert.Equal(t, i, int64(len(bs2)))
}

func TestWITHIN(t *testing.T) {
	code := WITHIN
	executor := NewExecutor([]byte{byte(code)})
	executor.EvalStack.PushInt64(10000)
	executor.EvalStack.PushInt64(9999)
	executor.EvalStack.PushInt64(8888)
	executor.Execute()

	val,err := executor.EvalStack.Pop()
	i, err := val.AsInt64()
	assert.Equal(t, err, nil)
	assert.Equal(t, i, int64(0))
}
func TestStackOpCode(t *testing.T) {
	checkStackOpCode(t, SWAP, []int{1, 2}, []int{2, 1})
	checkStackOpCode(t, XDROP, []int{3, 2, 1}, []int{2})
	checkStackOpCode(t, XSWAP, []int{3, 2, 1}, []int{2, 3})
	checkStackOpCode(t, XTUCK, []int{2, 1}, []int{2, 2})
	checkStackOpCode(t, DEPTH, []int{1, 2}, []int{1, 2, 2})
	checkStackOpCode(t, DROP, []int{1, 2}, []int{1})
	checkStackOpCode(t, DUP, []int{1, 2}, []int{1, 2 ,2})
	checkStackOpCode(t, NIP, []int{1, 2}, []int{2})
	checkStackOpCode(t, OVER, []int{1, 2}, []int{1, 2, 1})
	checkStackOpCode(t, PICK, []int{3, 2, 1}, []int{3, 2, 3})
	checkStackOpCode(t, ROLL, []int{3, 2, 1}, []int{2, 3})
	checkStackOpCode(t, ROT, []int{3, 1,1 , 1}, []int{1, 1, 1,3})
	checkStackOpCode(t, TUCK, []int{1, 2}, []int{2, 1, 2})


	checkStackOpCode(t, INVERT, []int{2}, []int{-3})
	checkStackOpCode(t, AND, []int{1, 2}, []int{0})
	checkStackOpCode(t, OR, []int{1, 2}, []int{3})
	checkStackOpCode(t, XOR, []int{1, 2}, []int{3})
	checkStackOpCode(t, EQUAL, []int{1, 2}, []int{0})

	checkStackOpCode(t, INC, []int{1}, []int{2})
	checkStackOpCode(t, DEC, []int{2}, []int{1})
	checkStackOpCode(t, SIGN, []int{1}, []int{1})
	checkStackOpCode(t, NEGATE, []int{1}, []int{-1})
	checkStackOpCode(t, ABS, []int{-9999}, []int{9999})
	checkStackOpCode(t, NOT, []int{1}, []int{0})
	checkStackOpCode(t, NZ, []int{1, 2}, []int{1,1})
	checkStackOpCode(t, ADD, []int{1, 2}, []int{3})
	checkStackOpCode(t, SUB, []int{1, 2}, []int{-1})
	checkStackOpCode(t, MUL, []int{1, 2}, []int{2})
	checkStackOpCode(t, DIV, []int{2, 1}, []int{2})
	checkStackOpCode(t, MOD, []int{1, 2}, []int{1})
	//SHL未实现
	//checkStackOpCode(t, SHL, []int{1, 2}, []int{2})
	//checkStackOpCode(t, SHR, []int{1, 2}, []int{2, 1})
	checkStackOpCode(t, BOOLAND, []int{1, 2}, []int{1})
	checkStackOpCode(t, BOOLOR, []int{1, 2}, []int{1})
	checkStackOpCode(t, NUMEQUAL, []int{1, 2}, []int{0})
	checkStackOpCode(t, NUMNOTEQUAL, []int{1, 2}, []int{1})
	checkStackOpCode(t, LT, []int{1, 2}, []int{1})
	checkStackOpCode(t, GT, []int{1, 2}, []int{0})
	checkStackOpCode(t, LTE, []int{1, 2}, []int{1})
	checkStackOpCode(t, GTE, []int{1, 2}, []int{0})
	checkStackOpCode(t, MIN, []int{1, 2}, []int{1})
	checkStackOpCode(t, MAX, []int{1, 2}, []int{2})
	//

}
