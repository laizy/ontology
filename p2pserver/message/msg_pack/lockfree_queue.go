package msgpack

import (
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/ontio/ontology/p2pserver/message/types"
)

type Node struct {
	next unsafe.Pointer
	v    types.Message
}

type MPSCQueue struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

func New() *MPSCQueue {
	stub := unsafe.Pointer(&Node{})
	return &MPSCQueue{head: stub, tail: stub}
}

func (q *MPSCQueue) Push(v types.Message) {
	n := unsafe.Pointer(&Node{v: v})
	prev := atomic.SwapPointer(&q.head, n)
	(*Node)(prev).next = n
}

func (q *MPSCQueue) Pop() types.Message {
	for {
		tail := (*Node)(q.tail)
		if tail.next != nil {
			q.tail = tail.next
			return (*Node)(q.tail).v
		}

		runtime.Gosched()
	}
}
