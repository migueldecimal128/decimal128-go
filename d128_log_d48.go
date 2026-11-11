// ln / log10 over the 48-digit D48 domain (D48GoWorkOrder.md Phase 3; Swift
// D128LogD48.swift): argument reduction x = k·10^e, square-root reduction until
// |c' − 1| < 0.02, then the [11/12] Padé kernel (d48_log_constants.go).

package decimal128

// d48_fromD128 is D128 → D48, exact: a 34-digit coefficient fits the low two
// limbs.
func d48_fromD128(x d128, sign bool) d48 {
	return d48Pack(sign, d128_qExp(x), x.ubdLo64, d128_hi49(x), 0)
}

// d48_padeEval is P(z)/Q(z) by Horner's method with fused multiply-adds.
// Shared by the ln/log10 and exp/exp10 kernels.
func d48_padeEval(zArg d48, p, q []d48) d48 {
	pAcc := p[len(p)-1]
	for i := len(p) - 2; i >= 0; i-- {
		pAcc = d48_fma(pAcc, zArg, p[i])
	}
	qAcc := q[len(q)-1]
	for i := len(q) - 2; i >= 0; i-- {
		qAcc = d48_fma(qAcc, zArg, q[i])
	}
	return d48_divide(pAcc, qAcc)
}

// extractKMostSigDigitRounded is the leading decimal digit of x rounded to
// nearest (ties away), in [1,10]; callers treat 10 as 1 with the exponent
// bumped.
func extractKMostSigDigitRounded(x d48) int {
	digitLen := d48_digitLen(x)
	if digitLen == 1 {
		return int(x.coeffDw0)
	}
	qr := divPow10_checked_q320res(d48_u320(x), digitLen-1)
	d := int(qr.quotDw0)
	if (qr.residue == const_RESIDUE_HALF) || (qr.residue == const_RESIDUE_GT_HALF) {
		d++ // ties away from zero
	}
	return d
}

// d48_logMagnitude is the 48-digit ln / log10 of a positive finite-nonzero
// value, before lowering to D128; pow consumes it directly so y · log10(x)
// keeps full precision. xd is the sign-stripped operand.
func d48_logMagnitude(xd d48, isLog10 bool) d48 {
	// Argument reduction: x = k·10^e with k the rounded leading digit, so
	// c' = x / (k·10^e) ∈ [~0.75, 1.5).
	eExp := d48_sciExp(xd)
	k := extractKMostSigDigitRounded(xd)
	if k == 10 {
		k = 1
		eExp++
	}

	divisor := d48Pack(false, eExp, int64(k), 0, 0)
	cPrime := d48_divide(xd, divisor)

	// Square-root reduction until |c' − 1| < 0.02; ln(c') = 2^N · ln(c'^(1/2^N)).
	zArg := d48_subtract(cPrime, const_D48_ONE)
	sqrtCount := 0
	for d48_cmpMag(zArg, const_D48_Z_THRESHOLD) >= 0 {
		cPrime = d48_sqrt(cPrime)
		zArg = d48_subtract(cPrime, const_D48_ONE)
		sqrtCount++
	}

	pWeights := padeLnPWeights
	if isLog10 {
		pWeights = padeLog10PWeights
	}
	r := d48_padeEval(zArg, pWeights, padeLogQWeights)
	r = d48_multiply(r, zArg) // log(c'^(1/2^N)) ≈ z·P(z)/Q(z)
	twoPowN := d48_fromInt32(1 << sqrtCount)
	eVal := d48_fromInt32(eExp)

	if isLog10 {
		// log10(x) = 2^N·r + log10(k) + e
		r = d48_fma(r, twoPowN, const_D48_LOG10_TABLE[k])
		r = d48_add(r, eVal)
	} else {
		// ln(x) = 2^N·r + ln(k) + e·ln(10)
		r = d48_fma(r, twoPowN, const_D48_LN_TABLE[k])
		r = d48_fma(eVal, const_D48_LN10, r)
	}
	return r
}
