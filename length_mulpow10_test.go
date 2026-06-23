package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func pow10big(p int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p)), nil)
}

func TestLengthFamily(t *testing.T) {
	r := rand.New(rand.NewSource(40))
	for i := 0; i < oracleIters; i++ {
		x128 := randU128(r)
		b128 := bigU128(x128)
		if got, want := bitLen_int_128(x128.dw0, x128.dw1), b128.BitLen(); got != want {
			t.Fatalf("bitLen_int_128(%s)=%d want %d", b128, got, want)
		}
		if b128.Sign() != 0 {
			if got, want := digitLen_int_128(x128.dw0, x128.dw1), len(b128.String()); got != want {
				t.Fatalf("digitLen_int_128(%s)=%d want %d", b128, got, want)
			}
		}

		x256 := randU256(r)
		b256 := bigU256(x256)
		if got, want := bitLen_int_256(x256.dw0, x256.dw1, x256.dw2, x256.dw3), b256.BitLen(); got != want {
			t.Fatalf("bitLen_int_256(%s)=%d want %d", b256, got, want)
		}
		if b256.Sign() != 0 {
			if got, want := digitLen_int_256(x256.dw0, x256.dw1, x256.dw2, x256.dw3), len(b256.String()); got != want {
				t.Fatalf("digitLen_int_256(%s)=%d want %d", b256, got, want)
			}
		}

		x32 := int(r.Int31())
		if x32 > 0 {
			want := len(big.NewInt(int64(x32)).String())
			if got := digitLen_int_32(x32); got != want {
				t.Fatalf("digitLen_int_32(%d)=%d want %d", x32, got, want)
			}
		}
	}
}

func TestMulPow10_128(t *testing.T) {
	r := rand.New(rand.NewSource(41))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		p := r.Intn(39)
		got := mulPow10_128_128(x.dw0, x.dw1, p)
		want := new(big.Int).And(new(big.Int).Mul(bigU128(x), pow10big(p)), bigMask128)
		if bigU128(got).Cmp(want) != 0 {
			t.Fatalf("mulPow10_128_128(x, 10^%d): got %s want %s", p, bigU128(got), want)
		}
	}
}

func TestMulPow10_256_128(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		p := r.Intn(78)
		prod := new(big.Int).Mul(bigU128(x), pow10big(p))
		if prod.BitLen() > 256 {
			continue // out of contract
		}
		got := mulPow10_256_128(x.dw0, x.dw1, p)
		check256(t, "mulPow10_256_128", got, prod)
	}
}

func TestMulPow10_256_256(t *testing.T) {
	r := rand.New(rand.NewSource(43))
	tested := 0
	for i := 0; i < 8*oracleIters; i++ {
		// Bound x < 2^118 so the 256x256 path's column accumulator never
		// overflows (see u256_mul.go header).
		xbits := r.Intn(119)
		x := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), uint(256-xbits)))
		p := r.Intn(78)
		prod := new(big.Int).Mul(bigU256(x), pow10big(p))
		if prod.BitLen() > 256 {
			continue
		}
		got := mulPow10_256_256(x.dw0, x.dw1, x.dw2, x.dw3, p)
		check256(t, "mulPow10_256_256", got, prod)
		tested++
	}
	if tested < 1000 {
		t.Fatalf("too few in-contract cases exercised: %d", tested)
	}
}

func TestFusedMulPow10Add(t *testing.T) {
	r := rand.New(rand.NewSource(44))
	pow19 := pow10big(19)
	for i := 0; i < oracleIters; i++ {
		pow10 := r.Intn(20) // 0..19
		// x with at most 77-pow10 digits so digitLen(x)+pow10 <= 77.
		bound := pow10big(77 - pow10)
		x := u256FromBig(new(big.Int).Mod(bigU256(randU256(r)), bound))
		addend := new(big.Int).Mod(bigU256(randU256(r)), pow19).Uint64()

		got := fusedMulPow10Add_256_256x64(x, pow10, addend)
		want := new(big.Int).Add(new(big.Int).Mul(bigU256(x), pow10big(pow10)), new(big.Int).SetUint64(addend))
		check256(t, "fusedMulPow10Add", got, want)
	}
}

func TestDoubleBridges(t *testing.T) {
	// Exact small integers round-trip through float64.
	for _, v := range []uint64{0, 1, 2, 1000, 1 << 20, (1 << 53) - 1} {
		x := u256{int64(v), 0, 0, 0}
		if got := u256ToFloorDouble(x); got != float64(v) {
			t.Fatalf("u256ToFloorDouble(%d)=%v want %v", v, got, float64(v))
		}
		got := u128FromDouble(float64(v))
		if uint64(got.dw0) != v || got.dw1 != 0 {
			t.Fatalf("u128FromDouble(%d)=%v", v, got)
		}
	}
	// Floor behavior: result <= x and within one ulp-of-53-bits below.
	r := rand.New(rand.NewSource(45))
	for i := 0; i < 1000; i++ {
		x := randU256(r)
		bx := new(big.Float).SetInt(bigU256(x))
		f := u256ToFloorDouble(x)
		bf := big.NewFloat(f)
		if bf.Cmp(bx) > 0 {
			t.Fatalf("u256ToFloorDouble overshoot: %v > %s", f, bigU256(x))
		}
	}
}
