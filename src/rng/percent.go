package rng

import (
	"errors"
	"strconv"
	"strings"
)

// ParsePercentExact parses a percentage into an exact probability num/den.
//
// Accepts:
//
//	"25", "25%", "25.432", "25.432%"
//	Leading/trailing spaces and an optional leading '+' are allowed.
//
// Rejects negatives and values > 100.
// Exact up to 7 decimal places (den = 100 * 10^d <= 1,000,000,000).
func ParsePercentExact(percentStr string) (num int, den int, err error) {
	s := strings.TrimSpace(percentStr)

	// Optional trailing '%'
	if v, ok := strings.CutSuffix(s, "%"); ok {
		s = strings.TrimSpace(v)
	}
	if s == "" {
		return 0, 0, errors.New("percent is empty")
	}

	// Optional leading '+', but no negatives
	s = strings.TrimPrefix(s, "+")
	if strings.HasPrefix(s, "-") {
		return 0, 0, errors.New("percent must not be negative")
	}

	// Allow at most one decimal point
	if strings.Count(s, ".") > 1 {
		return 0, 0, errors.New("invalid percent format")
	}
	intPart, fracPart, _ := strings.Cut(s, ".")

	if intPart == "" {
		intPart = "0"
	}

	// digits-only validation
	for _, ch := range intPart {
		if ch < '0' || ch > '9' {
			return 0, 0, errors.New("invalid percent format")
		}
	}
	for _, ch := range fracPart {
		if ch < '0' || ch > '9' {
			return 0, 0, errors.New("invalid percent format")
		}
	}

	// Remove trailing zeros in fractional part (exact, smaller denominator)
	fracPart = strings.TrimRight(fracPart, "0")
	decimals := len(fracPart)
	if decimals > 7 {
		return 0, 0, errors.New("too many decimal places; max is 7")
	}

	// Combine digits; keep at least "0"
	digits := strings.TrimLeft(intPart+fracPart, "0")
	if digits == "" {
		digits = "0"
	}

	target, convErr := strconv.ParseInt(digits, 10, 64)
	if convErr != nil {
		return 0, 0, errors.New("percent value too large")
	}

	// scale = 100 * 10^decimals
	// (used for both denominator and 100% bound)
	scale := int64(100)
	for i := 0; i < decimals; i++ {
		scale *= 10
		if scale > 1_000_000_000 {
			return 0, 0, errors.New("percent precision too high")
		}
	}
	denominator := scale
	maxNum := scale

	if target > maxNum {
		return 0, 0, errors.New("percent must not exceed 100")
	}

	if target == 0 {
		return 0, 1, nil // always fail
	}
	if target == maxNum {
		return 1, 1, nil // always pass
	}

	return int(target), int(denominator), nil
}
