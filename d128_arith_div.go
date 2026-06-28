// Divide. The general body dispatches the finite/zero/infinity/NaN cases; the
// FNZ÷FNZ path scales the numerator to 34 extra digits, runs the core 256÷128
// divide with residue, strips exact trailing zeros toward the preferred
// exponent, then finalizes. (divideInteger and remainder land later — they need
// roundToIntegral and fma.)

package decimal128

func d128_divide_tte(x, y d128) d128            { return d128_divide_rnd_ctxnull(x, y, TIES_TO_EVEN, nil) }
func d128_divide_rnd(x, y d128, rnd round) d128 { return d128_divide_rnd_ctxnull(x, y, rnd, nil) }
func d128_divide_ctx(x, y d128, ctx *context) d128 {
	return d128_divide_rnd_ctxnull(x, y, ctx.round, ctx)
}

func d128_divide_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	resultSign := d128_signFlag(x) != d128_signFlag(y)
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return divFnzFnz_rnd_ctx(x, y, resultSign, rounding, ctxnull)
		}
		if d128_isZER(x) && d128_isZER(y) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivZeroByZero)
		}
		if d128_isZER(x) {
			return d128_zero(resultSign, d128_qExp(x)-d128_qExp(y))
		}
		return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", d128_isINF(x) || d128_isINF(y))
	if d128_isINF(x) && d128_isINF(y) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivInfByInf)
	}
	if d128_isINF(y) {
		return d128_zero(resultSign, -6176)
	}
	return d128_infinity(resultSign)
}

// divideInteger never rounds: trunc(x/y) is exact (or NaN/Inf).

func d128_divideInteger(x, y d128) d128 { return d128_divideInteger_ctxnull(x, y, nil) }
func d128_divideInteger_ctx(x, y d128, ctx *context) d128 {
	return d128_divideInteger_ctxnull(x, y, ctx)
}

func d128_divideInteger_ctxnull(x, y d128, ctxnull *context) d128 {
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return divIntFnzFnz_rnd_ctx(x, y, ctxnull)
		}
		resultSign := d128_signFlag(x) != d128_signFlag(y)
		if d128_isZER(x) && d128_isZER(y) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivZeroByZero)
		}
		if d128_isZER(x) {
			return d128_zero(resultSign, 0)
		}
		return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	resultSign := d128_signFlag(x) != d128_signFlag(y)
	verify("", d128_isINF(x) || d128_isINF(y))
	if d128_isINF(x) && d128_isINF(y) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivInfByInf)
	}
	if d128_isINF(x) {
		return d128_infinity(resultSign)
	}
	return d128_zero(resultSign, 0)
}

func divIntFnzFnz_rnd_ctx(x, y d128, ctxnull *context) d128 {
	q1 := d128_divide_rnd_ctxnull(x, y, TOWARD_ZERO, nil) // quiet
	verify("", d128_isFIN(q1))
	q2 := d128_roundToIntegralTowardZeroSignalInexact_ctxnull(q1, nil)
	verify("", d128_isFIN(q2))
	verify("", d128_qExp(q2) >= 0)
	if d128_qExp(q2) == 0 {
		return q2
	}
	if d128_isZER(q2) {
		return d128_zero(d128_signFlag(q2), 0)
	}
	zQ := d128_qExp(q2)
	headroom := 34 - d128_digitLen(q2)
	if headroom < zQ {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivIntOverflowsCoefficient)
	}
	scaled := mulPow10_128_128(q2.ubdLo64, d128_hi49(q2), zQ)
	return decimalFNZ_128(d128_signFlag(q2), 0, scaled.dw0, scaled.dw1)
}

// Remainder never rounds: |r| < |y| (trunc) or |r| <= |y|/2 (near), exact or NaN.

func d128_remainderTrunc(x, y d128) d128 { return d128_remainderTrunc_ctxnull(x, y, nil) }
func d128_remainderTrunc_ctx(x, y d128, ctx *context) d128 {
	return d128_remainderTrunc_ctxnull(x, y, ctx)
}

func d128_remainderTrunc_ctxnull(x, y d128, ctxnull *context) d128 {
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			r, _ := remTruncFnzFnz(x, y, ctxnull)
			return r
		}
		if d128_isZER(x) && d128_isNotZER(y) {
			return d128_zero(d128_signFlag(x), min(d128_qExp(x), d128_qExp(y)))
		}
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivByZeroInRemainderOp)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	if d128_isINF(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeInfNumeratorInRemainderOp)
	}
	return x
}

