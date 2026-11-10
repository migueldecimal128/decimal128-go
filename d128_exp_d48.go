// exp / exp10 over the 48-digit D48 domain (D48GoWorkOrder.md Phase 3; Swift
// D128ExpD48.swift): range reduction, then exp(r) = exp(r/8)^8 via the [12/12]
// Padé kernel (d48_exp_constants.go) and three squarings. exp's reduction uses
// ln10 at 90 digits in u320.

package decimal128

import "math"

// scaled is a D48 value z scaled by 10^n: the exp / exp10 result before the
// final round.
type scaled struct {
	z d48
	n int
}

// expScaled_d48 is exp(x) = z·10^n via range reduction + [12/12] Padé of r/8 +
// three squarings. The caller has checked sciExp(x) < 6, so n = round(x/ln10)
// is small.
func expScaled_d48(x d128) scaled {
	xd := d48_fromD128(x, d128_signFlag(x))

	// Range reduction: n = round(x/ln10), r = x − n·ln10 (ln10 at 90 digits).
	nDec := d48_roundToIntegralTiesToAway(d48_divide(xd, d48_LN10))
	n := int(d48_toLongOrMin(nDec))
	r := xd
	if n != 0 {
		// n·ln10 < 10^96 for |n| <= 4.4e5; r = x − n·ln10 via the wide signed add.
		nMag := n
		if n < 0 {
			nMag = -n
		}
		nLn10 := mul_320_320x64(u320_LN10_WIDE_COEFF, int64(nMag))
		r = d48_addWide(d48_signFlag(xd), d48_qExp(xd), d48_u320(xd), n > 0, LN10_WIDE_QEXP, nLn10)
	}

	// exp(r) = exp(r/8)^8 -- three squarings.
	rPrime := d48_divide(r, d48_EIGHT)
	e := d48_padeEval(rPrime, padeExpPWeights, padeExpQWeights)
	e = d48_square(e)
	e = d48_square(e)
	e = d48_square(e)
	return scaled{z: e, n: n}
}

// exp10Scaled_d48 is 10^xd = z·10^n for a non-integer xd with |round(xd)| <= 6200.
func exp10Scaled_d48(xd d48) scaled {
	nDec := d48_roundToIntegralTiesToAway(xd)
	nLong := int(d48_toLongOrMin(nDec))
	r := d48_subtract(xd, nDec)
	rPrime := d48_divide(r, d48_EIGHT)
	z := d48_padeEval(rPrime, padeExp10PWeights, padeExp10QWeights)
	z = d48_square(z)
	z = d48_square(z)
	z = d48_square(z)
	return scaled{z: z, n: nLong}
}

// --- Rounding into D128 ---

// e_EXP_D48 is the error bound of the D48 exp / exp10 kernels in units of the
// last digit: the measured maximum on transc_truth.txt is 42 (d48_bounds_test.go);
// ×250 slack.
var e_EXP_D48 = u320FromDw0(10_000)

// finalizeScaled rounds exp(z)^8 scaled by 10^n once into decimal128; a
// non-special exp / exp10 / pow result is irrational, hence Inexact.
func finalizeScaled(e d48, n int, rounding round, ctxnull *context) d128 {
	result := finalize_256_ctxnull(false, d48_qExp(e)+n, u256{e.coeffDw0, e.coeffDw1, e.coeffDw2, 0},
		RESIDUE_EXACT, rounding, ctxnull)
	return d48_forceInexact(result, ctxnull)
}

func ambiguousScaled(s scaled, e u320) bool {
	return d48_ambiguous(d48_u320(s.z), d48_qExp(s.z)+s.n, e)
}

// exp10Trivial is 10^xd when it needs no kernel: an exact integer power, or
// certain overflow / underflow. ok = false otherwise.
func exp10Trivial(xd d48, rounding round, ctxnull *context) (d128, bool) {
	l := d48_toLongOrMin(xd)
	if l != math.MinInt64 {
		if l > 6144 {
			return finalizeOverflow_ctxnull(false, rounding, ctxnull), true
		}
		if l < -6176 {
			return finalizeUnderflowFullySwamped(false, rounding, ctxnull), true
		}
		return finalize_128_ctxnull(false, int(l), u128{1, 0}, RESIDUE_EXACT, rounding, ctxnull), true
	}
	nLong := d48_toLongOrMin(d48_roundToIntegralTiesToAway(xd))
	if nLong == math.MinInt64 {
		if d48_signFlag(xd) {
			return finalizeUnderflowFullySwamped(false, rounding, ctxnull), true
		}
		return finalizeOverflow_ctxnull(false, rounding, ctxnull), true
	}
	if nLong > 6200 {
		return finalizeOverflow_ctxnull(false, rounding, ctxnull), true
	}
	if nLong < -6200 {
		return finalizeUnderflowFullySwamped(false, rounding, ctxnull), true
	}
	return d128{}, false
}

// expImplFNZ_d48 is exp(x) for a finite non-zero x. Results within 10^4 units
// of a rounding boundary go to the tiny-argument rule; one it doesn't decide is
// rounded as is (the documented 1-ulp gap, CrossPlatformArchitecture.md).
func expImplFNZ_d48(x d128, rounding round, ctxnull *context) d128 {
	// |x| ≥ 10^6: exp certainly overflows (x > 0) or underflows to 0 (x < 0).
	if d128_sciExp(x) >= 6 {
		if d128_signFlag(x) {
			return finalizeUnderflowFullySwamped(false, rounding, ctxnull)
		}
		return finalizeOverflow_ctxnull(false, rounding, ctxnull)
	}
	s := expScaled_d48(x)
	if ambiguousScaled(s, e_EXP_D48) && (d128_sciExp(x) < TINY_SCI_EXP) {
		return d48_forceInexact(d48_onePlusTiny(!d128_signFlag(x), rounding, ctxnull), ctxnull)
	}
	return finalizeScaled(s.z, s.n, rounding, ctxnull)
}

// d128_exp10_from_d48 is 10^xd for a D48 argument, rounded once (the exp10
// entry; also tests).
func d128_exp10_from_d48(xd d48, rounding round, ctxnull *context) d128 {
	if t, ok := exp10Trivial(xd, rounding, ctxnull); ok {
		return t
	}
	s := exp10Scaled_d48(xd)
	return finalizeScaled(s.z, s.n, rounding, ctxnull)
}

// exp10ImplFNZ_d48 is exp10(x) for a finite non-zero x; the tiny rule as for
// exp, on u = x·ln10.
func exp10ImplFNZ_d48(x d128, rounding round, ctxnull *context) d128 {
	xd := d48_fromD128(x, d128_signFlag(x))
	if t, ok := exp10Trivial(xd, rounding, ctxnull); ok {
		return t
	}
	s := exp10Scaled_d48(xd)
	if ambiguousScaled(s, e_EXP_D48) && (d128_sciExp(x) < TINY_SCI_EXP-1) {
		return d48_forceInexact(d48_onePlusTiny(!d128_signFlag(x), rounding, ctxnull), ctxnull)
	}
	return finalizeScaled(s.z, s.n, rounding, ctxnull)
}
