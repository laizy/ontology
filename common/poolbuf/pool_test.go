package poolbuf

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffers(t *testing.T) {
	const dummyData = "dummy data"
	p := NewPool()

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				buf := p.Get()
				assert.Zero(t, buf.Len(), "Expected truncated buffer")

				buf.WriteString(dummyData)

				assert.Equal(t, buf.Len(), len(dummyData), "Expected buffer to contain dummy data")

				buf.Free()
			}
			wg.Done()
		}()
	}
	wg.Wait()

}

func BenchmarkNewPool(b *testing.B) {
	const dummyData = "dummy dataaaaaaaaaaaaaaaaaaaaaaaaa"
	p := NewPool()

	for i := 0; i < b.N; i++ {
		buf := p.Get()
		buf.WriteString(dummyData)
		buf.Free()
	}
}