func d128_remainderNear(x, y d128) d128 { return d128_remainderNear_ctxnull(x, y, nil) }
func d128_remainderNear_ctx(x, y d128, ctx *context) d128 {
	return d128_remainderNear_ctxnull(x, y, ctx)
}

func d128_remainderNear_ctxnull(x, y d128, ctxnull *context) d128 {
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return remainderNearFnzFnz(x, y, ctxnull)
		}
		if d128_isZER(x) && d128_isNotZER(y) {
			return d128_zero(d128_signFlag(x), min(d128_qExp(x), d128_qExp(y)))
		}
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivByZeroInRemainderOp)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	if d128_isINF(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeInfNumeratorInRemainderOp)
	}
	return x
}

// remTruncFnzFnz returns (r = x - trunc(x/y)*y, truncQuotientIsOdd).
func remTruncFnzFnz(x, y d128, ctxnull *context) (d128, bool) {
	truncCtx := newContextRound(TOWARD_ZERO) // scratch: discards flags
	n1 := d128_divide_ctx(x, y, truncCtx)
	if truncCtx.isOverflow() {
		nan := ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivIntOverflowsCoefficient)
		return nan, false
	}
	verify("", d128_isFIN(n1))
	n := n1
	if d128_qExp(n1) < 0 {
		n = d128_roundToIntegralTowardZeroSignalInexact_ctxnull(n1, truncCtx)
	}
	verify("", d128_qExp(n) >= 0)
	// If the integer quotient has more than 34 digits it cannot be represented
	// exactly, so the remainder is undefined (dectest "Division_impossible").
	if d128_digitLen(n)+d128_qExp(n) > 34 {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivIntOverflowsCoefficient), false
	}
	truncIsOdd := false
	if !d128_isZER(n) && d128_qExp(n) == 0 {
		truncIsOdd = (n.ubdLo64 & 1) != 0
	}
	r1 := d128_fma_tte(d128_negate(n), y, x) // r = x - n*y, single rounding
	if d128_isZER(r1) {
		return d128_zero(d128_signFlag(x), min(d128_qExp(x), d128_qExp(y))), truncIsOdd
	}
	return r1, truncIsOdd
}

func remainderNearFnzFnz(x, y d128, ctxnull *context) d128 {
	r1, truncIsOdd := remTruncFnzFnz(x, y, ctxnull)
	if !d128_isFIN(r1) {
		return r1
	}
	if d128_isZER(r1) {
		return r1
	}
	var truncCtx *context
	if ctxnull != nil {
		truncCtx = ctxnull.withRoundingAndNewFlags(TOWARD_ZERO)
	} else {
		truncCtx = newContextRound(TOWARD_ZERO)
	}
	var r2 d128
	if d128_signFlag(r1) {
		r2 = d128_add_ctx(r1, y, truncCtx)
	} else {
		r2 = d128_subtract_ctx(r1, y, truncCtx)
	}
	cmp := cmpMagFnzFnz(r1, r2)
	if cmp > 0 || (cmp == 0 && truncIsOdd) {
		return r2
	}
	return r1
}

// PT fast path: divide by a power of ten (value 10^(m + qExp(y))).
// The result is exact and keeps the dividend's coefficient: Cx * 10^(qPreferred - m),
// where qPreferred = qExp(x) - qExp(y). As in the general exact case the natural
// exponent sits m below the preferred exponent, so strip up to m trailing zeros from
// Cx back toward it (no-op when Cx has none), then let finalize apply the usual
// range/clamp/subnormal handling. Never rounds.
func divPow10Divisor_rnd_ctx(x d128, resultSign bool, qPreferred, m int, rounding round, ctxnull *context) d128 {
	qZ := qPreferred - m
	coefficient := d128_u128(x)
	if m > 0 {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, m)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctxnull(resultSign, qZ, coefficient, RESIDUE_EXACT, rounding, ctxnull)
}

