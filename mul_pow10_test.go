package decimal128

// mulPow10 conformance (PrimitiveLayer.md Section 4.4), cross-checked against
// math/big. A product that fits 256 bits must be exact; one that does not must
// trap (a trap check per trapping case).

import (
	"math/big"
	"testing"
)

var mulPow10X128 = [][2]int64{{0, 0}, {1, 0}, {-1, 0}, {-1, -1}, {0x0123_4567_89AB_CDEF, 0x0000_0000_DEAD_BEEF}, {0, 1}}

func TestMulPow10MulPow10_128_128_wrapsMod2pow128(t *testing.T) {
	for _, x := range mulPow10X128 {
		for p := 0; p <= 38; p++ {
			r := mulPow10_128_128(x[0], x[1], p)
			want := new(big.Int).Mod(new(big.Int).Mul(big128(x[0], x[1]), ten(p)), pow2(128))
			if big128(r.dw0, r.dw1).Cmp(want) != 0 {
				t.Errorf("10^%d", p)
			}
		}
	}
}

func TestMulPow10MulPow10_256_128_fitOrTrap(t *testing.T) {
	for _, x := range mulPow10X128 {
		for p := 0; p <= 77; p++ {
			want := new(big.Int).Mul(big128(x[0], x[1]), ten(p))
			f := func() u256 { return mulPow10_256_128(x[0], x[1], p) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Errorf("10^%d", p)
				}
			} else {
				expectTrap(t, "mulPow10_256_128", func() { f() })
			}
		}
	}
}

func TestMulPow10MulPow10_256_256_hiZero_allPaths(t *testing.T) {
	// x <= 128 bits: every path here uses an exact multiply, so fit/trap
	// is exact across the full power range.
	for _, x := range mulPow10X128 {
		for p := 0; p <= 77; p++ {
			want := new(big.Int).Mul(big128(x[0], x[1]), ten(p))
			f := func() u256 { return mulPow10_256_256(x[0], x[1], 0, 0, p) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Errorf("10^%d", p)
				}
			} else {
				expectTrap(t, "mulPow10_256_256", func() { f() })
			}
		}
	}
}

func TestMulPow10MulPow10_256_256_wideX_smallPow_exact(t *testing.T) {
	// x > 128 bits, pow <= 38: routes through exact 256x64 / 256x128.
	for _, x := range []u256{{-1, -1, 1, 0}, {0x1234, 0x5678, 0x9ABC, 0}, {-1, -1, -1, 0}, {1, 0, 0, 1}} {
		for p := 0; p <= 38; p++ {
			want := new(big.Int).Mul(big256v(x), ten(p))
			f := func() u256 { return mulPow10_256_256(x.dw0, x.dw1, x.dw2, x.dw3, p) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Errorf("10^%d", p)
				}
			} else {
				expectTrap(t, "mulPow10_256_256", func() { f() })
			}
		}
	}
}

func TestMulPow10MulPow10_256_256_wideX_bigPow_traps(t *testing.T) {
	// x > 2^128 and pow in 39..77: the product exceeds 2^256 (x >= 2^131,
	// 10^39 > 2^129), so mulPow10_256_256 must trap.
	for _, x := range []u256{{0x1234, 0x5678, 0x0000_00AB, 0}, {-1, 0x00FF, 0x000F, 0}, {7, 0, 0, 1}} {
		for p := 39; p <= 77; p++ {
			if new(big.Int).Mul(big256v(x), ten(p)).Cmp(pow2(256)) < 0 {
				t.Errorf("10^%d fits", p)
			}
			expectTrap(t, "mulPow10_256_256", func() { mulPow10_256_256(x.dw0, x.dw1, x.dw2, x.dw3, p) })
		}
	}
}
