package rng

import (
    "encoding/binary"
    "errors"
    "io"
)

type Reader = io.Reader

func ReadFull(r io.Reader, buf []byte) error {
    _, err := io.ReadFull(r, buf)
    return err
}

// UniformInt32 returns a uniform integer in [min, max] inclusive.
// Integer-only rejection sampling (no floats). This is unbiased assuming the uint32 stream is uniform.
func UniformInt32(r io.Reader, h *Health, min int, max int) (int32, error) {
    // Bounds
    if min < -1000000000 {
        return 0, errors.New("the minimum value should not be lower than -1,000,000,000")
    }
    if min > 1000000000 {
        return 0, errors.New("the minimum value should not be higher than 1,000,000,000")
    }
    if max < -1000000000 {
        return 0, errors.New("the maximum value should not be lower than -1,000,000,000")
    }
    if max > 1000000000 {
        return 0, errors.New("the maximum value should not be higher than 1,000,000,000")
    }
    if min > max {
        return 0, errors.New("the minimum value should be smaller than or equal to the maximum value")
    }

    rangeSize := uint32(max - min + 1)
    if rangeSize == 0 {
        return 0, errors.New("invalid range size")
    }

    // limit = floor(2^32 / rangeSize) * rangeSize
    limit := (uint64(1)<<32)/uint64(rangeSize) * uint64(rangeSize)

    var buf [4]byte
    for {
        if err := ReadFull(r, buf[:]); err != nil {
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

func BuildCharset(lowers, uppers, numbers, symbols bool) []byte {
    var b []byte
    if lowers {
        b = append(b, []byte("abcdefghijklmnopqrstuvwxyz")...)
    }
    if uppers {
        b = append(b, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")...)
    }
    if numbers {
        b = append(b, []byte("0123456789")...)
    }
    if symbols {
        b = append(b, []byte("!#$%&()*+,-./:;<=>?@[]^_{|}~")...)
    }
    return b
}
