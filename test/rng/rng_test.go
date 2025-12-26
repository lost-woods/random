package rng_test

import (
	"encoding/binary"
	"io"
	"math"
	"testing"

	"github.com/lost-woods/random/src/rng"
)

// uint32CounterReader emits an infinite stream of big-endian uint32 values: 0,1,2,3,...
type uint32CounterReader struct {
	next uint32
	buf  [4]byte
	off  int
}

func (r *uint32CounterReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if r.off == 0 {
			binary.BigEndian.PutUint32(r.buf[:], r.next)
			r.next++
		}
		copied := copy(p[n:], r.buf[r.off:])
		n += copied
		r.off = (r.off + copied) % 4
	}
	return n, nil
}

type scriptedReader struct {
	chunks [][]byte
	i      int
	off    int
}

func (r *scriptedReader) Read(p []byte) (int, error) {
	if r.i >= len(r.chunks) {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) {
		if r.i >= len(r.chunks) {
			break
		}
		c := r.chunks[r.i]
		if r.off >= len(c) {
			r.i++
			r.off = 0
			continue
		}
		copied := copy(p[n:], c[r.off:])
		n += copied
		r.off += copied
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestUniformInt32_PerfectUniformWhenRangeDivides2Pow32(t *testing.T) {
	// Range size 256 divides 2^32, so no rejection is needed and distribution is perfect over 65536 draws.
	r := &uint32CounterReader{next: 0}
	counts := make([]int, 256)

	draws := 65536
	for i := 0; i < draws; i++ {
		v, err := rng.UniformInt32(r, nil, 0, 255)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[int(v)]++
	}

	for i := 0; i < 256; i++ {
		if counts[i] != 256 {
			t.Fatalf("value %d count=%d want=256", i, counts[i])
		}
	}
}

func TestUniformInt32_RetriesOnRejectedValues(t *testing.T) {
	// For range size 10: limit = 4294967290, so 0xFFFFFFFA..0xFFFFFFFF are rejected.
	rejected := []byte{0xFF, 0xFF, 0xFF, 0xFA} // 4294967290 (reject)
	accepted := []byte{0x00, 0x00, 0x00, 0x00} // 0 (accept)
	r := &scriptedReader{chunks: [][]byte{rejected, accepted}}

	v, err := rng.UniformInt32(r, nil, 0, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 0 {
		t.Fatalf("got %d want 0", v)
	}
}

func TestUniformInt32_Invariants(t *testing.T) {
	r := &uint32CounterReader{next: 0}
	cases := []struct {
		min int
		max int
	}{
		{0, 0},
		{-5, -5},
		{-10, 10},
		{1, 2},
		{100, 1000},
		{-1000000000, -999999900},
	}

	for _, tc := range cases {
		for i := 0; i < 1000; i++ {
			v, err := rng.UniformInt32(r, nil, tc.min, tc.max)
			if err != nil {
				t.Fatalf("min=%d max=%d unexpected error: %v", tc.min, tc.max, err)
			}
			if v < int32(tc.min) || v > int32(tc.max) {
				t.Fatalf("min=%d max=%d got out-of-range %d", tc.min, tc.max, v)
			}
			if tc.min == tc.max && v != int32(tc.min) {
				t.Fatalf("min=max=%d got %d", tc.min, v)
			}
		}
	}
}

func TestUniformInt32_DistributionSanity_NonDivisorRanges(t *testing.T) {
	// Deterministic sanity test: counts should be close to uniform.
	// This will reliably catch modulo-bias regressions.
	ranges := []int{10, 52, 100, 365}
	draws := 300000

	for _, k := range ranges {
		r := &uint32CounterReader{next: 0}
		counts := make([]int, k)

		for i := 0; i < draws; i++ {
			v, err := rng.UniformInt32(r, nil, 0, k-1)
			if err != nil {
				t.Fatalf("range=%d unexpected error: %v", k, err)
			}
			counts[int(v)]++
		}

		expected := float64(draws) / float64(k)
		tol := expected * 0.015 // 1.5% tolerance (tight but safe)
		for i, c := range counts {
			if abs(float64(c)-expected) > tol {
				t.Fatalf("range=%d value=%d count=%d expected≈%.1f", k, i, c, expected)
			}
		}
	}
}

// Chi-square smoke test (seeded pseudo RNG) to catch gross skews.
// Deterministic seed => non-flaky; threshold is intentionally conservative.
type xorshift32 struct {
	x uint32
}

func (r *xorshift32) Read(p []byte) (int, error) {
	for i := 0; i < len(p); i++ {
		r.x ^= r.x << 13
		r.x ^= r.x >> 17
		r.x ^= r.x << 5
		p[i] = byte(r.x >> 24)
	}
	return len(p), nil
}

func chiSquare(counts []int, expected float64) float64 {
	var chi float64
	for _, c := range counts {
		diff := float64(c) - expected
		chi += diff * diff / expected
	}
	return chi
}

func TestUniformInt32_ChiSquareSmoke(t *testing.T) {
	tests := []struct {
		k      int
		draws  int
		maxChi float64
	}{
		{10, 500000, 60},
		{52, 800000, 140},
	}

	for _, tc := range tests {
		r := &xorshift32{x: 0x12345678}
		counts := make([]int, tc.k)
		for i := 0; i < tc.draws; i++ {
			v, err := rng.UniformInt32(r, nil, 0, tc.k-1)
			if err != nil {
				t.Fatalf("k=%d unexpected error: %v", tc.k, err)
			}
			counts[int(v)]++
		}
		exp := float64(tc.draws) / float64(tc.k)
		chi := chiSquare(counts, exp)
		if math.IsNaN(chi) || math.IsInf(chi, 0) {
			t.Fatalf("k=%d got invalid chi-square", tc.k)
		}
		if chi > tc.maxChi {
			t.Fatalf("k=%d chi-square too large: %.2f > %.2f", tc.k, chi, tc.maxChi)
		}
	}
}
