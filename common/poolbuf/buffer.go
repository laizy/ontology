package poolbuf

import (
	"bytes"
)

// PoolBuf is a thin wrapper around a byte slice. It's intended to be pooled, so
// the only way to construct one is via a Pool.
type PoolBuf struct {
	bytes.Buffer
	pool Pool
}

// Free returns the PoolBuf to its Pool.
//
// Callers must not retain references to the PoolBuf after calling Free.
func (b *PoolBuf) Free() {
	b.pool.put(b)
}
