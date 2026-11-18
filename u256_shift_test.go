package decimal128

import (
	"math/big"
	"testing"
)

var u256ShiftS = []u256{
	{0, 0, 0, 0},
	{1, 0, 0, 0},
	{-1, -1, -1, -1},
	{0x0123_4567_89AB_CDEF, s64(0x89AB_CDEF_0123_4567), s64(0xDEAD_BEEF_CAFE_BABE), 0x42},
	{-1, 0, -1, 0},
}

func TestU256ShiftShr_acrossAllBoundaries(t *testing.T) {
	for _, x := range u256ShiftS {
		for _, n := range []int{0, 1, 7, 63, 64, 65, 127, 128, 129, 191, 200, 255} {
			if big256v(shr_256_256(x.dw0, x.dw1, x.dw2, x.dw3, n)).Cmp(new(big.Int).Rsh(big256v(x), uint(n))) != 0 {
				t.Errorf("shr %d", n)
			}
		}
	}
}

func TestU256ShiftShl1_and_shr1(t *testing.T) {
	for _, x := range u256ShiftS {
		bx := big256v(x)
		if big256v(shl1_256_256(x.dw0, x.dw1, x.dw2, x.dw3)).Cmp(new(big.Int).Mod(new(big.Int).Lsh(bx, 1), pow2(256))) != 0 {
			t.Error("shl1")
		}
		if big256v(shr1_256_256(x.dw0, x.dw1, x.dw2, x.dw3)).Cmp(new(big.Int).Rsh(bx, 1)) != 0 {
			t.Error("shr1")
		}
	}
}
