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

// TestDivPow10Recipmul256 oracles the 256-bit-dividend kernel against math/big
// over pow10 1..77, with dividends drawn from [2^128, min(10^p·2^128, 2^256)-1]
// so hi != 0 (the 256-bit path) and the quotient still fits in 128 bits.
func TestDivPow10Recipmul256(t *testing.T) {
	r := rand.New(rand.NewSource(62))
	two128 := new(big.Int).Lsh(big.NewInt(1), 128)
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	for i := 0; i < oracleIters; i++ {
		p := r.Intn(77) + 1 // 1..77
		div := pow10big(p)

		// Largest dividend whose quotient still fits in 128 bits (x < 10^p·2^128),
		// capped at 2^256 - 1.
		xMax := new(big.Int).Mul(div, two128)
		if xMax.Cmp(two256) > 0 {
			xMax.Set(two256)
		}
		xMax.Sub(xMax, big.NewInt(1))

		// Uniform bx in [2^128, xMax].
		span := new(big.Int).Sub(xMax, two128)
		span.Add(span, big.NewInt(1))
		bx := new(big.Int).Rand(r, span)
		bx.Add(bx, two128)

		x := u256FromBig(bx)
		wantQ, wantR := new(big.Int).QuoRem(bx, div, new(big.Int))

		got := divPow10_q128res_256_recipmul(x, p)
		gotQ := bigU128(u128{got.quotDw0, got.quotDw1})
		if gotQ.Cmp(wantQ) != 0 {
			t.Fatalf("recipmul256 quotient: x=%s 10^%d got %s want %s", bx, p, gotQ, wantQ)
		}
		if got.residue != refResidue(wantR, div) {
			t.Fatalf("recipmul256 residue: x=%s 10^%d got %s want %s",
				bx, p, residueName(got.residue), residueName(refResidue(wantR, div)))
		}
	}
}
