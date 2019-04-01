package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewArray(t *testing.T) {
	a := NewArrayValue()
	for i:=0;i< 1024;i++ {
		v:=VmValueFromInt64(int64(i))
		err := a.Append(v)
		assert.Equal(t, err, nil)
	}
	v:=VmValueFromInt64(int64(1024))
	err := a.Append(v)
	assert.False(t, err == nil)
}

func TestArrayValue_RemoveAt(t *testing.T) {
	a := NewArrayValue()
	for i:=0;i< 10;i++ {
		v:=VmValueFromInt64(int64(i))
		err := a.Append(v)
		assert.Equal(t, err, nil)
	}
	err := a.RemoveAt(-1)
	assert.False(t, err == nil)
	err = a.RemoveAt(10)
	assert.False(t, err == nil)

	assert.Equal(t, a.Len(), int64(10))
	a.RemoveAt(0)
	assert.Equal(t, a.Len(), int64(9))
}
