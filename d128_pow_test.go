package decimal128

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
)

// Breadth cross-checks for pow / pown against float64, complementing the exact
// native golden vectors. Arguments are chosen so the result stays comfortably
// inside float64 range (|log10(result)| ≲ 130).

func TestPowVsFloat(t *testing.T) {
	ctx := newContextRound(TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0x90))
	for i := 0; i < 50000; i++ {
		base := math.Pow(10, (rng.Float64()*2-1)*2) * (1 + rng.Float64()*9) // ~[1e-2, 1e3]
		exp := (rng.Float64()*2 - 1) * 40                                   // [-40, 40]
		x := d128FromFloat(base)
		y := d128FromFloat(exp)
		got := d128ToFloat(t, d128_pow_ctx(x, y, ctx))
		checkRelErr(t, "pow", base, got, math.Pow(base, exp))
	}
}

func TestPownVsFloat(t *testing.T) {
	ctx := newContextRound(TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0x9010))
	for i := 0; i < 50000; i++ {
		base := math.Pow(10, (rng.Float64()*2-1)*2) * (1 + rng.Float64()*9)
		if rng.Intn(2) == 1 {
			base = -base
		}
		n := rng.Intn(81) - 40 // [-40, 40]
		x := d128FromFloat(base)
		got := d128ToFloat(t, d128_pown_ctx(x, n, ctx))
		want := math.Pow(base, float64(n))
		checkRelErr(t, "pown("+strconv.Itoa(n)+")", base, got, want)
	}
}
