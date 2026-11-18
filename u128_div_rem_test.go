package decimal128

// 128-bit divRem conformance (PrimitiveLayer.md D15), cross-checked against
// math/big. Divisors span < 2^64 (delegates to 128/64) and >= 2^64 (the
// normalized single-digit path), plus the x < y short-circuit.

import (
	"math"
	"math/big"
	"testing"
)

var u128DivRemLimbs = []int64{0, 1, 2, -1, math.MinInt64, math.MaxInt64, 0x0123_4567_89AB_CDEF, s64(0xDEAD_BEEF_CAFE_BABE)}

func TestU128DivRemDivRem_128x64_matchesBigInteger(t *testing.T) {
	divisors := []int64{1, 2, 3, 7, 0xFFFF, 0xFFFF_FFFB, 0x1_0000_0007, s64(0x8000_0000_0000_0001), -1, 1 << 40}
	for _, x1 := range u128DivRemLimbs {
		for _, x0 := range u128DivRemLimbs {
			bx := big128(x0, x1)
			for _, y := range divisors {
				r := divRem_q128r64_128x64(x0, x1, y)
				q, m := new(big.Int).QuoRem(bx, ubig(y), new(big.Int))
				if big128(r.quotDw0, r.quotDw1).Cmp(q) != 0 || ubig(r.remDw0).Cmp(m) != 0 {
					t.Errorf("%v / %d", bx, uint64(y))
				}
			}
		}
	}
}

func TestU128DivRemDivRem_128x128_matchesBigInteger(t *testing.T) {
	divisors := [][2]int64{
		{1, 0}, {7, 0}, {-1, 0}, // < 2^64
		{0, 1}, {1, 1}, {-1, -1}, // >= 2^64
		{0x1, math.MinInt64},                 // top bit set, s == 0
		{0xCAFE_BABE, 0x0000_0000_DEAD_BEEF}, // mid-range >= 2^64
		{3, 5},
	}
	for _, x1 := range u128DivRemLimbs {
		for _, x0 := range u128DivRemLimbs {
			bx := big128(x0, x1)
			for _, y := range divisors {
				by := big128(y[0], y[1])
				r := divRem_q128r128_128x128(x0, x1, y[0], y[1])
				q, m := new(big.Int).QuoRem(bx, by, new(big.Int))
				if big128(r.quotDw0, r.quotDw1).Cmp(q) != 0 || big128(r.remDw0, r.remDw1).Cmp(m) != 0 {
					t.Errorf("%v / %v", bx, by)
				}
			}
		}
	}
}

func TestU128DivRemDivisionByZeroTraps(t *testing.T) {
	expectTrap(t, "128/64 by zero", func() { divRem_q128r64_128x64(1, 2, 0) })
	expectTrap(t, "128/128 by zero", func() { divRem_q128r128_128x128(1, 2, 0, 0) })
}
