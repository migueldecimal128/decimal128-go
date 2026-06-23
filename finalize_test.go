package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

var allRoundings = []round{TIES_TO_EVEN, TIES_TO_AWAY, TOWARD_ZERO, TOWARD_POSITIVE, TOWARD_NEGATIVE}

// refFinalize is the correctly-rounded decimal128 of (sign, coeff·10^qExp) with
// the given trailing residue, for the in-range case (no overflow/underflow).
// Returns (eQExp, eCoeff); coeff must be nonzero.
func refFinalize(sign bool, coeff *big.Int, qExp int, inRes residue, rnd round) (int, *big.Int) {
	d := len(coeff.String())
	excess := d - 34
	var q *big.Int
	var res residue
	var eQExp int
	if excess > 0 {
		div := pow10big(excess)
		q = new(big.Int)
		rem := new(big.Int)
		q.QuoRem(coeff, div, rem)
		res = residueMerge(refResidue(rem, div), inRes)
		eQExp = qExp + excess
	} else {
		q = new(big.Int).Set(coeff)
		res = inRes
		eQExp = qExp
	}
	if res != RESIDUE_EXACT && refNeedsRoundUp(rnd.code, sign, q.Bit(0) == 1, res) {
		q.Add(q, big.NewInt(1))
		if q.Cmp(pow10big(34)) == 0 {
			q = pow10big(33)
			eQExp++
		}
	}
	return eQExp, q
}

func TestFinalizeRoundPath(t *testing.T) {
	r := rand.New(rand.NewSource(91))
	for i := 0; i < oracleIters; i++ {
		s := r.Intn(2) == 1
		d := r.Intn(76) + 1
		coeff := randBigDigits(r, d)
		qExp := r.Intn(201) - 100
		inRes := residue(r.Intn(4))
		rnd := allRoundings[r.Intn(5)]

		got := finalize_256_ctx(s, qExp, u256FromBig(coeff), inRes, rnd, nil)
		eQExp, eCoeff := refFinalize(s, coeff, qExp, inRes, rnd)

		if !d128_isFNZ(got) {
			t.Fatalf("expected finite nonzero, got class %d (coeff=%s qExp=%d res=%d)", d128_ieeeClass(got), coeff, qExp, inRes)
		}
		if d128_signFlag(got) != s {
			t.Fatalf("sign mismatch")
		}
		if d128_qExp(got) != eQExp {
			t.Fatalf("qExp: got %d want %d (coeff=%s d=%d res=%d %s)", d128_qExp(got), eQExp, coeff, d, inRes, roundName(rnd))
		}
		if bigU128(d128_u128(got)).Cmp(eCoeff) != 0 {
			t.Fatalf("coeff: got %s want %s (in=%s d=%d res=%d %s)", bigU128(d128_u128(got)), eCoeff, coeff, d, inRes, roundName(rnd))
		}
	}
}

func TestFinalizeOverflowClamp(t *testing.T) {
	// Overflow: 34-digit coeff pushed past the max exponent cannot be clamped.
	c34 := pow10U128(33) // 10^33 has 34 digits
	ovf := finalize_128_ctx(false, 6121, c34, RESIDUE_EXACT, TIES_TO_EVEN, nil)
	if !d128_isINF(ovf) || d128_isNegative(ovf) {
		t.Fatalf("expected +Inf on overflow, class=%d", d128_ieeeClass(ovf))
	}
	// Toward-zero overflow yields the max finite magnitude.
	mx := finalize_128_ctx(false, 6121, c34, RESIDUE_EXACT, TOWARD_ZERO, nil)
	if !d128_bitwiseEQ(mx, maxFiniteMagnitude(false)) {
		t.Fatalf("expected max finite on toward-zero overflow")
	}
	// Clamp: small coeff at qExp just past max absorbs the excess (numerically equal).
	cl := finalize_128_ctx(false, 6120, u128{5, 0}, RESIDUE_EXACT, TIES_TO_EVEN, nil)
	if !d128_isFNZ(cl) || d128_qExp(cl) != 6111 {
		t.Fatalf("expected clamp to qExp 6111, got qExp=%d", d128_qExp(cl))
	}
	if bigU128(d128_u128(cl)).Cmp(new(big.Int).Mul(big.NewInt(5), pow10big(9))) != 0 {
		t.Fatalf("clamp coeff wrong: %s", bigU128(d128_u128(cl)))
	}
}

