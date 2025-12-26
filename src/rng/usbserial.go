package rng

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/tarm/serial"
)

// NewSerialRNGFromEnv opens a serial port from env vars and performs an initial health check.
// Required env vars:
// - SERIAL_DEVICE_NAME (e.g. /dev/ttyACM0 or COM3)
// - SERIAL_BAUD_RATE
// - SERIAL_READ_TIMEOUT (milliseconds)
func NewSerialRNGFromEnv() (io.Reader, *Health, error) {
	name := os.Getenv("SERIAL_DEVICE_NAME")
	if name == "" {
		return nil, nil, errors.New("SERIAL_DEVICE_NAME is required")
	}

	baudStr := os.Getenv("SERIAL_BAUD_RATE")
	baud, err := strconv.Atoi(baudStr)
	if err != nil || baud <= 0 {
		return nil, nil, fmt.Errorf("invalid SERIAL_BAUD_RATE: %q", baudStr)
	}

	timeoutStr := os.Getenv("SERIAL_READ_TIMEOUT")
	timeoutMs, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutMs < 0 {
		return nil, nil, fmt.Errorf("invalid SERIAL_READ_TIMEOUT: %q", timeoutStr)
	}

	cfg := &serial.Config{
		Name:        name,
		Baud:        baud,
		Size:        8,
		ReadTimeout: time.Duration(timeoutMs) * time.Millisecond,
	}

	p, err := serial.OpenPort(cfg)
	if err != nil {
		return nil, nil, err
	}

	h := NewHealth()
	if err := HealthCheckRNG(p, h); err != nil {
		h.Set(false, err.Error())
		return nil, h, err
	}
	h.Set(true, "")

	return p, h, nil
}
