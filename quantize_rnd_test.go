package decimal128

// The quantize _rnd rung, through the public wrapper: Rounding.Quantize carries
// a rounding direction with NO flag sink, and agrees with the Context.Quantize
// (flag-bearing) rung fed the same direction.

import "testing"

func TestQuantizeRndRoundingQuantizeHonorsDirection(t *testing.T) {
	x, q := MustParse("2.175"), MustParse("0.01")
	// toward-zero truncates, ties-even rounds — direction is load-bearing.
	if !TowardZero.Quantize(x, q).Equal(MustParse("2.17")) || !TiesToEven.Quantize(x, q).Equal(MustParse("2.18")) ||
		!x.Quantize(q).Equal(MustParse("2.18")) { // bare method is the ties-even form
		t.Error("quantize direction")
	}
}

func TestQuantizeRndRndAgreesWithCtx(t *testing.T) {
	x, q := MustParse("2.175"), MustParse("0.01")
	for _, r := range []Rounding{TowardZero, TiesToEven, TowardPositive, TowardNegative, TiesToAway} {
		if !r.Quantize(x, q).Equal(NewContextWithRounding(r).Quantize(x, q)) {
			t.Errorf("quantize _rnd vs _ctx disagree for %v", r)
		}
	}
}
