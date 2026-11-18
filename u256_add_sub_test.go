package decimal128

// U256 add/sub/subAbs conformance (PrimitiveLayer.md Section 4.2),
// cross-checked against math/big. Trapping ops are checked both for the
// in-range result and for the overflow/underflow abort (a trap check per
// trapping case).

import (
	"math"
	"math/big"
	"testing"
)

var u256AddSubS = []u256{
	{0, 0, 0, 0},
	{1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, -1, 0, 0},
	{-1, -1, -1, 0},
	{-1, -1, -1, -1},
	{0x0123_4567_89AB_CDEF, s64(0x89AB_CDEF_0123_4567), s64(0xDEAD_BEEF_CAFE_BABE), 1},
	{math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64},
}

func TestU256AddSubAdd_256x256_andOverflowTrap(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range u256AddSubS {
			want := new(big.Int).Add(big256v(x), big256v(y))
			f := func() u256 { return add_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Error("add_256_256x256")
				}
			} else {
				expectTrap(t, "add_256_256x256", func() { f() })
			}
		}
	}
}

func TestU256AddSubAdd_256x128_andOverflowTrap(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range u256AddSubS {
			want := new(big.Int).Add(big256v(x), big128(y.dw0, y.dw1))
			f := func() u256 { return add_256_256x128(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Error("add_256_256x128")
				}
			} else {
				expectTrap(t, "add_256_256x128", func() { f() })
			}
		}
	}
}

func TestU256AddSubAdd_256x64_andOverflowTrap(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range []int64{0, 1, -1, math.MinInt64} {
			want := new(big.Int).Add(big256v(x), ubig(y))
			f := func() u256 { return add_256_256x64(x.dw0, x.dw1, x.dw2, x.dw3, y) }
			if want.Cmp(pow2(256)) < 0 {
				if big256v(f()).Cmp(want) != 0 {
					t.Error("add_256_256x64")
				}
			} else {
				expectTrap(t, "add_256_256x64", func() { f() })
			}
		}
	}
}

func TestU256AddSubAdd_128x128_neverOverflows(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range u256AddSubS {
			want := new(big.Int).Add(big128(x.dw0, x.dw1), big128(y.dw0, y.dw1))
			if big256v(add_256_128x128(x.dw0, x.dw1, y.dw0, y.dw1)).Cmp(want) != 0 {
				t.Error("add_256_128x128")
			}
		}
	}
}

func TestU256AddSubSub_256x256_andUnderflowTrap(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range u256AddSubS {
			bx, by := big256v(x), big256v(y)
			f := func() u256 { return sub_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3) }
			if bx.Cmp(by) >= 0 {
				if big256v(f()).Cmp(new(big.Int).Sub(bx, by)) != 0 {
					t.Error("sub_256_256x256")
				}
			} else {
				expectTrap(t, "sub_256_256x256", func() { f() })
			}
		}
	}
}

func TestU256AddSubSubAbs_256x256_absAndSwap(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range u256AddSubS {
			bx, by := big256v(x), big256v(y)
			d := subAbs_d256swap_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3)
			if big256(d.diffDw0, d.diffDw1, d.diffDw2, d.diffDw3).Cmp(new(big.Int).Abs(new(big.Int).Sub(bx, by))) != 0 || d.swap != (by.Cmp(bx) > 0) {
				t.Error("subAbs_d256swap_256x256")
			}
		}
	}
}

func TestU256AddSubSubAbs_256x128_absAndSwap(t *testing.T) {
	for _, x := range u256AddSubS {
		for _, y := range u256AddSubS {
			bx, by := big256v(x), big128(y.dw0, y.dw1)
			d := subAbs_d256swap_256x128(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1)
			if big256(d.diffDw0, d.diffDw1, d.diffDw2, d.diffDw3).Cmp(new(big.Int).Abs(new(big.Int).Sub(bx, by))) != 0 || d.swap != (by.Cmp(bx) > 0) {
				t.Error("subAbs_d256swap_256x128")
			}
		}
	}
}