// Quotient-first divide finish for a CD-width divisor (digitLen(y) <= 4; ET is drawn at
// CD width). The caller has integer-divided the coefficients cx / cy -> (qr.quot = qI,
// qr.remDw0 = R0) for single-dword divisor cy, with qI >= 1.
//
// qI supplies the high digitLen(qI) significant digits at the preferred exponent
// qPreferred = qExp(x) - qExp(y); the remaining fracScale = 34 - digitLen(qI) low digits
// come from the remainder alone: qF = (R0 * 10^fracScale) / cy. Since
// cx*10^fracScale = cy*(qI*10^fracScale + qF) + remF, the combined coefficient and remF are
// exactly the truncated quotient and residue a scale-by-fracScale divide would produce, so
// finalize rounds identically -- but the dividend never widened to 256 bits.
//
// R0 == 0: integer-exact, already at the preferred exponent -> no strip. remF == 0 (residue
// EXACT, fracScale > 0): terminating fraction sits fracScale digits below preferred with
// trailing zeros -> strip back toward preferred.
func divQuotientFirstFinish_rnd_ctx(qr quot128Rem64, cy int64, resultSign bool, qPreferred int, rounding round, ctxnull *context) d128 {
	// Exact: qI is already at the preferred exponent -- no strip, no fraction.
	if qr.remDw0 == 0 {
		return finalize_128_ctxnull(resultSign, qPreferred, fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1), RESIDUE_EXACT, rounding, ctxnull)
	}

	qILen := digitLen_int_128(qr.quotDw0, qr.quotDw1)
	fracScale := 34 - qILen
	if fracScale == 0 {
		// qI fills all 34 digits (cy is a single digit), so R0 < cy <= 9 -- within the
		// 64-bit residue classifier's range. The integer remainder is the whole residue.
		res := residueFromRemainderDivisor_64(qr.remDw0, cy)
		return finalize_128_ctxnull(resultSign, qPreferred, fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1), res, rounding, ctxnull)
	}

	// Fractional digits from the remainder only: (R0 * 10^fracScale) / cy.
	scaledRemainder := mulPow10_256_128(qr.remDw0, 0, fracScale)
	frac := u256DivResidue_128(scaledRemainder, fromLoHi_128_64x64(cy, 0))
	qIscaled := mulPow10_128_128(qr.quotDw0, qr.quotDw1, fracScale)
	coefficient := add_128_128x128(qIscaled.dw0, qIscaled.dw1, frac.quotDw0, frac.quotDw1)
	qZ := qPreferred - fracScale
	// A terminating fraction (remF == 0) is exact but fracScale digits below the preferred
	// exponent with trailing zeros -- strip them back toward preferred (capped at fracScale).
	if frac.residue == RESIDUE_EXACT {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, fracScale)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctxnull(resultSign, qZ, coefficient, frac.residue, rounding, ctxnull)
}

func divFnzFnz_rnd_ctx(x, y d128, resultSign bool, rounding round, ctxnull *context) d128 {
	// Power-of-ten divisor fast path (PT). When y is a power of ten,
	// x / y = Cx * 10^(qExp(x) - qExp(y) - m) is exact and coefficient-preserving
	// (m = digitLen(y) - 1), so skip the dividend scale + 256/128 divide and reuse
	// the exact trailing-zero strip + finalize. "Divisor coefficient is a power of ten"
	// is cohort-invariant, so one dword compare detects every representation of y.
	m := d128_digitLen(y) - 1
	yCoeff := d128_u128(y)
	if yCoeff.dw0 == pow10_128_dw0(m) && yCoeff.dw1 == pow10_128_dw1(m) {
		return divPow10Divisor_rnd_ctx(x, resultSign, d128_qExp(x)-d128_qExp(y), m, rounding, ctxnull)
	}

	// Quotient-first path for a CD-width divisor (digitLen(y) <= 4, so m <= 3; ET is
	// drawn at CD width too): integer-divide the coefficients, then fill the fractional
	// digits from the remainder only. The dividend stays 128 bits in the integer step,
	// and an exact quotient (R0 == 0) terminates with no rounding and no strip. Wider
	// divisors (WD/XD) keep the scaled-numerator divide below: there the extra integer
	// divide is pure overhead (the result is rarely exact and the fractional divide is
	// no cheaper). A sub-unit quotient (cx < cy) also falls through.
	if m <= 3 {
		cx := d128_u128(x)
		qr := divRem_q128r64_128x64(cx.dw0, cx.dw1, yCoeff.dw0)
		if !isZero_bool_128(qr.quotDw0, qr.quotDw1) {
			return divQuotientFirstFinish_rnd_ctx(qr, yCoeff.dw0, resultSign, d128_qExp(x)-d128_qExp(y), rounding, ctxnull)
		}
	}

	digitLenDelta := d128_digitLen(x) - d128_digitLen(y)
	numeratorScale := 34 - digitLenDelta
	verify("", numeratorScale > 0)

	scaledNumerator := mulPow10_256_128(x.ubdLo64, d128_hi49(x), numeratorScale)
	qr := u256DivResidue_128(scaledNumerator, d128_u128(y))
	coefficient := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	res := qr.residue

	qPreferred := d128_qExp(x) - d128_qExp(y)
	qZ := qPreferred - numeratorScale

	if res == RESIDUE_EXACT && qZ < qPreferred {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, qPreferred-qZ)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctxnull(resultSign, qZ, coefficient, res, rounding, ctxnull)
}
