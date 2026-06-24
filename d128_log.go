// Natural log and base-10 log. The finite non-zero magnitude is computed in the
// 38-digit D38 domain via a [9,9] Padé rational approximation after argument
// reduction, then lowered to D128 with the caller's rounding. Faithful port of
// the Padé scheme (near-last-digit accuracy), not a correctly-rounded redesign.

package decimal128

// --- ln ladder ---

func d128_ln_tte(x d128) d128            { return d128_ln_rnd_ctxnull(x, TIES_TO_EVEN, nil) }
func d128_ln_rnd(x d128, rnd round) d128 { return d128_ln_rnd_ctxnull(x, rnd, nil) }
func d128_ln_ctx(x d128, ctx *context) d128 {
	return d128_ln_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_ln_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logDispatch(x, false, rounding, ctxnull)
}

// --- log10 ladder ---

func d128_log10_tte(x d128) d128            { return d128_log10_rnd_ctxnull(x, TIES_TO_EVEN, nil) }
func d128_log10_rnd(x d128, rnd round) d128 { return d128_log10_rnd_ctxnull(x, rnd, nil) }
func d128_log10_ctx(x d128, ctx *context) d128 {
	return d128_log10_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_log10_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logDispatch(x, true, rounding, ctxnull)
}

// --- dispatch (IEEE 754-2019 §9.2 special cases) ---

func logDispatch(x d128, isLog10 bool, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			// log(±0) = −∞, signalling division-by-zero.
			return ctxSignalDivByZero(ctxnull, true)
		}
		if d128_signFlag(x) {
			return ctxSignalInvalidOperation(ctxnull, causeLogOfNegNumber)
		}
		if isLog10 && isExactPowerOf10(x) {
			// log10(10^n) = n exactly — no rounding, no inexact flag.
			return d128_fromInt(d128_sciExp(x))
		}
		return logImplFNZ(x, isLog10, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1(ctxnull, x)
	}
	verify("", d128_isINF(x))
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation(ctxnull, causeLogOfNegNumber)
	}
	return D128_POS_INFINITY // log(+∞) = +∞
}

// --- magnitude ---

func logImplFNZ(x d128, isLog10 bool, rounding round, ctxnull *context) d128 {
	xd := d38Pack(false, d128_qExp(x), x.ubdLo64, d128_hi49(x))
	r := logMagnitudeD38(xd, isLog10)

	result := finalize_128_ctx(d38_signFlag(r), d38_qExp(r), d38_u128(r), RESIDUE_EXACT, rounding, ctxnull)

	// A non-zero ln/log10 of a finite value is irrational — always INEXACT. The
	// final round usually sets it, but when the 38-digit intermediate lands on an
	// exact-looking value the round sees no residue, so force the flag. (r == 0
	// is the lone exact case, ln(1) = 0; log10 exacts are power-of-10 fast-pathed.)
	if d38_isZero(r) {
		return result
	}
	if ctxnull != nil {
		return ctxSignalInexact(ctxnull, result)
	}
	return result
}

// logMagnitudeD38 is the 38-digit ln / log10 of a positive FNZ value, as a D38
// before lowering to D128. logImplFNZ finalizes this; pow consumes it directly
// so the y·log10(x) product keeps full precision. xd is the sign-stripped operand.
func logMagnitudeD38(xd d38, isLog10 bool) d38 {
	// Argument reduction: x = k·10^e with k the rounded leading digit, so
	// c' = x / (k·10^e) ∈ [~0.75, 1.5).
	eExp := d38_sciExp(xd)
	k := extractKMostSigDigitRounded(xd)
	if k == 10 {
		k = 1
		eExp++
	}

	divisor := d38Pack(false, eExp, int64(k), 0)
	cPrime := d38_divide(xd, divisor)

	// Variable square-root reduction: repeatedly take √ — each halves the
	// distance to 1 — until z = c'−1 is small enough for the [9,9] Padé to be
	// correct to ≥34 digits. ln(c') = 2^N·ln(c'^(1/2^N)), so the reconstruction
	// multiplier is 2^N.
	zArg := d38_subtract(cPrime, D38_ONE)
	sqrtCount := 0
	for cmpMagD38(zArg, Z_THRESHOLD) >= 0 {
		cPrime = d38_sqrt(cPrime)
		zArg = d38_subtract(cPrime, D38_ONE)
		sqrtCount++
	}

	pWeights := padeLnPWeights
	if isLog10 {
		pWeights = padeLog10PWeights
	}
	r := padeEval(zArg, pWeights, padeLogQWeights)
	r = d38_multiply(r, zArg) // log(c'^(1/2^N)) ≈ z·P(z)/Q(z)
	twoPowN := d38_fromInt(1 << sqrtCount)
	eVal := d38_fromInt(eExp)

	if isLog10 {
		// log10(x) = 2^N·r + log10(k) + e
		r = d38_fma(r, twoPowN, LOG10_TABLE[k])
		r = d38_add(r, eVal)
	} else {
		// ln(x) = 2^N·r + ln(k) + e·ln(10)
		r = d38_fma(r, twoPowN, LN_TABLE[k])
		r = d38_fma(eVal, LN10, r)
	}
	return r
}

// extractKMostSigDigitRounded returns the leading decimal digit of x rounded to
// nearest (ties away), in [1,10]; callers treat 10 as 1 with the exponent bumped.
func extractKMostSigDigitRounded(x d38) int {
	digitLen := d38_digitLen(x)
	if digitLen == 1 {
		return int(x.coeffLo64)
	}
	qr := div_128_pow10_checked_q128res(d38_u128(x), digitLen-1)
	d := qr.quotDw0
	if qr.residue == RESIDUE_HALF || qr.residue == RESIDUE_GT_HALF {
		d++ // ties away from zero
	}
	return int(d)
}

// isExactPowerOf10 reports whether x is exactly an integer power of ten (1·10^n)
// — the log10 fast path.
func isExactPowerOf10(x d128) bool {
	digitLen := d128_digitLen(x)
	c := d128_u128(x)
	return cmp_int_128x128_pow10(c.dw0, c.dw1, digitLen-1) == 0
}
