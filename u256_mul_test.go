package decimal128

// U256 mul conformance (PrimitiveLayer.md Section 4.2), cross-checked against
// math/big.

import (
	"math"
	"math/big"
	"testing"
)

var u256MulLimbs = []int64{0, 1, -1, math.MinInt64, 0x0123_4567_89AB_CDEF, s64(0xDEAD_BEEF_CAFE_BABE)}

func TestU256MulMul_128x128_isExactFullWidth(t *testing.T) {
	for _, x0 := range u256MulLimbs {
		for _, x1 := range u256MulLimbs {
			for _, y0 := range u256MulLimbs {
				for _, y1 := range u256MulLimbs {
					want := new(big.Int).Mul(big128(x0, x1), big128(y0, y1))
					if big256v(mul_256_128x128(x0, x1, y0, y1)).Cmp(want) != 0 {
						t.Error("mul_256_128x128")
					}
				}
			}
		}
	}
}

func TestU256MulMul_256x64_andOverflowTrap(t *testing.T) {
	x0, x1, x2 := int64(0x0123_4567_89AB_CDEF), s64(0xCAFE_BABE_0000_0001), int64(0x0000_0000_DEAD_BEEF)
	for _, x3 := range u256MulLimbs {
		for _, y := range []int64{0, 1, -1, 1_000_000} {
			want := new(big.Int).Mul(big256(x0, x1, x2, x3), ubig(y))
			f := func() u256 { return mul_256_256x64(x0, x1, x2, x3, y) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Error("mul_256_256x64")
				}
			} else {
				expectTrap(t, "mul_256_256x64", func() { f() })
			}
		}
	}
}

func TestU256MulMul_256x128_andOverflowTrap(t *testing.T) {
	xs := [][4]int64{{1, 0, 0, 0}, {-1, -1, 0, 0}, {-1, -1, -1, 0}, {0x1, 0x2, 0x3, 0x4}, {-1, -1, -1, -1}}
	ys := [][2]int64{{1, 0}, {-1, 0}, {-1, -1}, {0xDEAD, 0xBEEF}}
	for _, x := range xs {
		for _, y := range ys {
			want := new(big.Int).Mul(big256(x[0], x[1], x[2], x[3]), big128(y[0], y[1]))
			f := func() u256 { return mul_256_256x128(x[0], x[1], x[2], x[3], y[0], y[1]) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Error("mul_256_256x128")
				}
			} else {
				expectTrap(t, "mul_256_256x128", func() { f() })
			}
		}
	}
}

func TestU256MulMul_128x64_interior_isExact(t *testing.T) {
	for _, x0 := range u256MulLimbs {
		for _, x1 := range u256MulLimbs {
			for _, y := range u256MulLimbs {
				want := new(big.Int).Mul(big128(x0, x1), ubig(y))
				if big256v(mul_256_128x64(fromLoHi_128_64x64(x0, x1), uint64(y))).Cmp(want) != 0 {
					t.Error("mul_256_128x64")
				}
			}
		}
	}
}

func TestU256MulMul_256x256_withinEnvelope_isExact(t *testing.T) {
	// Operands bounded to <= 2^120 (top limb <= 2^56) so intermediate
	// columns stay within the 128-bit accumulator — the supported
	// envelope. Products here are <= 2^240 and verified exactly.
	small := []int64{0, 1, 0x00FF_FFFF_FFFF_FFFF, 0x0012_3456_789A_BCDE}
	full := []int64{0, 1, -1, s64(0xCAFE_BABE_1234_5678)}
	for _, x2 := range small {
		for _, x0 := range full {
			for _, y2 := range small {
				for _, y0 := range full {
					want := new(big.Int).Mul(big256(x0, 0, x2, 0), big256(y0, 0, y2, 0))
					if want.Cmp(pow2(256)) < 0 && big256v(mul_256_256x256(x0, 0, x2, 0, y0, 0, y2, 0)).Cmp(want) != 0 {
						t.Error("mul_256_256x256")
					}
				}
			}
		}
	}
}

func TestU256MulMul_256x256_clearOverflowTraps(t *testing.T) {
	// 2^200 * 2^200 = 2^400, unambiguously over 2^256.
	hiLimb := int64(1) << (200 - 192) // bit 200 lives in limb 3
	expectTrap(t, "2^200 * 2^200", func() { mul_256_256x256(0, 0, 0, hiLimb, 0, 0, 0, hiLimb) })
}
