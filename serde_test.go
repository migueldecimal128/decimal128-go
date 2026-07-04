package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

// randD128 builds a random canonical d128 across all classes.
func randD128(r *rand.Rand) d128 {
	switch r.Intn(6) {
	case 0:
		return d128_zero(r.Intn(2) == 1, r.Intn(12288)-6176)
	case 1:
		return d128_infinity(r.Intn(2) == 1)
	case 2:
		payload := new(big.Int).Mod(bigU128(randU128(r)), pow10big(33))
		pu := u128FromBig(payload)
		return d128_nan_64x64(r.Intn(2) == 1, r.Intn(2) == 1, pu.dw0, pu.dw1)
	default:
		sign := r.Intn(2) == 1
		qExp := r.Intn(12288) - 6176 // [-6176, 6111]
		d := r.Intn(34) + 1
		cu := u128FromBig(randBigDigits(r, d))
		return decimalFNZ_64x64(sign, qExp, cu.dw0, cu.dw1)
	}
}

func TestBIDRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(100))
	for i := 0; i < oracleIters; i++ {
		d := randD128(r)
		rt := d128_from_bid128(d128_to_bid128(d))
		if !d128_bitwiseEQ(rt, d) {
			t.Fatalf("BID round-trip failed: class=%d hi=%#x lo=%#x -> hi=%#x lo=%#x",
				d128_ieeeClass(d), d.ubdHi64, d.ubdLo64, rt.ubdHi64, rt.ubdLo64)
		}
	}
}

func TestDPDRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(101))
	for i := 0; i < oracleIters; i++ {
		d := randD128(r)
		rt := d128_from_dpd128(d128_to_dpd128(d))
		if !d128_bitwiseEQ(rt, d) {
			t.Fatalf("DPD round-trip failed: class=%d hi=%#x lo=%#x -> hi=%#x lo=%#x",
				d128_ieeeClass(d), d.ubdHi64, d.ubdLo64, rt.ubdHi64, rt.ubdLo64)
		}
	}
}

func TestSerdeKnownVectors(t *testing.T) {
	one := decimalFNZ_64x64(false, 0, 1, 0)
	posZero := d128_zero(false, 0)

	// Canonical decimal128 encodings of 1 and +0E+0.
	bidOne := d128_to_bid128(one)
	if bidOne.dw1 != int64(0x3040000000000000) || bidOne.dw0 != 1 {
		t.Fatalf("BID(1) = hi=%#x lo=%#x", bidOne.dw1, bidOne.dw0)
	}
	dpdOne := d128_to_dpd128(one)
	if dpdOne.dw1 != int64(0x2208000000000000) || dpdOne.dw0 != 1 {
		t.Fatalf("DPD(1) = hi=%#x lo=%#x", dpdOne.dw1, dpdOne.dw0)
	}
	bidZero := d128_to_bid128(posZero)
	if bidZero.dw1 != int64(0x3040000000000000) || bidZero.dw0 != 0 {
		t.Fatalf("BID(+0E0) = hi=%#x lo=%#x", bidZero.dw1, bidZero.dw0)
	}
	dpdZero := d128_to_dpd128(posZero)
	if dpdZero.dw1 != int64(0x2208000000000000) || dpdZero.dw0 != 0 {
		t.Fatalf("DPD(+0E0) = hi=%#x lo=%#x", dpdZero.dw1, dpdZero.dw0)
	}

	// Decode back.
	if !d128_bitwiseEQ(d128_from_bid128(bidOne), one) || !d128_bitwiseEQ(d128_from_dpd128(dpdOne), one) {
		t.Fatalf("decode of 1 mismatch")
	}
}

func TestSerdeHexParsers(t *testing.T) {
	wantOne := u128{dw0: 1, dw1: int64(0x3040000000000000)}
	if got, ok := bid128ParseIntelHex("[3040000000000000,0000000000000001]"); !ok || got != wantOne {
		t.Fatalf("bid Intel comma form: %v ok=%v", got, ok)
	}
	if got, ok := bid128ParseIntelHex("[30400000000000000000000000000001]"); !ok || got != wantOne {
		t.Fatalf("bid Intel single form: %v ok=%v", got, ok)
	}
	if _, ok := bid128ParseIntelHex("[zz400000000000000000000000000001]"); ok {
		t.Fatalf("bad hex should fail")
	}
	if _, ok := bid128ParseIntelHex("not a vector"); ok {
		t.Fatalf("bad length should fail")
	}

	wantDpdOne := u128{dw0: 1, dw1: int64(0x2208000000000000)}
	if got, ok := dpd128ParseDecTestHex("#22080000000000000000000000000001"); !ok || got != wantDpdOne {
		t.Fatalf("dpd dectest form: %v ok=%v", got, ok)
	}
	if _, ok := dpd128ParseDecTestHex("22080000000000000000000000000001"); ok {
		t.Fatalf("missing # should fail")
	}
}
