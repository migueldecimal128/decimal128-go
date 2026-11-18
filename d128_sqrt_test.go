package decimal128

// d128_sqrt — IEEE 754-2019 §5.4.1 squareRoot.
//
// The value/flags coverage — exact perfect squares, the special non-finite /
// negative dispatch, and the correctly-rounded reference table — lives as
// native Rosetta cases (sqrt.txt). What remains here is structural, not an
// (op, operands, result) tuple: the directed-rounding bracket property over a
// large pseudo-random sweep. (root−1ulp)² rounded toward zero and (root+1ulp)²
// rounded toward +∞ must straddle the radicand. Directed rounding only widens
// the bracket, so a correctly-rounded root never false-fails; a root off by
// >1 ulp is still caught.

import (
	"fmt"
	"testing"
)

func TestD128SqrtTestBracketFixed(t *testing.T) {
	for _, s := range []string{
		"1", "2", "3", "4", "5", "7", "11", "1.5", "0.5", "0.25",
		"1E-100", "1E+100", "1E-6000", "1E+6000", "7E-5000", "3E+5000",
		"1234567890123456789012345678901234",
		"9999999999999999999999999999999999",
		"1000000000000000000000000000000000",
		"9.999999999999999999999999999999999E+6144", "1E-6143",
	} {
		d128SqrtBracket(t, s)
	}
}

func TestD128SqrtTestBracketPseudoRandom(t *testing.T) {
	// Deterministic LCG — reproducible, no clock/random dependency.
	state := uint64(0x2545F4914F6CDD1D)
	next := func() uint64 {
		state = state*6364136223846793005 + 1442695040888963407
		return state
	}
	for i := 0; i < 20000; i++ {
		digitLen := int(next()%34) + 1
		sb := []byte{byte('1' + next()%9)} // 1-9
		for j := 1; j < digitLen; j++ {
			sb = append(sb, byte('0'+next()%10)) // 0-9
		}
		exp := int(next()%(6111+6176)) - 6176
		d128SqrtBracket(t, fmt.Sprintf("%sE%d", sb, exp))
	}
}

// d128SqrtBracket: a correctly-rounded sqrt s of r satisfies
// (s−1ulp)² ≤ r ≤ (s+1ulp)². The squares are rounded *outward* (minus toward
// zero, plus toward +∞) so the bracket only ever widens.
func d128SqrtBracket(t *testing.T, s string) {
	t.Helper()
	radicand := mustDec(t, s)
	root := d128_sqrt_tte(radicand)
	minus := d128_nextDown_ctxnull(root, nil)
	minusSquared := d128_multiply_rnd(minus, minus, const_ROUND_TOWARD_ZERO)
	plus := d128_nextUp_ctxnull(root, nil)
	plusSquared := d128_multiply_rnd(plus, plus, const_ROUND_TOWARD_POSITIVE)
	// "≤" is LT or EQ, i.e. compare code ≤ CMP754_EQ (excludes GT, UNORDERED).
	if d128_comparePartial(minusSquared, radicand) > const_CMP754_EQ {
		t.Errorf("sqrt too high for %s: root=%s", s, d128_format(root, const_FORMAT_AUTO))
	}
	if d128_comparePartial(radicand, plusSquared) > const_CMP754_EQ {
		t.Errorf("sqrt too low for %s: root=%s", s, d128_format(root, const_FORMAT_AUTO))
	}
}
