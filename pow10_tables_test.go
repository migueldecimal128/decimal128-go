package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestPow10Tables(t *testing.T) {
	ten := big.NewInt(10)
	for p := 0; p <= 77; p++ {
		want := new(big.Int).Exp(ten, big.NewInt(int64(p)), nil)
		if p <= 38 {
			got := bigU128(u128{pow10_128_dw0(p), pow10_128_dw1(p)})
			if got.Cmp(want) != 0 {
				t.Fatalf("10^%d (128): got %s want %s", p, got, want)
			}
			if bigU128(pow10U128(p)).Cmp(want) != 0 {
				t.Fatalf("pow10U128(%d) mismatch", p)
			}
		} else {
			i := p - 39
			got := bigU256(u256{pow10_256_dw0(i), pow10_256_dw1(i), pow10_256_dw2(i), pow10_256_dw3(i)})
			if got.Cmp(want) != 0 {
				t.Fatalf("10^%d (256): got %s want %s", p, got, want)
			}
		}
	}
}

func TestPow5Table(t *testing.T) {
	five := big.NewInt(5)
	for p := 0; p <= 13; p++ {
		want := new(big.Int).Exp(five, big.NewInt(int64(p)), nil)
		if pow5_64(p) != want.Int64() {
			t.Fatalf("5^%d: got %d want %s", p, pow5_64(p), want)
		}
	}
}

func TestPow10Compare(t *testing.T) {
	ten := big.NewInt(10)
	r := rand.New(rand.NewSource(30))
	for i := 0; i < 2000; i++ {
		x := randU128(r)
		bx := bigU128(x)
		p := r.Intn(39)
		want := bx.Cmp(new(big.Int).Exp(ten, big.NewInt(int64(p)), nil))
		if got := cmp_int_128x128_pow10(x.dw0, x.dw1, p); got != want {
			t.Fatalf("cmp_int_128x128_pow10(x, 10^%d) = %d want %d", p, got, want)
		}
	}
	p34 := new(big.Int).Exp(ten, big.NewInt(34), nil)
	for i := 0; i < 2000; i++ {
		x := randU128(r)
		bx := bigU128(x)
		if isLt_pow10_34_bool_128(x.dw0, x.dw1) != (bx.Cmp(p34) < 0) {
			t.Fatalf("isLt_pow10_34 wrong")
		}
		if isGe_pow10_34_bool_128(x.dw0, x.dw1) != (bx.Cmp(p34) >= 0) {
			t.Fatalf("isGe_pow10_34 wrong")
		}
	}
}

func TestResiduePow10(t *testing.T) {
	ten := big.NewInt(10)
	r := rand.New(rand.NewSource(31))
	for i := 0; i < oracleIters; i++ {
		// 64-bit form: divisor 10^pow10, pow10 in 0..18 (fits in 64 bits),
		// remainder in [0, divisor).
		p := r.Intn(19)
		div := pow10_128_dw0(p) // 10^p < 2^63 for p <= 18
		rem := int64(r.Uint64() % uint64(div))
		want := refResidue(big.NewInt(rem), big.NewInt(div))
		if got := residueFromRemainderPow10_64(rem, p); got != want {
			t.Fatalf("residuePow10_64(rem=%d, 10^%d) = %s want %s", rem, p, residueName(got), residueName(want))
		}

		// 128-bit form: divisor 10^pow10, pow10 in 0..38, remainder < divisor.
		p2 := r.Intn(39)
		bdiv := new(big.Int).Exp(ten, big.NewInt(int64(p2)), nil)
		brem := new(big.Int).Mod(bigU128(randU128(r)), bdiv)
		rm := u128FromBig(brem)
		want2 := refResidue(brem, bdiv)
		if got := residueFromRemainderPow10_128(rm.dw0, rm.dw1, p2); got != want2 {
			t.Fatalf("residuePow10_128(10^%d) = %s want %s", p2, residueName(got), residueName(want2))
		}
	}
}
