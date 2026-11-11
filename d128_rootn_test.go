package decimal128

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
)

// Breadth cross-checks for rootn / compound against float64, complementing the
// exact native golden vectors.

func TestRootnVsFloat(t *testing.T) {
	ctx := newContextRound(const_ROUND_TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0x600))
	for i := 0; i < 50000; i++ {
		base := math.Pow(10, (rng.Float64()*2-1)*2) * (1 + rng.Float64()*9) // positive ~[1e-2, 1e3]
		n := rng.Intn(80) - 40
		if n == 0 {
			n = 1
		}
		got := d128ToFloat(t, d128_rootn_ctx(d128FromFloat(base), n, ctx))
		checkRelErr(t, "rootn("+strconv.Itoa(n)+")", base, got, math.Pow(base, 1.0/float64(n)))
	}
}

func TestCompoundVsFloat(t *testing.T) {
	ctx := newContextRound(const_ROUND_TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0xC0))
	for i := 0; i < 50000; i++ {
		x := math.Pow(10, (rng.Float64()*2-1)*2)*(1+rng.Float64()*9) - 1 // 1+x ~[1e-2, 1e3], x > -1
		n := rng.Intn(80) - 40
		got := d128ToFloat(t, d128_compound_ctx(d128FromFloat(x), n, ctx))
		// Looser bound: when 1+x is small the float64 reference loses precision
		// to cancellation, amplified by |n|; the decimal result is more accurate.
		checkRelErrTol(t, "compound("+strconv.Itoa(n)+")", x, got, math.Pow(1+x, float64(n)), 1e-11)
	}
}
