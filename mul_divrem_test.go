package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestU256Mul128x128(t *testing.T) {
	r := rand.New(rand.NewSource(10))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		y := randU128(r)
		got := mul_256_128x128(x.dw0, x.dw1, y.dw0, y.dw1)
		want := new(big.Int).Mul(bigU128(x), bigU128(y)) // < 2^256, exact
		check256(t, "mul128x128", got, want)
	}
}

func TestU256Mul256x64(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for i := 0; i < oracleIters; i++ {
		// Bound x < 2^192 so x*y (y < 2^64) stays < 2^256.
		x := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), 64))
		y := int64(r.Uint64())
		got := mul_256_256x64(x.dw0, x.dw1, x.dw2, x.dw3, y)
		want := new(big.Int).Mul(bigU256(x), new(big.Int).SetUint64(uint64(y)))
		check256(t, "mul256x64", got, want)
	}
}

func TestU256Mul256x128(t *testing.T) {
	r := rand.New(rand.NewSource(12))
	for i := 0; i < oracleIters; i++ {
		// x is a 128-bit value (dw2=dw3=0); y is 128-bit; product < 2^256.
		xLo := randU128(r)
		x := u256{xLo.dw0, xLo.dw1, 0, 0}
		y := randU128(r)
		got := mul_256_256x128(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1)
		want := new(big.Int).Mul(bigU256(x), bigU128(y))
		check256(t, "mul256x128", got, want)
	}
}

func TestU256Mul256x256(t *testing.T) {
	r := rand.New(rand.NewSource(13))
	for i := 0; i < oracleIters; i++ {
		// Bound operands < 2^120 so no column sum overflows the 128-bit
		// accumulator (see u256_mul.go header) and the product stays < 2^256.
		x := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), 136))
		y := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), 136))
		got := mul_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3)
		want := new(big.Int).Mul(bigU256(x), bigU256(y))
		check256(t, "mul256x256", got, want)
	}
}

func normalizedU64(r *rand.Rand) uint64 { return r.Uint64() | (uint64(1) << 63) }

func TestMGReciprocal64(t *testing.T) {
	r := rand.New(rand.NewSource(14))
	max128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	for i := 0; i < oracleIters; i++ {
		d := normalizedU64(r)
		got := uint64(vReciprocal_64_64_mg(int64(d)))
		want := new(big.Int).And(new(big.Int).Div(max128, new(big.Int).SetUint64(d)), bigMask64)
		if got != want.Uint64() {
			t.Fatalf("vReciprocal_64_64_mg(%d)=%d want %d", d, got, want.Uint64())
		}
	}
}

func TestMGDiv2by1(t *testing.T) {
	r := rand.New(rand.NewSource(15))
	for i := 0; i < oracleIters; i++ {
		d := normalizedU64(r)
		u1 := r.Uint64() % d // u1 < d
		u0 := r.Uint64()
		v := vReciprocal_64_64_mg(int64(d))
		got := divRem_q64r64_128x64x64_v_mg(int64(u1), int64(u0), int64(d), v)

		n := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(u1), 64), new(big.Int).SetUint64(u0))
		bd := new(big.Int).SetUint64(d)
		wantQ, wantR := new(big.Int).DivMod(n, bd, new(big.Int))
		if uint64(got.quotDw0) != wantQ.Uint64() || uint64(got.remDw0) != wantR.Uint64() {
			t.Fatalf("DIV2BY1(u1=%d,u0=%d,d=%d): got q=%d r=%d want q=%s r=%s",
				u1, u0, d, uint64(got.quotDw0), uint64(got.remDw0), wantQ, wantR)
		}
	}
}

func TestMGDiv3by2(t *testing.T) {
	r := rand.New(rand.NewSource(16))
	for i := 0; i < oracleIters; i++ {
		d1 := normalizedU64(r)
		d0 := r.Uint64()
		n2 := r.Uint64() % d1 // (n2:n1) < (d1:d0) since n2 < d1
		n1 := r.Uint64()
		n0 := r.Uint64()
		v := vReciprocal_64_128_mg(int64(d1), int64(d0))
		got := divRem_q64r128_192x128x64_v_mg(int64(n2), int64(n1), int64(n0), int64(d1), int64(d0), v)

		n := new(big.Int).SetUint64(n2)
		n.Lsh(n, 64).Or(n, new(big.Int).SetUint64(n1))
		n.Lsh(n, 64).Or(n, new(big.Int).SetUint64(n0))
		d := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(d1), 64), new(big.Int).SetUint64(d0))
		wantQ, wantR := new(big.Int).DivMod(n, d, new(big.Int))
		gotR := bigU128(u128{got.remDw0, got.remDw1})
		if uint64(got.quotDw0) != wantQ.Uint64() || gotR.Cmp(wantR) != 0 {
			t.Fatalf("DIV3BY2: got q=%d r=%s want q=%s r=%s", uint64(got.quotDw0), gotR, wantQ, wantR)
		}
	}
}

func TestU128DivRem(t *testing.T) {
	r := rand.New(rand.NewSource(17))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		bx := bigU128(x)

		// 128 / 64
		y64 := int64(r.Uint64())
		if y64 == 0 {
			y64 = 1
		}
		q := divRem_q128r64_128x64(x.dw0, x.dw1, y64)
		by := new(big.Int).SetUint64(uint64(y64))
		wq, wr := new(big.Int).DivMod(bx, by, new(big.Int))
		if bigU128(u128{q.quotDw0, q.quotDw1}).Cmp(wq) != 0 || uint64(q.remDw0) != wr.Uint64() {
			t.Fatalf("q128r64 mismatch")
		}

		// 128 / 128
		y := randU128(r)
		if y.dw0 == 0 && y.dw1 == 0 {
			y.dw0 = 1
		}
		byy := bigU128(y)
		q2 := divRem_q128r128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)
		wq2, wr2 := new(big.Int).DivMod(bx, byy, new(big.Int))
		if bigU128(u128{q2.quotDw0, q2.quotDw1}).Cmp(wq2) != 0 {
			t.Fatalf("q128r128 quotient: got %s want %s (x=%s y=%s)",
				bigU128(u128{q2.quotDw0, q2.quotDw1}), wq2, bx, byy)
		}
		if bigU128(u128{q2.remDw0, q2.remDw1}).Cmp(wr2) != 0 {
			t.Fatalf("q128r128 remainder: got %s want %s (x=%s y=%s)",
				bigU128(u128{q2.remDw0, q2.remDw1}), wr2, bx, byy)
		}
	}
}
