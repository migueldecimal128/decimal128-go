package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

var bigPow128 = new(big.Int).Lsh(big.NewInt(1), 128)

// randBigDigits returns a uniform random integer with exactly d decimal digits.
func randBigDigits(r *rand.Rand, d int) *big.Int {
	lo := pow10big(d - 1)
	span := new(big.Int).Sub(pow10big(d), lo)
	buf := make([]byte, 34)
	r.Read(buf)
	rnd := new(big.Int).SetBytes(buf)
	rnd.Mod(rnd, span)
	return rnd.Add(rnd, lo)
}

// TestRRMP10 walks every record in rrmp10Params and checks divPow10_q*res_*_rrmp10
// against math/big for dividends across the record's [qMin, qMax] digit range.
func TestRRMP10(t *testing.T) {
	r := rand.New(rand.NewSource(70))
	records := 0
	cases := 0

	pIndex := 1
	for pIndex < len(rrmp10Params) {
		descriptor := rrmp10Params[pIndex]
		mLen := unpackMDwordLen(descriptor)
		qMin := unpackQMin(descriptor)
		qMax := unpackQMax(descriptor)
		k := unpackK(descriptor)
		is128 := qMin <= 38
		div := pow10big(k)
		records++

		d := qMin
		for d <= qMax {
			trial := 0
			for trial < 16 {
				trial++
				x := randBigDigits(r, d)
				if is128 && x.Cmp(bigPow128) >= 0 {
					continue // does not fit the u128 dividend
				}
				quotBig, remBig := new(big.Int).QuoRem(x, div, new(big.Int))
				if quotBig.BitLen() > 128 {
					continue // entry demands the quotient fit 128 bits
				}

				var got quot128Residue
				if is128 {
					xu := u128FromBig(x)
					got = divPow10_q128res_128_rrmp10(xu.dw0, xu.dw1, pIndex, k)
				} else {
					xu := u256FromBig(x)
					got = divPow10_q128res_256_rrmp10(xu.dw0, xu.dw1, xu.dw2, xu.dw3, pIndex, k)
				}
				gotQ := bigU128(u128{got.quotDw0, got.quotDw1})
				if gotQ.Cmp(quotBig) != 0 {
					t.Fatalf("pIndex=%d k=%d d=%d: quotient got %s want %s (x=%s)", pIndex, k, d, gotQ, quotBig, x)
				}
				if got.residue != refResidue(remBig, div) {
					t.Fatalf("pIndex=%d k=%d d=%d: residue got %s want %s (x=%s)", pIndex, k, d,
						residueName(got.residue), residueName(refResidue(remBig, div)), x)
				}
				cases++
			}
			d++
		}
		pIndex += 1 + mLen
	}

	t.Logf("RRMP10: %d records, %d cases checked", records, cases)
	if records < 60 {
		t.Fatalf("expected to walk all RRMP10 records, only saw %d", records)
	}
}
