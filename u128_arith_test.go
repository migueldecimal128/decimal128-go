package decimal128

// U128 vocabulary conformance (PrimitiveLayer.md Section 5), cross-checked
// against math/big as the unsigned 128-bit oracle. Operand pairs span the sign
// boundary and the carry/borrow boundary, where synthesized 128-bit arithmetic
// diverges from naive signed 64-bit work.

import (
	"math"
	"math/big"
	"testing"
)

// Representative limbs: zero, one, max, top-bit set, and arbitrary.
var u128ArithLimbs = []int64{0, 1, 2, -1, math.MinInt64, math.MaxInt64, 0x0123_4567_89AB_CDEF, s64(0xDEAD_BEEF_CAFE_BABE)}

func u128ArithSamples() []u128 {
	var out []u128
	for _, hi := range u128ArithLimbs {
		for _, lo := range u128ArithLimbs {
			out = append(out, fromLoHi_128_64x64(lo, hi))
		}
	}
	return out
}

func u128Val(v u128) *big.Int { return big128(v.dw0, v.dw1) }

func mod128(x *big.Int) *big.Int { return new(big.Int).Mod(x, pow2(128)) }

func TestU128ArithAdd_sub_mul_wrapMod2pow128(t *testing.T) {
	for _, x := range u128ArithSamples() {
		for _, y := range u128ArithSamples() {
			bx, by := u128Val(x), u128Val(y)
			if u128Val(add_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)).Cmp(mod128(new(big.Int).Add(bx, by))) != 0 {
				t.Error("add")
			}
			if u128Val(sub_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)).Cmp(mod128(new(big.Int).Sub(bx, by))) != 0 {
				t.Error("sub")
			}
			if u128Val(mul_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)).Cmp(mod128(new(big.Int).Mul(bx, by))) != 0 {
				t.Error("mul")
			}
		}
	}
}

func TestU128ArithBitwise_perLimb(t *testing.T) {
	for _, x := range u128ArithSamples() {
		for _, y := range u128ArithSamples() {
			bx, by := u128Val(x), u128Val(y)
			if u128Val(and_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)).Cmp(new(big.Int).And(bx, by)) != 0 {
				t.Error("and")
			}
			if u128Val(or_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)).Cmp(new(big.Int).Or(bx, by)) != 0 {
				t.Error("or")
			}
		}
		want := new(big.Int).Sub(new(big.Int).Sub(pow2(128), big.NewInt(1)), u128Val(x))
		if u128Val(not_128_128(x.dw0, x.dw1)).Cmp(want) != 0 {
			t.Error("not")
		}
	}
}

func TestU128ArithShifts_acrossAllBoundaries(t *testing.T) {
	for _, x := range u128ArithSamples() {
		bx := u128Val(x)
		for _, n := range []int{0, 1, 7, 31, 63, 64, 65, 100, 127} {
			if u128Val(shl_128_128(x.dw0, x.dw1, n)).Cmp(mod128(new(big.Int).Lsh(bx, uint(n)))) != 0 {
				t.Errorf("shl by %d", n)
			}
			if u128Val(shr_128_128(x.dw0, x.dw1, n)).Cmp(new(big.Int).Rsh(bx, uint(n))) != 0 {
				t.Errorf("shr by %d", n)
			}
		}
	}
}

func TestU128ArithCmp_isUnsignedMinusOneZeroPlusOne(t *testing.T) {
	for _, x := range u128ArithSamples() {
		for _, y := range u128ArithSamples() {
			if cmp_int_128x128(x.dw0, x.dw1, y.dw0, y.dw1) != u128Val(x).Cmp(u128Val(y)) {
				t.Error("cmp")
			}
		}
	}
	// Spot-check the sign boundary explicitly: 2^64 vs 1.
	if cmp_int_128x128(0, 1, 1, 0) != 1 || cmp_int_128x128(1, 0, 0, 1) != -1 {
		t.Error("sign boundary")
	}
}

func TestU128ArithIsZero_fromLoHi(t *testing.T) {
	if !isZero_bool_128(0, 0) || isZero_bool_128(1, 0) || isZero_bool_128(0, 1) {
		t.Error("isZero")
	}
	v := fromLoHi_128_64x64(0xAAAA, 0xBBBB)
	if v.dw0 != 0xAAAA || v.dw1 != 0xBBBB {
		t.Error("fromLoHi")
	}
}

func TestU128ArithMul64x64_isExactFullWidth(t *testing.T) {
	for _, a := range u128ArithLimbs {
		for _, b := range u128ArithLimbs {
			if u128Val(mul_128_64x64(a, b)).Cmp(new(big.Int).Mul(ubig(a), ubig(b))) != 0 {
				t.Error("mul_128_64x64")
			}
		}
	}
}

func TestU128ArithSubAbs_absoluteDifferenceAndSwapFlag(t *testing.T) {
	for _, x := range u128ArithSamples() {
		for _, y := range u128ArithSamples() {
			bx, by := u128Val(x), u128Val(y)
			d := subAbs_d128swap_128x128(x.dw0, x.dw1, y.dw0, y.dw1)
			if big128(d.diffDw0, d.diffDw1).Cmp(new(big.Int).Abs(new(big.Int).Sub(bx, by))) != 0 {
				t.Error("|x-y|")
			}
			if d.swap != (by.Cmp(bx) > 0) {
				t.Error("swap == (y > x)")
			}
		}
	}
}
