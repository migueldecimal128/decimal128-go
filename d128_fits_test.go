package decimal128

import "testing"

func mustParseFits(t *testing.T, s string) d128 {
	t.Helper()
	x, ok := d128_parse(s)
	if !ok {
		t.Fatalf("d128_parse(%q) failed", s)
	}
	return x
}

func TestD128FitsInt64(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// In range: finite and within [Int64.min, Int64.max].
		{"0", true},
		{"-0", true},
		{"1", true},
		{"-1", true},
		{"3.7", true},
		{"-3.7", true},
		{"1E-10", true},
		{"9223372036854775807", true},  // Int64.max
		{"-9223372036854775808", true}, // Int64.min
		// Out of range: overflow or non-finite.
		{"9223372036854775808", false},   // Int64.max + 1
		{"-9223372036854775809", false},  // Int64.min - 1
		{"9223372036854775807.5", false}, // rounds up past Int64.max
		{"18446744073709551615", false},  // UInt64.max
		{"1E100", false},
		{"NaN", false},
		{"Infinity", false},
		{"-Infinity", false},
	}
	for _, c := range cases {
		x := mustParseFits(t, c.in)
		if got := d128_fitsInt64(x); got != c.want {
			t.Errorf("d128_fitsInt64(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestD128FitsUInt64(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Fit: finite, non-negative (incl. -0), at most UInt64.max.
		{"0", true},
		{"-0", true}, // negative zero MUST fit
		{"1", true},
		{"3.7", true},
		{"9223372036854775808", true},  // > Int64.max but < UInt64.max
		{"18446744073709551615", true}, // UInt64.max
		// Not: negative nonzero, overflow, or non-finite.
		{"-1", false},
		{"-0.3", false},                   // negative nonzero MUST NOT fit
		{"18446744073709551616", false},   // UInt64.max + 1
		{"18446744073709551615.5", false}, // rounds up past UInt64.max
		{"NaN", false},
		{"Infinity", false},
	}
	for _, c := range cases {
		x := mustParseFits(t, c.in)
		if got := d128_fitsUInt64(x); got != c.want {
			t.Errorf("d128_fitsUInt64(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
