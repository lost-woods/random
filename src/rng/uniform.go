package rng

import (
	"encoding/binary"
	"errors"
	"io"
)

// UniformInt32 returns a uniform integer in [min, max] inclusive.
// Integer-only rejection sampling (no floats). This is unbiased assuming the uint32 stream is uniform.
func UniformInt32(r io.Reader, h *Health, min int, max int) (int32, error) {
	// Bounds
	minBound := -1_000_000_000
	maxBound := 1_000_000_000

	if min < minBound || min > maxBound ||
		max < minBound || max > maxBound {
		return 0, errors.New("min and max must be between -1,000,000,000 and 1,000,000,000")
	}

	if min > max {
		return 0, errors.New("min must be less than or equal to max")
	}

	// Range and mod bias elimination
	rangeSize := uint32(max - min + 1)
	if rangeSize == 0 {
		return 0, errors.New("invalid range size")
	}

	// limit = floor(2^32 / rangeSize) * rangeSize
	limit := (uint64(1) << 32) / uint64(rangeSize) * uint64(rangeSize)

	var buf [4]byte
	for {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			if h != nil {
				h.Set(false, "error fetching random bytes: "+err.Error())
			}
			return 0, errors.New("error fetching random bytes")
		}

		x := binary.BigEndian.Uint32(buf[:])
		if uint64(x) < limit {
			return int32(x%rangeSize) + int32(min), nil
		}
		// reject and retry
	}
}
