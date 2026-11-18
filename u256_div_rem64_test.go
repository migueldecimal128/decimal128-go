package decimal128

import (
	"math"
	"math/big"
	"testing"
)

var u256DivRem64Cases = []struct {
	name string
	x    [4]int64
	y0   uint64
}{
	{"y=1, x=0", [4]int64{0, 0, 0, 0}, 1},
	{"y=1, x=max256", [4]int64{-1, -1, -1, -1}, 1},
	{"x<2^64, exact", [4]int64{1_000_000, 0, 0, 0}, 1_000},
	{"x<2^64, with remainder", [4]int64{1_000_007, 0, 0, 0}, 1_000},
	{"x<2^64, x==y", [4]int64{12345, 0, 0, 0}, 12345},
	{"x<2^64, x<y", [4]int64{99, 0, 0, 0}, 100},
	{"x<2^64, x==UInt64.max", [4]int64{-1, 0, 0, 0}, 3},
	{"pow2 y=2, x even", [4]int64{s64(0xFFFF_FFFF_FFFF_FFFE), 0, 0, 0}, 2},
	{"pow2 y=2, x odd", [4]int64{s64(0xFFFF_FFFF_FFFF_FFFF), 0, 0, 0}, 2},
	{"pow2 y=256, x spanning limbs", [4]int64{s64(0xDEAD_BEEF_CAFE_BABE), 0x1234_5678, 0, 0}, 256},
	{"pow2 y=2^32, full 256", [4]int64{-1, -1, -1, -1}, 1 << 32},
	{"pow2 y=2^63, x=2^136", [4]int64{0, 1 << 8, 0, 0}, 1 << 63},
	{"x16 y=3, x=10", [4]int64{10, 0, 0, 0}, 3},
	{"x16 y=7, x=2^64", [4]int64{0, 1, 0, 0}, 7},
	{"x16 y=10, x=10^20", [4]int64{0x6BC7_5E2D_6310_0000, 0x5, 0, 0}, 10},
	{"x16 y=65535 (max 16-bit), x=2^192", [4]int64{0, 0, 0, 1}, 0xFFFF},
	{"x16 y=9999, x=max256", [4]int64{-1, -1, -1, -1}, 9_999},
	{"x32 y=100003 (prime ~17 bits), x=2^64", [4]int64{0, 1, 0, 0}, 100_003},
	{"x32 y=2^31-1 (Mersenne), x=2^128-1", [4]int64{-1, -1, 0, 0}, 0x7FFF_FFFF},
	{"x32 y=UInt32.max, x=2^256-2^192", [4]int64{0, 0, 0, -1}, 0xFFFF_FFFF},
	{"x32 y=0xDEAD_BEEF, x=max256", [4]int64{-1, -1, -1, -1}, 0xDEAD_BEEF},
	{"knuth y=2^33, x=2^64", [4]int64{0, 1, 0, 0}, 1 << 33},
	{"knuth y=2^40-5, x=2^128-ish", [4]int64{0x2345_6789_ABCD_EF01, s64(0xABCD_EF01), 0, 0}, 0xF_FFFF_FFFB},
	{"knuth y=UInt64.max, x=max256", [4]int64{-1, -1, -1, -1}, math.MaxUint64},
	{"knuth y=UInt64.max-1, x=max256", [4]int64{-1, -1, -1, -1}, math.MaxUint64 - 1},
	{"knuth y=10^19, x=max256", [4]int64{-1, -1, -1, -1}, 10_000_000_000_000_000_000},
	{"knuth y=arbitrary 64-bit, x spanning all limbs", [4]int64{0x4444_4444_4444_4444, 0x3333_3333_3333_3333, 0x2222_2222_2222_2222, 0x1111_1111_1111_1111}, 0x9E37_79B9_7F4A_7C15},
}

func TestU256DivRem64TestCase(t *testing.T) {
	for _, tc := range u256DivRem64Cases {
		qr := divRem_q256r64_256x64(tc.x[0], tc.x[1], tc.x[2], tc.x[3], int64(tc.y0))
		r := uint64(qr.remDw0)
		// Remainder must be strictly less than the divisor.
		if r >= tc.y0 {
			t.Errorf("remainder >= divisor for %s", tc.name)
		}
		// Round-trip invariant: q * y0 + r == x.
		prod := mul_256_256x64(qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3, int64(tc.y0))
		got := add_256_256x64(prod.dw0, prod.dw1, prod.dw2, prod.dw3, int64(r))
		if got != (u256{tc.x[0], tc.x[1], tc.x[2], tc.x[3]}) {
			t.Errorf("round-trip failed for %s", tc.name)
		}
	}
}

var u256DivRem64Dividends = [][4]int64{
	{0, 0, 0, 0},
	{12345, 0, 0, 0},
	{-1, 0, 0, 0},
	{0x0123_4567_89AB_CDEF, 0xCAFE, 0, 0},
	{-1, -1, 0, 0},
	{0x1111_2222_3333_4444, 0x5555_6666_7777_8888, 0xABCD, 0},
	{-1, -1, -1, 0},
	{-1, -1, -1, -1},
	{0x0123_4567_89AB_CDEF, s64(0x89AB_CDEF_0123_4567), s64(0xFEDC_BA98_7654_3210), 0x0011_2233_4455_6677},
	{1, 0, 0, 1},
}

var u256DivRem64Divisors = []int64{
	1, 2, 3, 7, 100, 1024, // tiny / power-of-two
	65521, 0xFFFF, // < 2^16 (prime, max16)
	0x1_0000, 1_000_003, 0xFFFF_FFFB, // 2^16 .. < 2^32
	1 << 40,                    // power-of-two > 2^32
	0x1_0000_0007,              // just over 2^32
	s64(0x8000_0000_0000_0001), // 64-bit, top bit set
	-1,                         // 2^64 - 1
	7_237_005_577_332_262_213,  // a 63-bit prime
}

func TestU256DivRem64DivRem_matchesBigInteger_acrossAllBranches(t *testing.T) {
	for _, x := range u256DivRem64Dividends {
		bx := big256(x[0], x[1], x[2], x[3])
		for _, y := range u256DivRem64Divisors {
			r := divRem_q256r64_256x64(x[0], x[1], x[2], x[3], y)
			q, m := new(big.Int).QuoRem(bx, ubig(y), new(big.Int))
			if big256(r.quotDw0, r.quotDw1, r.quotDw2, r.quotDw3).Cmp(q) != 0 || ubig(r.remDw0).Cmp(m) != 0 {
				t.Errorf("%v / %d", bx, uint64(y))
			}
		}
	}
}

func TestU256DivRem64DivisionByZeroTraps(t *testing.T) {
	expectTrap(t, "256/64 by zero", func() { divRem_q256r64_256x64(1, 2, 3, 4, 0) })
}
