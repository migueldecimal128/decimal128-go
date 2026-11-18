package decimal128

// Vector test for divPow10_q128res_256_recipmul: quotient and residue of
// x / 10^k against an exact math/big oracle, over boundary vectors and a
// deterministic random sweep, for k = 1..77.

import (
	"math/big"
	"testing"
)

func recipMul256Check(t *testing.T, x *big.Int, k int) {
	t.Helper()
	pow := ten(k)
	g := divPow10_q128res_256_recipmul(u256Of(x), k)
	q, r := new(big.Int).QuoRem(x, pow, new(big.Int))
	if big128(g.quotDw0, g.quotDw1).Cmp(q) != 0 {
		t.Errorf("quot 10^%d x=%v", k, x)
	}
	if g.residue != expectedResidueOf(r, pow) {
		t.Errorf("residue 10^%d x=%v", k, x)
	}
}

// recipMul256MaxQ is the largest quotient at k: < 2^128 and keeping x < 2^256.
func recipMul256MaxQ(k int) *big.Int {
	maxQ := new(big.Int).Sub(pow2(128), big.NewInt(1))
	if byX := new(big.Int).Quo(new(big.Int).Sub(pow2(256), big.NewInt(1)), ten(k)); byX.Cmp(maxQ) < 0 {
		return byX
	}
	return maxQ
}

func TestDivPow10RecipMul256Recipmul256_boundaryVectorsAllK(t *testing.T) {
	// m·10^k + {0, 1, half-1, half, half+1, 10^k-1} for m in {0, 1, 2, 3,
	// maxQ/2, maxQ-1, maxQ}: maxQ is the largest quotient keeping x < 2^256
	// and < 2^128, so the top-of-range cases land exactly on the residue
	// transitions (the worst case for reciprocal exactness).
	one := big.NewInt(1)
	xHiCap := new(big.Int).Sub(pow2(256), one)
	for k := 1; k <= 77; k++ {
		maxQ, powK := recipMul256MaxQ(k), ten(k)
		half := new(big.Int).Rsh(powK, 1)
		for _, m := range []*big.Int{big.NewInt(0), one, big.NewInt(2), big.NewInt(3), new(big.Int).Rsh(maxQ, 1), new(big.Int).Sub(maxQ, one), maxQ} {
			if m.Cmp(maxQ) > 0 {
				continue
			}
			base := new(big.Int).Mul(m, powK)
			for _, off := range []*big.Int{big.NewInt(0), one, new(big.Int).Sub(half, one), half, new(big.Int).Add(half, one), new(big.Int).Sub(powK, one)} {
				if x := new(big.Int).Add(base, off); x.Cmp(xHiCap) <= 0 {
					recipMul256Check(t, x, k)
				}
			}
		}
	}
}

func TestDivPow10RecipMul256Recipmul256_randomSweepAllK(t *testing.T) {
	rng := splitMix64{0x1234_5678_9ABC_DEF0}
	one := big.NewInt(1)
	xHiCap := new(big.Int).Sub(pow2(256), one)
	for k := 1; k <= 77; k++ {
		maxQ, powK := recipMul256MaxQ(k), ten(k)
		// x_hi for the 256-path regime: min(2^256-1, 10^k·2^128 - 1)
		xHiK := new(big.Int).Sub(new(big.Int).Lsh(powK, 128), one)
		if xHiK.Cmp(xHiCap) > 0 {
			xHiK = xHiCap
		}
		for i := 0; i < 300; i++ {
			// (a) q·10^k + r: guarantees the constraints and hits every residue.
			xa := new(big.Int).Add(new(big.Int).Mul(rng.below(new(big.Int).Add(maxQ, one)), powK), rng.below(powK))
			if xa.Cmp(xHiCap) <= 0 {
				recipMul256Check(t, xa, k)
			}
			// (b) a uniform full-range x (exercises hi != 0, the 256-path)
			recipMul256Check(t, rng.below(new(big.Int).Add(xHiK, one)), k)
		}
	}
}
