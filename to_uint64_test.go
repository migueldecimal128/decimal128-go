package decimal128

// d128_toUInt64TowardZero_ctxnull: truncate toward zero to uint64; returns 0 and
// raises invalidOperation out of [0, MaxUint64]. Quiet on a discarded fraction
// (the non-exact toward-zero form, mirroring d128_toInt64TowardZero_ctxnull).
// See d128_to_integral.go.

import "testing"

func TestToUInt64ToUInt64_valid(t *testing.T) {
	for s, want := range map[string]uint64{
		"0": 0, "-0": 0, "1": 1,
		"3.7":                    3, // truncates toward zero
		"-0.3":                   0, // (-1, 0] -> 0
		"255":                    255,
		"1E3":                    1000, // coeff × 10^qExp
		"9223372036854775807":    9223372036854775807,
		"9223372036854775808":    9223372036854775808, // > MaxInt64, valid unsigned
		"10000000000000000000":   10000000000000000000,
		"18446744073709551615":   18446744073709551615, // MaxUint64
		"18446744073709551615.9": 18446744073709551615, // truncates to max
	} {
		if got := d128_toUInt64TowardZero_ctxnull(mustDec(t, s), nil); got != want {
			t.Errorf("%s: got %d, want %d", s, got, want)
		}
	}
}

func TestToUInt64ToUInt64_invalid_returnsZeroAndSignals(t *testing.T) {
	for _, s := range []string{"-1", "-3.7", "18446744073709551616", "1E20", "1E100", "NaN", "Infinity", "-Infinity"} {
		ctx := newContext()
		if d128_toUInt64TowardZero_ctxnull(mustDec(t, s), ctx) != 0 {
			t.Errorf("%s: expected 0 sentinel", s)
		}
		if !ctx.flags.isSet(const_INVALID_OPERATION) {
			t.Errorf("%s: expected invalidOperation", s)
		}
	}
}

func TestToUInt64ToUInt64_inRangeDoesNotSignalInvalid(t *testing.T) {
	ctx := newContext()
	d128_toUInt64TowardZero_ctxnull(mustDec(t, "18446744073709551615"), ctx)
	d128_toUInt64TowardZero_ctxnull(mustDec(t, "3.7"), ctx) // fraction discarded quietly
	if ctx.flags.isSet(const_INVALID_OPERATION) {
		t.Error("invalidOperation raised in range")
	}
}
