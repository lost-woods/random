package rng

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type Health struct {
	mu            sync.RWMutex
	ok            bool
	lastErr       string
	lastCheckedAt time.Time
	lastSample32  uint32
	repeatCount32 int
}

func NewHealth() *Health { return &Health{ok: false} }

func (h *Health) Set(ok bool, errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ok = ok
	h.lastErr = errMsg
	h.lastCheckedAt = time.Now()
}

func (h *Health) Snapshot() (ok bool, errMsg string, t time.Time) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ok, h.lastErr, h.lastCheckedAt
}

// HealthCheckRNG performs a lightweight sanity check.
// It cannot prove randomness, but detects disconnection/stuck output/common failures.
func HealthCheckRNG(r io.Reader, h *Health) error {
	const sampleBytes = 256
	buf := make([]byte, sampleBytes)

	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("serial RNG read failed: %w", err)
	}

	// Trivial stuck check: all identical
	allSame := true
	for i := 1; i < len(buf); i++ {
		if buf[i] != buf[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return errors.New("serial RNG appears stuck (all sampled bytes identical)")
	}

	// Excessive 32-bit repeats
	if len(buf) >= 8 {
		var prev uint32
		repeats := 0
		words := 0
		for i := 0; i+4 <= len(buf); i += 4 {
			w := binary.BigEndian.Uint32(buf[i : i+4])
			if words > 0 && w == prev {
				repeats++
			}
			prev = w
			words++
		}
		if words > 1 && repeats > (words-1)*3/4 {
			return errors.New("serial RNG appears stuck (32-bit words repeating excessively)")
		}

		if h != nil {
			h.mu.Lock()
			h.lastSample32 = prev
			h.repeatCount32 = 0
			h.mu.Unlock()
		}
	}

	// Too few distinct byte values
	distinct := make(map[byte]struct{}, 256)
	for _, b := range buf {
		distinct[b] = struct{}{}
	}
	if len(distinct) < 8 {
		return fmt.Errorf("serial RNG sample has too few distinct byte values (%d); suspicious", len(distinct))
	}

	return nil
}

func PeriodicHealthCheck(r io.Reader, h *Health, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	var buf [4]byte
	for range ticker.C {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			h.Set(false, "serial RNG read failed: "+err.Error())
			continue
		}

		w := binary.BigEndian.Uint32(buf[:])

		h.mu.Lock()
		if w == h.lastSample32 {
			h.repeatCount32++
		} else {
			h.repeatCount32 = 0
		}
		h.lastSample32 = w

		// 20 identical 32-bit values in a row is astronomically unlikely for a healthy RNG.
		if h.repeatCount32 >= 20 {
			h.ok = false
			h.lastErr = "serial RNG appears stuck (repeating identical 32-bit outputs)"
			h.lastCheckedAt = time.Now()
			h.mu.Unlock()
			continue
		}

		h.ok = true
		h.lastErr = ""
		h.lastCheckedAt = time.Now()
		h.mu.Unlock()
	}
}
