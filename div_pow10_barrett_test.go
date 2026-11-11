package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestDivPow10Barrett128(t *testing.T) {
	r := rand.New(rand.NewSource(50))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		bx := bigU128(x)
		p := r.Intn(const_BARRETT_POW10_MAX) + 1 // 1..13
		div := pow10big(p)
		wantQ, wantR := new(big.Int).QuoRem(bx, div, new(big.Int))

		qr := divPow10_q128r64_128_barrett(x.dw0, x.dw1, p)
		if bigU128(u128{qr.quotDw0, qr.quotDw1}).Cmp(wantQ) != 0 || uint64(qr.remDw0) != wantR.Uint64() {
			t.Fatalf("divPow10_q128r64_128_barrett(x,10^%d): got q=%s r=%d want q=%s r=%s",
				p, bigU128(u128{qr.quotDw0, qr.quotDw1}), uint64(qr.remDw0), wantQ, wantR)
		}

		res := divPow10_q128res_128_barrett(x.dw0, x.dw1, p)
		if bigU128(u128{res.quotDw0, res.quotDw1}).Cmp(wantQ) != 0 {
			t.Fatalf("q128res quotient mismatch")
		}
		if res.residue != refResidue(wantR, div) {
			t.Fatalf("q128res residue: got %s want %s", residueName(res.residue), residueName(refResidue(wantR, div)))
		}
	}
}

func TestDivPow10Barrett256(t *testing.T) {
	r := rand.New(rand.NewSource(51))
	for i := 0; i < oracleIters; i++ {
		x := randU256(r)
		bx := bigU256(x)
		p := r.Intn(const_BARRETT_POW10_MAX) + 1
		div := pow10big(p)
		wantQ, wantR := new(big.Int).QuoRem(bx, div, new(big.Int))

		qr := divPow10_q256r64_256_barrett(x.dw0, x.dw1, x.dw2, x.dw3, p)
		gotQ := bigU256(u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3})
		if gotQ.Cmp(wantQ) != 0 || uint64(qr.remDw0) != wantR.Uint64() {
			t.Fatalf("divPow10_q256r64_256_barrett(x,10^%d): got q=%s r=%d want q=%s r=%s",
				p, gotQ, uint64(qr.remDw0), wantQ, wantR)
		}

		res := divPow10_q256res_256_barrett(x.dw0, x.dw1, x.dw2, x.dw3, p)
		if bigU256(u256{res.quotDw0, res.quotDw1, res.quotDw2, res.quotDw3}).Cmp(wantQ) != 0 {
			t.Fatalf("q256res quotient mismatch")
		}
		if res.residue != refResidue(wantR, div) {
			t.Fatalf("q256res residue mismatch")
		}
	}
}

func refStrip(c *big.Int, maxToStrip int) (*big.Int, int) {
	v := new(big.Int).Set(c)
	count := 0
	ten := big.NewInt(10)
	for count < maxToStrip {
		q, rem := new(big.Int).QuoRem(v, ten, new(big.Int))
		if rem.Sign() != 0 {
			break
		}
		v = q
		count++
	}
	return v, count
}

func TestStrip128(t *testing.T) {
	r := rand.New(rand.NewSource(53))
	for i := 0; i < oracleIters; i++ {
		z := r.Intn(20) // number of trailing zeros to attach
		base := new(big.Int).Mod(bigU128(randU128(r)), pow10big(38-z))
		if base.Sign() == 0 {
			base = big.NewInt(int64(r.Intn(1000) + 1))
		}
		c := new(big.Int).Mul(base, pow10big(z))
		if c.Sign() == 0 || c.BitLen() > 128 {
			continue
		}
		maxToStrip := r.Intn(40)
		wantV, wantC := refStrip(c, maxToStrip)

		cu := u128FromBig(c)
		got := strip_s128sc_128(cu.dw0, cu.dw1, maxToStrip)
		gotV := bigU128(u128{got.strippedDw0, got.strippedDw1})
		if gotV.Cmp(wantV) != 0 || int(got.stripCount) != wantC {
			t.Fatalf("strip(%s, max=%d): got v=%s c=%d want v=%s c=%d",
				c, maxToStrip, gotV, got.stripCount, wantV, wantC)
		}
	}
}
