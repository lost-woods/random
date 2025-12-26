package rng

import (
	"io"
	"sync"
)

// LockedReader wraps an io.Reader and serializes Read calls with a mutex.
// This is critical for fairness and correctness when a single entropy source
// is shared across concurrent HTTP requests (and background health checks).
type LockedReader struct {
	r  io.Reader
	mu sync.Mutex
}

func (lr *LockedReader) Read(p []byte) (int, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.r.Read(p)
}

// NewLockedReader returns a io.Reader that is safe for concurrent use.
// If r is already a *LockedReader, it is returned as-is.
func NewLockedReader(r io.Reader) io.Reader {
	if r == nil {
		return nil
	}
	if _, ok := r.(*LockedReader); ok {
		return r
	}
	return &LockedReader{r: r}
}
