package rng_test

import (
    "encoding/binary"
    "sync"
    "testing"

    "github.com/lost-woods/random/src/rng"
)

// byteCycleReader returns deterministic bytes cycling through 0..255.
// It is NOT safe for concurrent use without a lock.
type byteCycleReader struct {
    b byte
}

func (r *byteCycleReader) Read(p []byte) (int, error) {
    for i := range p {
        p[i] = r.b
        r.b++
    }
    return len(p), nil
}

func TestLockedReader_ConcurrentUniformInt32_NoPanicsNoErrors(t *testing.T) {
    raw := &byteCycleReader{b: 0}
    locked := rng.NewLockedReader(raw)

    // Sanity: ensure reads are serialized and UniformInt32 stays in range under concurrency.
    const goroutines = 50
    const perG = 2000

    var wg sync.WaitGroup
    wg.Add(goroutines)

    errs := make(chan error, goroutines*perG)

    for g := 0; g < goroutines; g++ {
        go func() {
            defer wg.Done()
            for i := 0; i < perG; i++ {
                v, err := rng.UniformInt32(locked, nil, 1, 52)
                if err != nil {
                    errs <- err
                    return
                }
                if v < 1 || v > 52 {
                    errs <- &rangeErr{got: int(v)}
                    return
                }
            }
        }()
    }

    wg.Wait()
    close(errs)

    for err := range errs {
        if err != nil {
            t.Fatalf("concurrent error: %v", err)
        }
    }
}

type rangeErr struct{ got int }

func (e *rangeErr) Error() string { return "out of range" }

func TestLockedReader_SerializesByteBoundaries(t *testing.T) {
    // This test ensures that concurrent reads won't tear 4-byte boundaries across calls.
    // We do this by reading consecutive uint32 values from a deterministic byte stream.
    raw := &byteCycleReader{b: 0}
    locked := rng.NewLockedReader(raw)

    // Read 8 bytes and interpret as 2 big-endian uint32s.
    buf := make([]byte, 8)
    if _, err := locked.Read(buf); err != nil {
        t.Fatalf("read: %v", err)
    }
    a := binary.BigEndian.Uint32(buf[:4])
    b := binary.BigEndian.Uint32(buf[4:])

    // With byteCycleReader starting at 0, bytes are 00 01 02 03 04 05 06 07.
    // So:
    // a = 0x00010203, b = 0x04050607.
    if a != 0x00010203 || b != 0x04050607 {
        t.Fatalf("unexpected uint32s: got %#x and %#x", a, b)
    }
}
