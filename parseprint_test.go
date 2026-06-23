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
		if got, want := d128_format(x, FORMAT_COEFFICIENT_QEXP), refCoeffQExp(x); got != want {
			t.Fatalf("coeffQExp: got %q want %q", got, want)
		}
	}
}

func TestParsePrintRoundTrip(t *testing.T) {
	styles := []int{FORMAT_AUTO, FORMAT_EXPONENTIAL, FORMAT_ENGINEERING, FORMAT_COEFFICIENT_QEXP}
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
			if !d128_isFIN(x) || style == FORMAT_COEFFICIENT_QEXP {
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
		{decimalFNZ_64x64(false, 0, 1, 0), FORMAT_AUTO, "1"},
		{decimalFNZ_64x64(true, 0, 1, 0), FORMAT_AUTO, "-1"},
		{decimalFNZ_64x64(false, 0, 123, 0), FORMAT_AUTO, "123"},
		{decimalFNZ_64x64(false, -2, 1, 0), FORMAT_AUTO, "0.01"},
		{decimalFNZ_64x64(false, -1, 15, 0), FORMAT_AUTO, "1.5"},
		{decimalFNZ_64x64(false, 5, 1, 0), FORMAT_AUTO, "1E+5"},
		{decimalFNZ_64x64(false, -8, 1, 0), FORMAT_AUTO, "1E-8"},
		{d128_zero(false, 0), FORMAT_AUTO, "0"},
		{d128_zero(true, 0), FORMAT_AUTO, "-0"},
		{d128_infinity(false), FORMAT_AUTO, "Infinity"},
		{d128_infinity(true), FORMAT_AUTO, "-Infinity"},
		{D128_QNAN0, FORMAT_AUTO, "NaN"},
		{D128_SNAN0, FORMAT_AUTO, "sNaN"},
		{decimalFNZ_64x64(false, 0, 1, 0), FORMAT_EXPONENTIAL, "1E+0"},
		{decimalFNZ_64x64(false, 0, 1, 0), FORMAT_COEFFICIENT_QEXP, "1E+0"},
	}
	for _, c := range cases {
		if got := d128_format(c.x, c.style); got != c.want {
			t.Errorf("format(class %d, style %d) = %q, want %q", d128_ieeeClass(c.x), c.style, got, c.want)
		}
	}
}

func TestParseKnown(t *testing.T) {
	// Valid forms: check the value via the COEFFICIENT_QEXP render.
	valid := []struct {
		in   string
		want string // expected coeffQExp render of the parsed value
	}{
		{"1", "1E+0"},
		{"-1", "-1E+0"},
		{"1.5", "15E-1"},
		{"1E3", "1E+3"},
		{"1.23E-4", "123E-6"},
		{"000123", "123E+0"},
		{"0", "0E+0"},
		{"1_000", "1000E+0"},
	}
	for _, c := range valid {
		x, ok := d128_parse(c.in)
		if !ok {
			t.Fatalf("parse(%q) failed", c.in)
		}
		if got := d128_format(x, FORMAT_COEFFICIENT_QEXP); got != c.want {
			t.Errorf("parse(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// Special values.
	if x, ok := d128_parse("Infinity"); !ok || !d128_isINF(x) || d128_isNegative(x) {
		t.Errorf("parse Infinity wrong")
	}
	if x, ok := d128_parse("-inf"); !ok || !d128_isINF(x) || !d128_isNegative(x) {
		t.Errorf("parse -inf wrong")
	}
	if x, ok := d128_parse("NaN"); !ok || !d128_isQNAN(x) {
		t.Errorf("parse NaN wrong")
	}
	if x, ok := d128_parse("sNaN"); !ok || !d128_isSNAN(x) {
		t.Errorf("parse sNaN wrong")
	}

	// Malformed.
	// Note: "1_" (trailing separator after a digit) is accepted by this parser.
	for _, bad := range []string{"", "xyz", "1.2.3", "1E", "--1", "_1", "1.2.3E4"} {
		if _, ok := d128_parse(bad); ok {
			t.Errorf("parse(%q) should have failed", bad)
		}
	}
}
