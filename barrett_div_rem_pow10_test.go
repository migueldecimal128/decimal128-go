package decimal128

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

var barrettCases = []struct {
	name      string
	x         string // decimal string for U256
	pow10     int
	expectedQ string // decimal string for U256
	expectedR uint64
}{
	{"pow10=1, x=0", "0", 1, "0", 0},
	{"pow10=1, x=10 (exact)", "10", 1, "1", 0},
	{"pow10=1, x=27", "27", 1, "2", 7},
	{"pow10=1, x=9 (less than divisor)", "9", 1, "0", 9},
	{"pow10=1, x=UInt64.max", "18446744073709551615", 1, "1844674407370955161", 5},
	{"pow10=2, x=12345", "12345", 2, "123", 45},
	{"pow10=3, x=1234567890", "1234567890", 3, "1234567", 890},
	{"pow10=4, x=99999 (exact-1)", "99999", 4, "9", 9999},
	{"pow10=4, x=2^128-1 (exercises 54_172 kernel)", "340282366920938463463374607431768211455", 4, "34028236692093846346337460743176821", 1455},
	{"pow10=4, x=2^192-1 (exercises 54_226 kernel)", "6277101735386680763835789423207666416102355444464034512895", 4, "627710173538668076383578942320766641610235544446403451", 2895},
	{"pow10=4, x=10^77-1 (max 77-digit value, exercises 54_256 kernel)", "99999999999999999999999999999999999999999999999999999999999999999999999999999", 4, "9999999999999999999999999999999999999999999999999999999999999999999999999", 9999},
	{"pow10=5, x=10^10 (exact)", "10000000000", 5, "100000", 0},
	{"pow10=6, x=10^12+1", "1000000000001", 6, "1000000", 1},
	{"pow10=7, x=10^15-1", "999999999999999", 7, "99999999", 9999999},
	{"pow10=10, x=10^20", "100000000000000000000", 10, "10000000000", 0},
	{"pow10=13, x=10^26 (exact)", "100000000000000000000000000", 13, "10000000000000", 0},
	{"pow10=10, x=2^128", "340282366920938463463374607431768211456", 10, "34028236692093846346337460743", 1768211456},
	{"pow10=10, x=2^192-1 (exercises 32_192 kernel)", "6277101735386680763835789423207666416102355444464034512895", 10, "627710173538668076383578942320766641610235544446", 4034512895},
	{"pow10=10, x=10^77-1 (max 77-digit value, exercises 54_256 kernel)", "9999999999_9999999999_9999999999_9999999999_9999999999_9999999999_9999999999_9999999", 10, "9999999999999999999999999999999999999999999999999999999999999999999", 9999999999},
	{"pow10=13, x=10^77-1 (max 77-digit value, exercises 54_256 kernel)", "9999999999_9999999999_9999999999_9999999999_9999999999_9999999999_9999999999_9999999", 13, "9999999999999999999999999999999999999999999999999999999999999999", 9999999999999},
	{"pow10=5, x=2^64 (just over 64-bit boundary)", "18446744073709551616", 5, "184467440737095", 51616},
	{"pow10=5, x=2^128 (just over 128-bit boundary)", "340282366920938463463374607431768211456", 5, "3402823669209384634633746074317682", 11456},
	{"pow10=5, x=2^192 (just over 192-bit boundary)", "6277101735386680763835789423207666416102355444464034512896", 5, "62771017353866807638357894232076664161023554444640345", 12896},
}

func barrettRoundTrip(t *testing.T, x, q u256, r uint64, pow10 int, label string) {
	t.Helper()
	// q * 10^pow10 + r == x
	prod := mul_256_256x64(q.dw0, q.dw1, q.dw2, q.dw3, pow10_128_dw0(pow10))
	if add_256_256x64(prod.dw0, prod.dw1, prod.dw2, prod.dw3, int64(r)) != x {
		t.Errorf("round-trip failed for %s", label)
	}
}

func TestBarrettDivRemPow10BarrettDivPow10(t *testing.T) {
	for _, tc := range barrettCases {
		x := u256Of(bigStr(strings.ReplaceAll(tc.x, "_", "")))
		qr := divPow10_q256r64_256_barrett(x.dw0, x.dw1, x.dw2, x.dw3, tc.pow10)
		q := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
		if q != u256Of(bigStr(tc.expectedQ)) {
			t.Errorf("quotient mismatch for %s", tc.name)
		}
		if uint64(qr.remDw0) != tc.expectedR {
			t.Errorf("remainder mismatch for %s", tc.name)
		}
		barrettRoundTrip(t, x, q, uint64(qr.remDw0), tc.pow10, tc.name)
	}
}

func TestBarrettDivRemPow10BarrettDivPow10_randomized(t *testing.T) {
	rng := splitMix64{0x0BA2_2E77_D1F0_0010}
	for i := 0; i < 1_000; i++ {
		x := u256{int64(rng.next()), int64(rng.next()), int64(rng.next()), int64(rng.next())}
		pow10 := rng.rangeIn(1, 13)
		qr := divPow10_q256r64_256_barrett(x.dw0, x.dw1, x.dw2, x.dw3, pow10)
		q := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
		if uint64(qr.remDw0) >= uint64(pow10_128_dw0(pow10)) {
			t.Error("remainder out of range")
		}
		barrettRoundTrip(t, x, q, uint64(qr.remDw0), pow10, "random")
	}
}

// MARK: - 64-bit Barrett step and the 128-bit divide (math/big oracle)

func TestBarrettDivRemPow10BarrettStep_matchesUnsignedDivRem(t *testing.T) {
	dw64 := []int64{0, 1, 9, 10, 99, -1, math.MinInt64, 0x0123_4567_89AB_CDEF, 1_000_000_000_000_000_000}
	for i := 1; i <= 13; i++ {
		denom := pow5_64(i)
		mu := int64(const_BARRETT_POW5_MU[i])
		for _, dw := range dw64 {
			r := divRem_q64r64_64x64x64_mu_barrettStep(dw, denom, mu)
			if uint64(r.quotDw0) != uint64(dw)/uint64(denom) || uint64(r.remDw0) != uint64(dw)%uint64(denom) {
				t.Errorf("%d / 5^%d", uint64(dw), i)
			}
		}
	}
}

func TestBarrettDivRemPow10BarrettPow10_exactQuotientAndRemainder(t *testing.T) {
	x128 := [][2]int64{{0, 0}, {1, 0}, {9, 0}, {10, 0}, {-1, 0}, {-1, -1}, {0x0123_4567_89AB_CDEF, 0x0000_0000_DEAD_BEEF}, {-1, 0x0000_FFFF_FFFF_FFFF}}
	for _, x := range x128 {
		bx := big128(x[0], x[1])
		for p := 1; p <= 13; p++ {
			q, m := new(big.Int).QuoRem(bx, ten(p), new(big.Int))
			r := divPow10_q128r64_128_barrett(x[0], x[1], p)
			if big128(r.quotDw0, r.quotDw1).Cmp(q) != 0 || ubig(r.remDw0).Cmp(m) != 0 {
				t.Errorf("%v / 10^%d", bx, p)
			}
		}
	}
}
