package decimal128

// Range predicates d128_fitsInt64 / d128_fitsUInt64 (rounding-independent,
// conservative closed-interval semantics). See d128_to_integral.go.

import "testing"

func fitsInt64Check(t *testing.T, f func(d128) bool, want bool, inputs ...string) {
	t.Helper()
	for _, s := range inputs {
		if f(mustDec(t, s)) != want {
			t.Errorf("%s: want %v", s, want)
		}
	}
}

func TestFitsInt64FitsInt64_inRange(t *testing.T) {
	fitsInt64Check(t, d128_fitsInt64, true, "0", "1", "-1",
		"3.7", "-3.7", // range only, not integrality
		"1E-10",                // tiny nonzero
		"9223372036854775807",  // MaxInt64 exactly
		"-9223372036854775808") // MinInt64 exactly
}

func TestFitsInt64FitsInt64_outOfRange(t *testing.T) {
	fitsInt64Check(t, d128_fitsInt64, false,
		"9223372036854775808",   // MaxInt64 + 1
		"-9223372036854775809",  // MinInt64 - 1
		"9223372036854775807.5", // conservative: real value > max
		"18446744073709551615",  // MaxUint64 (> MaxInt64)
		"1E100")
}

func TestFitsInt64FitsInt64_nonFinite(t *testing.T) {
	fitsInt64Check(t, d128_fitsInt64, false, "NaN", "Infinity", "-Infinity")
}

func TestFitsInt64FitsUInt64_inRange(t *testing.T) {
	fitsInt64Check(t, d128_fitsUInt64, true, "0",
		"-0", // negative zero converts to 0
		"1", "3.7",
		"9223372036854775808",  // > MaxInt64, fits uint64
		"18446744073709551615") // MaxUint64 exactly
}

func TestFitsInt64FitsUInt64_rejectsNegativeAndOverflow(t *testing.T) {
	fitsInt64Check(t, d128_fitsUInt64, false, "-1",
		"-0.3",                   // negative nonzero, even though trunc -> 0
		"18446744073709551616",   // MaxUint64 + 1
		"18446744073709551615.5", // conservative
		"NaN", "Infinity")
}
