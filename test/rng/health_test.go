package rng_test

import (
    "bytes"
    "testing"

    "github.com/lost-woods/random/src/rng"
)

func TestHealthCheckRNG_AllSameFails(t *testing.T) {
    h := rng.NewHealth()
    r := bytes.NewReader(make([]byte, 256))
    if err := rng.HealthCheckRNG(r, h); err == nil {
        t.Fatalf("expected error for all-identical sample")
    }
}

func TestHealthCheckRNG_OKOnVariedBytes(t *testing.T) {
    h := rng.NewHealth()
    buf := make([]byte, 256)
    for i := 0; i < len(buf); i++ {
        buf[i] = byte(i)
    }
    r := bytes.NewReader(buf)
    if err := rng.HealthCheckRNG(r, h); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
