package decimal128

import (
	"math/big"
	"testing"
)

var u256DivCases = []struct {
	name string
	x    [4]int64
	y    string // decimal u128 divisor
}{
	{"y in 64 bits, small", [4]int64{s64(0xDEADBEEFCAFEBABE), 0x12345678, 0x0, 0x0}, "1000003"},
	{"y=1 (identity)", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "1"},
	{"y=UInt64.max, x=max256", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "18446744073709551615"},
	{"x<y, both 128-bit", [4]int64{0x23456789ABCDEF01, 0x1, 0x0, 0x0}, "36893488147419103232"},
	{"x=y, both 128-bit", [4]int64{0x123456789ABCDEF0, s64(0xDEADBEEFCAFEBABE), 0x0, 0x0}, "295990755076957304699390954001009532656"},
	{"x>y, both 128-bit, exact", [4]int64{0x0, 0x5, 0x0, 0x0}, "18446744073709551616"},
	{"x>y, both 128-bit, with remainder", [4]int64{0x7, 0x100, 0x0, 0x0}, "18446744073709551616"},
	{"x=2^128-1, y=2^65", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), 0x0, 0x0}, "36893488147419103232"},
	{"x=2^128-1, y=2^128-1", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), 0x0, 0x0}, "340282366920938463463374607431768211455"},
	{"knuth y=2^65, x=2^200", [4]int64{0x0, 0x0, 0x100, 0x0}, "36893488147419103232"},
	{"knuth y=2^65+1, x=max256", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "36893488147419103233"},
	{"knuth y=2^96-1, x=max256", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "79228162514264337593543950335"},
	{"knuth y=2^96, x=max256 (boundary n=3 vs n=4)", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "79228162514264337593543950336"},
	{"knuth y=2^96+1, x=max256 (just over n=4 boundary)", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "79228162514264337593543950337"},
	{"knuth y=2^127 (top bit only), x=max256", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "170141183460469231731687303715884105728"},
	{"knuth y=UInt128.max, x=max256", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "340282366920938463463374607431768211455"},
	{"knuth y=UInt128.max-1, x=max256", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF)}, "340282366920938463463374607431768211454"},
	{"knuth messy n=3", [4]int64{0xFEDCBA987654321, s64(0xFEDCBA9876543210), 0x123456789ABCDEF0, s64(0xDEADBEEFCAFEBABE)}, "79228162544111796486576700437"},
	{"knuth messy n=4", [4]int64{0x6666777788889999, 0x2222333344445555, s64(0xEEEEFFFF00001111), s64(0xAAAABBBBCCCCDDDD)}, "295990755076957304709479900352868940821"},
	{"knuth y=2^96+3, x=y*7", [4]int64{0x15, 0x0, 0x7, 0x0}, "79228162514264337593543950339"},
	{"knuth y=10^20, x=y*10", [4]int64{0x35C9ADC5DEA00000, 0x36, 0x0, 0x0}, "100000000000000000000"},
	{"knuth y=10^25, x=y*7 (exact)", [4]int64{s64(0x9A8C08FA06000000), 0x39E713, 0x0, 0x0}, "10000000000000000000000000"},
	{"knuth bitLenDelta=1, n=3", [4]int64{s64(0xFFFFFFFFFFFFFFFF), s64(0xFFFFFFFFFFFFFFFF), 0x1, 0x0}, "79228162523487709630398726144"},
	{"knuth bitLenDelta=1, n=4", [4]int64{0x0, 0x0, 0x0, s64(0xFFFFFFFFFFFFFFFF)}, "170141183460469231731687303715884105728"},
}

func u256DivCheck(t *testing.T, x u256, y *big.Int, label string) {
	t.Helper()
	qr := u256DivRem128(x, u128Of(y))
	r := big128(qr.remDw0, qr.remDw1)
	// Remainder strictly less than divisor.
	if r.Cmp(y) >= 0 {
		t.Errorf("remainder >= divisor for %s", label)
	}
	// Round-trip invariant: q * y + r == x.
	got := new(big.Int).Add(new(big.Int).Mul(big256(qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3), y), r)
	if got.Cmp(big256v(x)) != 0 {
		t.Errorf("round-trip failed for %s", label)
	}
}

func TestU256DivTestCase(t *testing.T) {
	for _, tc := range u256DivCases {
		u256DivCheck(t, u256{tc.x[0], tc.x[1], tc.x[2], tc.x[3]}, bigStr(tc.y), tc.name)
	}
}

func TestU256DivDivRem128_randomized(t *testing.T) {
	rng := splitMix64{0x0256_0128_D1F0}
	for i := 0; i < 2_000; i++ {
		x := u256{int64(rng.next()), int64(rng.next()), int64(rng.next()), int64(rng.next())}
		// Random y in (1, 2^128-1]; force at least one bit set.
		y := big128(int64(rng.next()), int64(rng.next()))
		if y.Cmp(big.NewInt(2)) < 0 {
			y = big.NewInt(2)
		}
		u256DivCheck(t, x, y, "random")
	}
}