func TestFinalizeUnderflowSubnormal(t *testing.T) {
	// Fully swamped: 1e-6300 underflows to zero under ties-to-even.
	z := finalize_128_ctx(false, -6300, u128{1, 0}, RESIDUE_EXACT, TIES_TO_EVEN, nil)
	if !d128_isZER(z) {
		t.Fatalf("expected zero on full underflow, class=%d", d128_ieeeClass(z))
	}
	// Subnormal: 25e-6177 = 2.5e-6176 rounds to 2e-6176 (ties to even).
	sn := finalize_128_ctx(false, -6177, u128{25, 0}, RESIDUE_EXACT, TIES_TO_EVEN, nil)
	if !d128_isSubnormal(sn) || d128_qExp(sn) != -6176 {
		t.Fatalf("expected subnormal at qExp -6176, got qExp=%d class=%d", d128_qExp(sn), d128_ieeeClass(sn))
	}
	if bigU128(d128_u128(sn)).Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("subnormal coeff: got %s want 2", bigU128(d128_u128(sn)))
	}
}

func TestFinalizeFlags(t *testing.T) {
	// A coeff guaranteed inexact: 10^34 + 7 (35 digits; trimming drops a 7).
	ctx := newContext()
	c := new(big.Int).Add(pow10big(34), big.NewInt(7))
	_ = finalize_256_ctx(false, 0, u256FromBig(c), RESIDUE_EXACT, TIES_TO_EVEN, ctx)
	if !ctx.isSet(INEXACT) || ctx.exceptionLetters() != "x" {
		t.Fatalf("expected inexact flag, got %q", ctx.exceptionLetters())
	}
	// An exact result sets no flags.
	ctx2 := newContext()
	_ = finalize_256_ctx(false, 0, u256FromBig(big.NewInt(12345)), RESIDUE_EXACT, TIES_TO_EVEN, ctx2)
	if ctx2.exceptionLetters() != "" {
		t.Fatalf("expected no flags, got %q", ctx2.exceptionLetters())
	}
}

// refD38 is the ties-to-even rounding of coeff to <=38 digits (D38 form).
func refD38(coeff *big.Int) (int, *big.Int) {
	d := len(coeff.String())
	excess := d - 38
	if excess <= 0 {
		return 0, new(big.Int).Set(coeff)
	}
	div := pow10big(excess)
	q := new(big.Int)
	rem := new(big.Int)
	q.QuoRem(coeff, div, rem)
	res := refResidue(rem, div)
	if res == RESIDUE_GT_HALF || (res == RESIDUE_HALF && q.Bit(0) == 1) {
		q.Add(q, big.NewInt(1))
		if q.Cmp(pow10big(38)) == 0 {
			q = pow10big(37)
			excess++
		}
	}
	return excess, q
}

func TestFinalize38(t *testing.T) {
	r := rand.New(rand.NewSource(92))
	for i := 0; i < oracleIters; i++ {
		s := r.Intn(2) == 1
		d := r.Intn(76) + 1
		coeff := randBigDigits(r, d)
		qExp := r.Intn(201) - 100

		got := finalize38_256(s, qExp, u256FromBig(coeff), RESIDUE_EXACT)
		dExcess, eCoeff := refD38(coeff)
		if d38_signFlag(got) != s {
			t.Fatalf("d38 sign mismatch")
		}
		if d38_qExp(got) != qExp+dExcess {
			t.Fatalf("d38 qExp: got %d want %d (coeff=%s)", d38_qExp(got), qExp+dExcess, coeff)
		}
		if bigU128(d38_u128(got)).Cmp(eCoeff) != 0 {
			t.Fatalf("d38 coeff: got %s want %s (in=%s)", bigU128(d38_u128(got)), eCoeff, coeff)
		}
	}
}

func randBigDigitsAt(d int) *big.Int {
	r := rand.New(rand.NewSource(int64(d) * 7))
	return randBigDigits(r, d)
}
