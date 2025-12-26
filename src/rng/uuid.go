package rng

import (
	"encoding/hex"
	"fmt"
	"io"
)

// NewUUIDv4FromRNG generates an RFC4122 UUID v4 using the same RNG stream.
// Generated ONLY after a successful outcome is computed (so it doesn't bias outcomes).
func NewUUIDv4FromRNG(r io.Reader) (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return "", err
	}

	// Set version (4) and variant (10xx)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	hex32 := make([]byte, 32)
	hex.Encode(hex32, b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex32[0:8], hex32[8:12], hex32[12:16], hex32[16:20], hex32[20:32],
	), nil
}
