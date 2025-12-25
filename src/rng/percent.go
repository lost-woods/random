package rng

import (
    "errors"
    "strconv"
    "strings"
)

// ParsePercentExact parses percent string EXACTLY into num/den for P(pass)=num/den.
// Accepts 0% (always fail) and 100% (always pass).
// Exact up to 7 decimal places because den=100*10^d must be <= 1,000,000,000.
func ParsePercentExact(percentStr string) (num int, den int, err error) {
    s := strings.TrimSpace(percentStr)
    if s == "" {
        return 0, 0, errors.New("percent is empty")
    }
    if strings.HasPrefix(s, "+") {
        s = strings.TrimPrefix(s, "+")
    }
    if strings.HasPrefix(s, "-") {
        return 0, 0, errors.New("percent must not be negative")
    }

    parts := strings.Split(s, ".")
    if len(parts) > 2 {
        return 0, 0, errors.New("invalid percent format")
    }

    intPart := parts[0]
    fracPart := ""
    if len(parts) == 2 {
        fracPart = parts[1]
    }

    if intPart == "" {
        intPart = "0"
    }
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

    // Remove trailing zeros in fractional part (keeps value exact, reduces denominator)
    fracPart = strings.TrimRight(fracPart, "0")
    decimals := len(fracPart)
    if decimals > 7 {
        return 0, 0, errors.New("too many decimal places; max is 7")
    }

    digits := strings.TrimLeft(intPart+fracPart, "0")
    if digits == "" {
        digits = "0"
    }

    val, convErr := strconv.ParseInt(digits, 10, 64)
    if convErr != nil {
        return 0, 0, errors.New("percent value too large")
    }

    den64 := int64(100)
    for i := 0; i < decimals; i++ {
        den64 *= 10
        if den64 > 1000000000 {
            return 0, 0, errors.New("percent precision too high")
        }
    }

    maxNum := int64(100)
    for i := 0; i < decimals; i++ {
        maxNum *= 10
    }

    if val < 0 {
        return 0, 0, errors.New("percent must not be negative")
    }
    if val > maxNum {
        return 0, 0, errors.New("percent must not exceed 100")
    }

    if val == 0 {
        return 0, 1, nil // always fail
    }
    if val == maxNum {
        return 1, 1, nil // always pass
    }

    return int(val), int(den64), nil
}
