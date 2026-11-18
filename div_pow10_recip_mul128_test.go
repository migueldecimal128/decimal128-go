package decimal128

// Validates divPow10_q128res_128_recipmul (the single-multiply reciprocal
// kernel for U128 / 10^pow10, pow10 ∈ 1…38) against an independent math/big
// divide/mod reference. The reference is the definition of the result, so any
// disagreement is a kernel (or M/S table) defect. The random sweep is the real
// exercise of the table: large dividends are the worst case for reciprocal
// exactness.

import (
	"math/big"
	"testing"
)

// expectedResidueOf classifies remainder r against half of d (D17).
func expectedResidueOf(r, d *big.Int) residue {
	if r.Sign() == 0 {
		return const_RESIDUE_EXACT
	}
	switch new(big.Int).Lsh(r, 1).Cmp(d) {
	case -1:
		return const_RESIDUE_LT_HALF
	case 0:
		return const_RESIDUE_HALF
	}
	return const_RESIDUE_GT_HALF
}

func recipMul128Check(t *testing.T, x *big.Int, k int, label string) {
	t.Helper()
	d := ten(k)
	q, r := new(big.Int).QuoRem(x, d, new(big.Int))
	xv := u128Of(x)
	got := divPow10_q128res_128_recipmul(xv.dw0, xv.dw1, k)
	if big128(got.quotDw0, got.quotDw1).Cmp(q) != 0 {
		t.Errorf("quotient mismatch [%s] x=%v k=%d", label, x, k)
	}
	if got.residue != expectedResidueOf(r, d) {
		t.Errorf("residue mismatch [%s] x=%v k=%d: got %s", label, x, k, residueName(got.residue))
	}
}

func TestDivPow10RecipMul128ExplicitEdges(t *testing.T) {
	// Each residue class at k=1, plus k=0-adjacent magnitudes.
	u128max := new(big.Int).Sub(pow2(128), big.NewInt(1))
	recipMul128Check(t, big.NewInt(0), 1, "zero")
	recipMul128Check(t, big.NewInt(100), 2, "exact_100/100")
	recipMul128Check(t, big.NewInt(102), 2, "ltHalf_102/100")
	recipMul128Check(t, big.NewInt(150), 2, "half_150/100")
	recipMul128Check(t, big.NewInt(107), 2, "gtHalf_107/100")
	recipMul128Check(t, new(big.Int).Sub(pow2(64), big.NewInt(1)), 1, "u64max")
	recipMul128Check(t, u128max, 1, "u128max/10")
	recipMul128Check(t, u128max, 38, "u128max/1e38")
	// Quotient that just spills past 64 bits.
	recipMul128Check(t, pow2(70), 3, "2^70/1000")
}

// Around each residue boundary m·10^k + {0, 1, half-1, half, half+1, tenK-1},
// for several multipliers m including the largest that keeps x ≤ U128.max.
// The fixed multipliers wrap past 2^128 at k = 38 (m = 3), as Swift's do.
func TestDivPow10RecipMul128BoundariesAllK(t *testing.T) {
	one := big.NewInt(1)
	u128max := new(big.Int).Sub(pow2(128), one)
	for k := 1; k <= 38; k++ {
		tenK := ten(k)
		half := new(big.Int).Rsh(tenK, 1)
		maxM := new(big.Int).Quo(new(big.Int).Sub(u128max, new(big.Int).Sub(tenK, one)), tenK)
		ms := []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(2), big.NewInt(3), new(big.Int).Rsh(maxM, 1), new(big.Int).Sub(maxM, one), maxM}
		for _, m := range ms {
			base := new(big.Int).Mul(m, tenK)
			for _, off := range []*big.Int{big.NewInt(0), one, new(big.Int).Sub(half, one), half, new(big.Int).Add(half, one), new(big.Int).Sub(tenK, one)} {
				x := mod128(new(big.Int).Add(base, off))
				recipMul128Check(t, x, k, "boundary")
			}
		}
	}
}

func TestDivPow10RecipMul128RandomSweepAllK(t *testing.T) {
	rng := splitMix64{0x1234_5678_9ABC_DEF0}
	for k := 1; k <= 38; k++ {
		for i := 0; i < 3000; i++ {
			// Vary magnitude: mask to a random bit length so small
			// dividends are covered alongside full-width ones.
			bits := uint(rng.next()%128) + 1 // 1...128
			lo := rng.next()
			hi := rng.next()
			x := big128(int64(lo), int64(hi))
			if bits < 128 {
				x.And(x, new(big.Int).Sub(pow2(bits), big.NewInt(1)))
			}
			recipMul128Check(t, x, k, "rand")
		}
	}
}
