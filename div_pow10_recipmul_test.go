package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestDivMagic(t *testing.T) {
	r := rand.New(rand.NewSource(60))
	ten := big.NewInt(10)
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		bx := bigU128(x)
		wantQ, wantR := new(big.Int).QuoRem(bx, ten, new(big.Int))
		got := div1e1_q128res_128_magic(x.dw0, x.dw1)
		if bigU128(u128{got.quotDw0, got.quotDw1}).Cmp(wantQ) != 0 {
			t.Fatalf("div1e1 quotient: x=%s got %s want %s", bx, bigU128(u128{got.quotDw0, got.quotDw1}), wantQ)
		}
		if got.residue != refResidue(wantR, ten) {
			t.Fatalf("div1e1 residue: got %s want %s", residueName(got.residue), residueName(refResidue(wantR, ten)))
		}
	}
}

func TestDivPow10Recipmul(t *testing.T) {
	r := rand.New(rand.NewSource(61))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		bx := bigU128(x)
		p := r.Intn(38) + 1 // 1..38
		div := pow10big(p)
		wantQ, wantR := new(big.Int).QuoRem(bx, div, new(big.Int))

		got := divPow10_q128res_128_recipmul(x.dw0, x.dw1, p)
		if bigU128(u128{got.quotDw0, got.quotDw1}).Cmp(wantQ) != 0 {
			t.Fatalf("recipmul quotient: x=%s 10^%d got %s want %s",
				bx, p, bigU128(u128{got.quotDw0, got.quotDw1}), wantQ)
		}
		if got.residue != refResidue(wantR, div) {
			t.Fatalf("recipmul residue: x=%s 10^%d got %s want %s",
				bx, p, residueName(got.residue), residueName(refResidue(wantR, div)))
		}
	}
}
