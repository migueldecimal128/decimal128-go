package decimal128

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

// sameValue reports whether two finite d128 are numerically equal (same value,
// possibly different cohort representative).
func sameValue(a, b d128) bool {
	if d128_signFlag(a) != d128_signFlag(b) {
		return false
	}
	ca := bigU128(d128_u128(a))
	cb := bigU128(d128_u128(b))
	qa := d128_qExp(a)
	qb := d128_qExp(b)
	minq := min(qa, qb)
	va := new(big.Int).Mul(ca, pow10big(qa-minq))
	vb := new(big.Int).Mul(cb, pow10big(qb-minq))
	return va.Cmp(vb) == 0
}

// refCoeffQExp is the exact coefficient×10^qExp rendering of a finite d128.
func refCoeffQExp(x d128) string {
	s := ""
	if d128_signFlag(x) {
		s = "-"
	}
	s += bigU128(d128_u128(x)).String()
	s += "E"
	q := d128_qExp(x)
	if q < 0 {
		s += "-"
		q = -q
	} else {
		s += "+"
	}
	return s + strconv.Itoa(q)
}

func TestPrintCoefficientQExp(t *testing.T) {
	r := rand.New(rand.NewSource(110))
	for i := 0; i < oracleIters; i++ {
		x := randD128(r)
		if !d128_isFIN(x) {
			continue
		}
		if got, want := d128_format(x, const_FORMAT_COEFFICIENT_QEXP), refCoeffQExp(x); got != want {
			t.Fatalf("coeffQExp: got %q want %q", got, want)
		}
	}
}

func TestParsePrintRoundTrip(t *testing.T) {
	styles := []int{const_FORMAT_AUTO, const_FORMAT_EXPONENTIAL, const_FORMAT_ENGINEERING, const_FORMAT_COEFFICIENT_QEXP}
	r := rand.New(rand.NewSource(111))
	for i := 0; i < oracleIters; i++ {
		x := randD128(r)
		for _, style := range styles {
			s := d128_format(x, style)
			got, ok := d128_parse(s)
			if !ok {
				t.Fatalf("parse(%q) failed (style %d, class %d)", s, style, d128_ieeeClass(x))
			}
			// Non-finite and the cohort-preserving coefficient form round-trip
			// bitwise; AUTO/EXP/ENG preserve the numeric value (ENG may pad with
			// trailing zeros into a different cohort representative).
			var equal bool
			if !d128_isFIN(x) || style == const_FORMAT_COEFFICIENT_QEXP {
				equal = d128_bitwiseEQ(got, x)
			} else {
				equal = d128_isFIN(got) && sameValue(got, x)
			}
			if !equal {
				t.Fatalf("round-trip mismatch via %q (style %d): hi=%#x lo=%#x -> hi=%#x lo=%#x",
					s, style, x.ubdHi64, x.ubdLo64, got.ubdHi64, got.ubdLo64)
			}
		}
	}
}

func TestPrintKnown(t *testing.T) {
	cases := []struct {
		x     d128
		style int
		want  string
	}{
		{decimalFNZ_64x64(false, 0, 1, 0), const_FORMAT_AUTO, "1"},
		{decimalFNZ_64x64(true, 0, 1, 0), const_FORMAT_AUTO, "-1"},
		{decimalFNZ_64x64(false, 0, 123, 0), const_FORMAT_AUTO, "123"},
		{decimalFNZ_64x64(false, -2, 1, 0), const_FORMAT_AUTO, "0.01"},
		{decimalFNZ_64x64(false, -1, 15, 0), const_FORMAT_AUTO, "1.5"},
		{decimalFNZ_64x64(false, 5, 1, 0), const_FORMAT_AUTO, "1E+5"},
		{decimalFNZ_64x64(false, -8, 1, 0), const_FORMAT_AUTO, "1E-8"},
		{d128_zero(false, 0), const_FORMAT_AUTO, "0"},
		{d128_zero(true, 0), const_FORMAT_AUTO, "-0"},
		{d128_infinity(false), const_FORMAT_AUTO, "Infinity"},
		{d128_infinity(true), const_FORMAT_AUTO, "-Infinity"},
		{const_D128_QNAN0, const_FORMAT_AUTO, "NaN"},
		{const_D128_SNAN0, const_FORMAT_AUTO, "sNaN"},
		{decimalFNZ_64x64(false, 0, 1, 0), const_FORMAT_EXPONENTIAL, "1E+0"},
		{decimalFNZ_64x64(false, 0, 1, 0), const_FORMAT_COEFFICIENT_QEXP, "1E+0"},
	}
	for _, c := range cases {
		if got := d128_format(c.x, c.style); got != c.want {
			t.Errorf("format(class %d, style %d) = %q, want %q", d128_ieeeClass(c.x), c.style, got, c.want)
		}
	}
}

// TestParseResidual — d128_parse (IEEE 754-2019 §5.4.2 convertFromDecimalCharacter,
// strict form). The (input, result) coverage — cohort-preserving finite parses,
// underscore separators, every infinity/NaN spelling and payload, and the reject
// set — now lives as native Rosetta cases in resources/rosetta/native/parse.txt
// (run through both the core and the public-API harness). What remains here is the
// handful of reject cases the space-delimited native grammar cannot carry: an
// operand that IS empty or contains whitespace can't be written as a single native
// token. These pin that the strict parser rejects them (ok=false, never NaN-on-error).
func TestParseResidual(t *testing.T) {
	for _, bad := range []string{
		"",      // empty string
		"1 ",    // trailing space after a valid number
		"nan 1", // space between the NaN word and its payload
	} {
		if _, ok := d128_parse(bad); ok {
			t.Errorf("parse(%q) should have failed (empty/whitespace)", bad)
		}
	}
}
