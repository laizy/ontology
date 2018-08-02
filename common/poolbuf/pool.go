package poolbuf

import "sync"

var (
	_pool = NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	Get = _pool.Get
)

// A Pool is a type-safe wrapper around a sync.Pool.
type Pool struct {
	p *sync.Pool
}

// NewPool constructs a new Pool.
func NewPool() Pool {
	return Pool{p: &sync.Pool{
		New: func() interface{} {
			return &PoolBuf{}
		},
	}}
}

// Get retrieves a PoolBuf from the pool, creating one if necessary.
func (p Pool) Get() *PoolBuf {
	buf := p.p.Get().(*PoolBuf)
	buf.Reset()
	buf.pool = p
	return buf
}

func (p Pool) put(buf *PoolBuf) {
	p.p.Put(buf)
}
