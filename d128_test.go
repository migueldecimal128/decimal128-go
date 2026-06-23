package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestD128FiniteRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(80))
	p34 := pow10big(34)
	for i := 0; i < oracleIters; i++ {
		sign := r.Intn(2) == 1
		qExp := r.Intn(6111-(-6176)+1) + (-6176) // [-6176, 6111]
		coeffBig := new(big.Int).Add(big.NewInt(1), new(big.Int).Mod(bigU128(randU128(r)), new(big.Int).Sub(p34, big.NewInt(1))))
		cu := u128FromBig(coeffBig)

		x := decimalFNZ_64x64(sign, qExp, cu.dw0, cu.dw1)
		if d128_qExp(x) != qExp {
			t.Fatalf("qExp: got %d want %d", d128_qExp(x), qExp)
		}
		if d128_signFlag(x) != sign {
			t.Fatalf("sign: got %v want %v", d128_signFlag(x), sign)
		}
		if bigU128(d128_u128(x)).Cmp(coeffBig) != 0 {
			t.Fatalf("coeff: got %s want %s", bigU128(d128_u128(x)), coeffBig)
		}
		if !d128_isFinite(x) || !d128_isFNZ(x) || d128_isZER(x) || d128_isNAN(x) || d128_isINF(x) {
			t.Fatalf("finite predicates wrong for %s", coeffBig)
		}
		if d128_digitLen(x) != len(coeffBig.String()) {
			t.Fatalf("digitLen: got %d want %d", d128_digitLen(x), len(coeffBig.String()))
		}
		wantSci := qExp + len(coeffBig.String()) - 1
		if d128_sciExp(x) != wantSci {
			t.Fatalf("sciExp: got %d want %d", d128_sciExp(x), wantSci)
		}
	}
}

func TestD128Zero(t *testing.T) {
	for _, sign := range []bool{false, true} {
		z := d128_zero(sign, 5)
		if !d128_isZER(z) || !d128_isFinite(z) || d128_isFNZ(z) {
			t.Fatalf("zero predicates wrong")
		}
		if d128_qExp(z) != 5 || d128_signFlag(z) != sign {
			t.Fatalf("zero qExp/sign wrong")
		}
		want := CLASS_POSITIVE_ZERO
		if sign {
			want = CLASS_NEGATIVE_ZERO
		}
		if d128_ieeeClass(z) != want {
			t.Fatalf("zero class wrong")
		}
	}
}

func TestD128NonFinite(t *testing.T) {
	posInf := d128_infinity(false)
	negInf := d128_infinity(true)
	if !d128_isINF(posInf) || !d128_isNonFinite(posInf) || d128_isFinite(posInf) || d128_isNAN(posInf) {
		t.Fatalf("posInf predicates wrong")
	}
	if d128_ieeeClass(posInf) != CLASS_POSITIVE_INFINITY || d128_ieeeClass(negInf) != CLASS_NEGATIVE_INFINITY {
		t.Fatalf("inf class wrong")
	}
	if !d128_bitwiseEQ(posInf, D128_POS_INFINITY) || !d128_bitwiseEQ(negInf, D128_NEG_INFINITY) {
		t.Fatalf("inf constants mismatch")
	}

	if !d128_isQNAN(D128_QNAN0) || !d128_isNAN(D128_QNAN0) || d128_isSNAN(D128_QNAN0) {
		t.Fatalf("QNAN0 predicates wrong")
	}
	if !d128_isSNAN(D128_SNAN0) || !d128_isNAN(D128_SNAN0) || d128_isQNAN(D128_SNAN0) {
		t.Fatalf("SNAN0 predicates wrong")
	}
	if d128_ieeeClass(D128_QNAN0) != CLASS_QUIET_NAN || d128_ieeeClass(D128_SNAN0) != CLASS_SIGNALING_NAN {
		t.Fatalf("nan class wrong")
	}
}

func TestD128NanPayload(t *testing.T) {
	r := rand.New(rand.NewSource(81))
	p33 := pow10big(33)
	for i := 0; i < oracleIters; i++ {
		sign := r.Intn(2) == 1
		signaling := r.Intn(2) == 1
		payloadBig := new(big.Int).Mod(bigU128(randU128(r)), p33) // < 10^33
		pu := u128FromBig(payloadBig)

		x := d128_nan_64x64(sign, signaling, pu.dw0, pu.dw1)
		if !d128_isNAN(x) {
			t.Fatalf("not NaN")
		}
		if signaling != d128_isSNAN(x) || (!signaling) != d128_isQNAN(x) {
			t.Fatalf("signaling bit wrong")
		}
		if d128_signFlag(x) != sign {
			t.Fatalf("nan sign wrong")
		}
		if bigU128(d128_nanPayload(x)).Cmp(payloadBig) != 0 {
			t.Fatalf("payload round-trip: got %s want %s", bigU128(d128_nanPayload(x)), payloadBig)
		}
		// quieting an sNaN yields a qNaN with the same payload.
		if signaling {
			q := d128_quieted(x)
			if !d128_isQNAN(q) || bigU128(d128_nanPayload(q)).Cmp(payloadBig) != 0 {
				t.Fatalf("quieted wrong")
			}
		}
	}
}

func TestD128SubnormalNormal(t *testing.T) {
	// Smallest-exponent single-digit coefficient is subnormal.
	sub := decimalFNZ_64x64(false, -6176, 1, 0)
	if !d128_isSubnormal(sub) || d128_isNormal(sub) {
		t.Fatalf("expected subnormal")
	}
	if d128_ieeeClass(sub) != CLASS_POSITIVE_SUBNORMAL {
		t.Fatalf("subnormal class wrong")
	}
	// A mid-range value is normal.
	norm := decimalFNZ_64x64(true, 0, 12345, 0)
	if d128_isSubnormal(norm) || !d128_isNormal(norm) {
		t.Fatalf("expected normal")
	}
	if d128_ieeeClass(norm) != CLASS_NEGATIVE_NORMAL {
		t.Fatalf("normal class wrong")
	}
}

func TestD128FromInt(t *testing.T) {
	for _, n := range []int{0, 1, -1, 42, -42, 1 << 40, -(1 << 40)} {
		x := d128_fromInt(n)
		absBig := big.NewInt(int64(n))
		absBig.Abs(absBig)
		if bigU128(d128_u128(x)).Cmp(absBig) != 0 {
			t.Fatalf("fromInt(%d) coeff = %s want %s", n, bigU128(d128_u128(x)), absBig)
		}
		if d128_signFlag(x) != (n < 0) {
			t.Fatalf("fromInt(%d) sign wrong", n)
		}
		if d128_qExp(x) != 0 {
			t.Fatalf("fromInt(%d) qExp != 0", n)
		}
	}
}
