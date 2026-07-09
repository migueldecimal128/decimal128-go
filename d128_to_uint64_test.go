package decimal128

import "testing"

func mustParseToUInt64(t *testing.T, s string) d128 {
	t.Helper()
	x, ok := d128_parse(s)
	if !ok {
		t.Fatalf("d128_parse(%q) failed", s)
	}
	return x
}

func TestToUInt64(t *testing.T) {
	// VALID: truncate toward zero, no invalidOperation.
	valid := []struct {
		in   string
		want uint64
	}{
		{"0", 0},
		{"-0", 0},
		{"1", 1},
		{"3.7", 3},
		{"-0.3", 0}, // truncates to -0 -> 0
		{"255", 255},
		{"1E3", 1000},
		{"9223372036854775807", 9223372036854775807},   // Int64.max
		{"9223372036854775808", 9223372036854775808},   // Int64.max + 1
		{"10000000000000000000", 10000000000000000000}, // 1e19
		{"18446744073709551615", 18446744073709551615}, // MaxUint64
		{"18446744073709551615.9", 18446744073709551615},
	}
	for _, c := range valid {
		ctx := newContext()
		x := mustParseToUInt64(t, c.in)
		got := d128_toUInt64TowardZero_ctxnull(x, ctx)
		if got != c.want {
			t.Errorf("d128_toUInt64TowardZero_ctxnull(%q) = %d, want %d", c.in, got, c.want)
		}
		if ctx.isSet(INVALID_OPERATION) {
			t.Errorf("d128_toUInt64TowardZero_ctxnull(%q) raised invalidOperation, want none", c.in)
		}
	}

	// INVALID: -> 0 + invalidOperation.
	invalid := []string{
		"-1",
		"-3.7",
		"18446744073709551616", // MaxUint64 + 1
		"1E20",
		"1E100",
		"NaN",
		"Infinity",
		"-Infinity",
	}
	for _, in := range invalid {
		ctx := newContext()
		x := mustParseToUInt64(t, in)
		got := d128_toUInt64TowardZero_ctxnull(x, ctx)
		if got != 0 {
			t.Errorf("d128_toUInt64TowardZero_ctxnull(%q) = %d, want 0", in, got)
		}
		if !ctx.isSet(INVALID_OPERATION) {
			t.Errorf("d128_toUInt64TowardZero_ctxnull(%q) did not raise invalidOperation", in)
		}
	}
}
