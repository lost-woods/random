package rng_test

import (
	"testing"

	"github.com/lost-woods/random/src/rng"
)

func TestParsePercentExact(t *testing.T) {
	tests := []struct {
		in      string
		wantNum int
		wantDen int
		wantErr bool
	}{
		{"0", 0, 1, false},
		{"0.0", 0, 1, false},
		{"100", 1, 1, false},
		{"100.0000000", 1, 1, false},

		{"25", 25, 100, false},
		{"25.5", 255, 1000, false},
		{"1.23456789", 0, 0, true}, // >7 decimals

		{"-1", 0, 0, true},
		{"100.0000001", 0, 0, true},
		{"101", 0, 0, true},
		{"", 0, 0, true},
		{"abc", 0, 0, true},
		{"1..2", 0, 0, true},
		{".5", 5, 1000, false},
		{"000.5000", 5, 1000, false},
		{"+12.34", 1234, 10000, false},
	}

	for _, tc := range tests {
		gotNum, gotDen, err := rng.ParsePercentExact(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("input=%q expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("input=%q unexpected error: %v", tc.in, err)
		}
		if gotNum != tc.wantNum || gotDen != tc.wantDen {
			t.Fatalf("input=%q got %d/%d want %d/%d", tc.in, gotNum, gotDen, tc.wantNum, tc.wantDen)
		}
	}
}
