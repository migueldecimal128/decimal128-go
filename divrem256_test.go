package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestU256BitLen(t *testing.T) {
	cases := []struct {
		x    u256
		want int
	}{
		{u256{0, 0, 0, 0}, 0},
		{u256{1, 0, 0, 0}, 1},
		{u256{0xFF, 0, 0, 0}, 8},
		{u256{0, 1, 0, 0}, 65},
		{u256{0, 0, 1, 0}, 129},
		{u256{0, 0, 0, 1}, 193},
		{u256{-1, -1, -1, -1}, 256},
	}
	for _, c := range cases {
		if got := c.x.bitLen(); got != c.want {
			t.Errorf("bitLen(%v) = %d, want %d", c.x, got, c.want)
		}
	}
}

func checkDivRem256x64(t *testing.T, x u256, y0 int64) {
	t.Helper()
	got := divRem_q256r64_256x64(x.dw0, x.dw1, x.dw2, x.dw3, y0)
	bx := bigU256(x)
	by := new(big.Int).SetUint64(uint64(y0))
	wantQ, wantR := new(big.Int).DivMod(bx, by, new(big.Int))
	gotQ := bigU256(u256{got.quotDw0, got.quotDw1, got.quotDw2, got.quotDw3})
	if gotQ.Cmp(wantQ) != 0 {
		t.Fatalf("q256r64 quotient: x=%s y=%d got %s want %s", bx, uint64(y0), gotQ, wantQ)
	}
	if uint64(got.remDw0) != wantR.Uint64() {
		t.Fatalf("q256r64 remainder: x=%s y=%d got %d want %s", bx, uint64(y0), uint64(got.remDw0), wantR)
	}
}

func TestU256DivRem64(t *testing.T) {
	r := rand.New(rand.NewSource(20))
	for i := 0; i < oracleIters; i++ {
		x := randU256(r)
		// Cover every dispatch branch with divisors of each width class.
		divisors := []int64{
			1,
			int64(r.Uint64() & 0xFFFF),           // 16-bit (x16 path)
			int64(r.Uint64() & 0xFFFFFFFF),       // up to 32-bit (x32 path)
			int64(r.Uint64()),                    // 33–64-bit (MG path)
			int64(uint64(1) << uint(r.Intn(63))), // power of two
		}
		for _, y0 := range divisors {
			if y0 == 0 {
				y0 = 1
			}
			checkDivRem256x64(t, x, y0)
		}

		// Also a small dividend (bitLen <= 64) against a 33–64-bit divisor.
		small := u256{int64(r.Uint64()), 0, 0, 0}
		checkDivRem256x64(t, small, int64(r.Uint64()|1))
	}
}
