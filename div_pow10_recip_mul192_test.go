package decimal128

// Vector test for divPow10_q128res_192_recipmul (DivPow10RecipMul192WorkOrder.md).
//
// The 3x3 kernel is a narrowed sibling of divPow10_q128res_256_recipmul: for a
// stripped dividend A = x>>(k-1) < 2^191 it uses a 192x192->384 comba (9 macs)
// where the 4x4 uses 16. Two oracles per case: the 4x4 256-kernel (PRIMARY —
// the 192 result must be BIT-IDENTICAL), and an exact math/big x/10^k,
// x%10^k. The dispatch gate (bitLen(x) <= 190 + pow10) is exercised separately
// against the Core routing.

import (
	"math/big"
	"testing"
)

// recipMul192AFits191 reports A = x>>(k-1) < 2^191 (the 3x3 precondition).
func recipMul192AFits191(x *big.Int, k int) bool {
	return new(big.Int).Rsh(x, uint(k-1)).Cmp(pow2(191)) < 0
}

func recipMul192Check(t *testing.T, x *big.Int, k int) {
	t.Helper()
	pow := ten(k)
	g := divPow10_q128res_192_recipmul(u256Of(x), k)
	q, r := new(big.Int).QuoRem(x, pow, new(big.Int))
	if big128(g.quotDw0, g.quotDw1).Cmp(q) != 0 || g.residue != expectedResidueOf(r, pow) {
		t.Errorf("10^%d x=%v", k, x)
	}
	// PRIMARY: bit-identical to the 4x4 256-kernel.
	if g256 := divPow10_q128res_256_recipmul(u256Of(x), k); g != g256 {
		t.Errorf("3x3 != 4x4 at 10^%d x=%v", k, x)
	}
}

// recipMul192Bounds gives xCap (x < 2^256, quotient < 2^128, A < 2^191) and the
// matching maxQ.
func recipMul192Bounds(k int) (maxQ, powK, xCap *big.Int) {
	one := big.NewInt(1)
	powK = ten(k)
	xCap = new(big.Int).Sub(pow2(256), one)
	for _, c := range []*big.Int{new(big.Int).Sub(new(big.Int).Lsh(powK, 128), one), new(big.Int).Sub(pow2(uint(190+k)), one)} {
		if c.Cmp(xCap) < 0 {
			xCap = c
		}
	}
	maxQ = new(big.Int).Sub(pow2(128), one)
	if byX := new(big.Int).Quo(xCap, powK); byX.Cmp(maxQ) < 0 {
		maxQ = byX
	}
	return maxQ, powK, xCap
}

func TestDivPow10RecipMul192Recipmul192_boundaryVectorsAllK(t *testing.T) {
	// m·10^k + {0,1,half-1,half,half+1,10^k-1} for m in {0,1,2,3,maxQ/2,maxQ-1,
	// maxQ}: at large k the top of range lands A right below 2^191 (the worst
	// case for a 192-bit reciprocal) exactly on the residue transitions.
	one := big.NewInt(1)
	for k := 1; k <= 77; k++ {
		maxQ, powK, xCap := recipMul192Bounds(k)
		half := new(big.Int).Rsh(powK, 1)
		for _, m := range []*big.Int{big.NewInt(0), one, big.NewInt(2), big.NewInt(3), new(big.Int).Rsh(maxQ, 1), new(big.Int).Sub(maxQ, one), maxQ} {
			if m.Cmp(maxQ) > 0 {
				continue
			}
			base := new(big.Int).Mul(m, powK)
			for _, off := range []*big.Int{big.NewInt(0), one, new(big.Int).Sub(half, one), half, new(big.Int).Add(half, one), new(big.Int).Sub(powK, one)} {
				if x := new(big.Int).Add(base, off); x.Cmp(xCap) <= 0 && recipMul192AFits191(x, k) {
					recipMul192Check(t, x, k)
				}
			}
		}
	}
}

func TestDivPow10RecipMul192Recipmul192_randomSweepAllK(t *testing.T) {
	rng := splitMix64{0x0192_0192_0192_0192}
	one := big.NewInt(1)
	for k := 1; k <= 77; k++ {
		maxQ, powK, xCap := recipMul192Bounds(k)
		for i := 0; i < 300; i++ {
			// (a) q·10^k + r: hits every residue class, respects all bounds.
			xa := new(big.Int).Add(new(big.Int).Mul(rng.below(new(big.Int).Add(maxQ, one)), powK), rng.below(powK))
			if xa.Cmp(xCap) <= 0 && recipMul192AFits191(xa, k) {
				recipMul192Check(t, xa, k)
			}
			// (b) a uniform x in [0, xCap] (A near 2^191 for large k)
			if xb := rng.below(new(big.Int).Add(xCap, one)); recipMul192AFits191(xb, k) {
				recipMul192Check(t, xb, k)
			}
		}
	}
}

// The dispatch gate: bitLen(x) <= 190 + pow10  <=>  A = x>>(pow10-1) < 2^191.
func TestDivPow10RecipMul192Dispatch_gateEquivalence(t *testing.T) {
	rng := splitMix64{0x0DE7_0DE7_0DE7_0DE7}
	for k := 1; k <= 77; k++ {
		for i := 0; i < 500; i++ {
			x := rng.below(pow2(256))
			xu := u256Of(x)
			if (bitLen_int_256(xu.dw0, xu.dw1, xu.dw2, xu.dw3) <= 190+k) != recipMul192AFits191(x, k) {
				t.Errorf("gate mismatch k=%d x=%v", k, x)
			}
		}
	}
}

// The Core dispatcher (div_256_pow10_q128res) must be correct on both sides of
// the gate: 3x3 when A < 2^191, 4x4 otherwise. Inputs have x.hi != 0 (256-path)
// and straddle the A = 2^191 threshold where it is reachable (k >= ~28).
func TestDivPow10RecipMul192Dispatch_correctBothBranches(t *testing.T) {
	rng := splitMix64{0x0C0F_FEE0_C0FF_EE00}
	one := big.NewInt(1)
	two128, two256m1 := pow2(128), new(big.Int).Sub(pow2(256), one)
	took192, took256 := 0, 0
	for k := 1; k <= 77; k++ {
		powK := ten(k)
		maxQ := new(big.Int).Sub(two128, one)
		if byX := new(big.Int).Quo(two256m1, powK); byX.Cmp(maxQ) < 0 {
			maxQ = byX
		}
		for i := 0; i < 400; i++ {
			q := rng.below(new(big.Int).Add(maxQ, one))
			r := rng.below(powK)
			x := new(big.Int).Add(new(big.Int).Mul(q, powK), r)
			if x.Cmp(two128) < 0 || x.Cmp(two256m1) > 0 {
				continue // x.hi != 0, < 2^256
			}
			g := div_256_pow10_q128res(u256Of(x), k)
			if big128(g.quotDw0, g.quotDw1).Cmp(q) != 0 || g.residue != expectedResidueOf(r, powK) {
				t.Errorf("dispatch 10^%d x=%v", k, x)
			}
			if recipMul192AFits191(x, k) {
				took192++
			} else {
				took256++
			}
		}
	}
	if took192 == 0 || took256 == 0 {
		t.Errorf("both branches must be exercised: 3x3 %d, 4x4 %d", took192, took256)
	}
}
