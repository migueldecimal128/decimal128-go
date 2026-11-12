// Divide. The general body dispatches the finite/zero/infinity/NaN cases; the
// FNZ÷FNZ path scales the numerator to 34 extra digits, runs the core 256÷128
// divide with residue, strips exact trailing zeros toward the preferred
// exponent, then finalizes. (divideInteger and remainder land later — they need
// roundToIntegral and fma.)

package decimal128

func d128_divide_tte(x, y d128) d128 {
	return d128_divide_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
}
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
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
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
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if d128_isINF(x) && d128_isINF(y) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivInfByInf)
	}
	if d128_isINF(x) {
		return d128_infinity(resultSign)
	}
	return d128_zero(resultSign, 0)
}

func divIntFnzFnz_rnd_ctx(x, y d128, ctxnull *context) d128 {
	q1 := d128_divide_rnd_ctxnull(x, y, const_ROUND_TOWARD_ZERO, nil) // quiet
	verify("", func() bool { return d128_isFIN(q1) })
	q2 := d128_roundToIntegralTowardZeroSignalInexact_ctxnull(q1, nil)
	verify("", func() bool { return d128_isFIN(q2) })
	q2Q := d128_qExp(q2)
	verify("", func() bool { return q2Q >= 0 })
	if q2Q == 0 {
		return q2
	}
	if d128_isZER(q2) {
		return d128_zero(d128_signFlag(q2), 0)
	}
	zQ := q2Q
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
			return remainderTruncFnzFnz(x, y)
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
			return remainderNearFnzFnz(x, y)
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

// RemainderExactWorkOrder.md §2. Both remainders are exact for every finite
// non-zero x and y, however large the quotient x/y (IEEE 754-2019 §5.3.1).
// Everything happens on coefficients at the common exponent
// e = min(qExp(x), qExp(y)):
//
//	X = |x|'s coefficient at e,  h = |y|'s coefficient at e,  r2 = X mod 2h
//
// r2 gives both the truncated remainder (r2 or r2 - h) and the parity of the
// truncated quotient (odd iff r2 >= h), which remainderNear needs for ties.
// When qExp(x) > qExp(y), X = xCoeff * 10^k is never formed: r2 = xCoeff * (10^k mod 2h)
// mod 2h. No quotient is ever formed, and the result's coefficient is below h
// (or below xCoeff when qExp(x) < qExp(y)), so it always fits 34 digits at e.
// Signs are applied last: the result takes the sign of x, or its negation
// when remainderNear rounds the quotient away from zero.

// remQuotientIsZero: whether qExp(x) < qExp(y) and the digit lengths alone show
// |y| > 2|x|: both quotients are 0 and both remainders are x itself.
func remQuotientIsZero(x, y d128) bool {
	d := d128_qExp(y) - d128_qExp(x)
	return (d > 0) && (d128_digitLen(y)+d > d128_digitLen(x)+1)
}

// remDivisorAtMinExp: h, |y|'s coefficient at e = min(qExp(x), qExp(y)).
// Below 10^35 once remQuotientIsZero has been ruled out.
func remDivisorAtMinExp(x, y d128) u128 {
	d := d128_qExp(y) - d128_qExp(x)
	if d <= 0 {
		return d128_u128(y)
	}
	verify("", func() bool { return d <= 34 })
	return mulPow10_128_128(y.ubdLo64, d128_hi49(y), d)
}

// remMod: a mod m.
func remMod(a, m u128) u128 {
	qr := u256DivRem_128(u256{a.dw0, a.dw1, 0, 0}, m)
	return u128{qr.remDw0, qr.remDw1}
}

// remMulMod: (a * b) mod m. The caller guarantees a * b / m < 2^128 (the
// u256DivRem_128 contract): here always a < 2^128 and b < m, or a, b < m.
func remMulMod(a, b, m u128) u128 {
	p := mul_256_128x128(a.dw0, a.dw1, b.dw0, b.dw1)
	qr := u256DivRem_128(p, m)
	return u128{qr.remDw0, qr.remDw1}
}

// remPow10Mod: 10^k mod m, k in 1..12287, m >= 2. k <= 38 reads the table;
// larger k runs left-to-right over 5-bit windows of k: acc <- acc^32 * 10^w
// mod m. Each squaring has acc < m, and each window multiply has 10^w < 2^103,
// so every quotient stays below 2^128.
func remPow10Mod(k int, m u128) u128 {
	if k <= 38 {
		return remMod(pow10U128(k), m)
	}
	shift := 0
	for (k >> shift) > 31 {
		shift += 5
	}
	acc := remMod(pow10U128(k>>shift), m)
	for shift > 0 {
		shift -= 5
		i := 0
		for i < 5 {
			acc = remMulMod(acc, acc, m)
			i += 1
		}
		w := (k >> shift) & 31
		if w != 0 {
			acc = remMulMod(pow10U128(w), acc, m)
		}
	}
	return acc
}

// remCoeffMod2H: r2 = X mod 2h, X = |x|'s coefficient at e = min(qExp(x), qExp(y)).
func remCoeffMod2H(x, y d128, h u128) u128 {
	m := shl_128_128(h.dw0, h.dw1, 1)
	k := d128_qExp(x) - d128_qExp(y)
	if k <= 0 {
		return remMod(d128_u128(x), m)
	}
	return remMulMod(d128_u128(x), remPow10Mod(k, m), m)
}

// remResult: the finite result sign * r * 10^e; a zero r gives a zero of that sign.
func remResult(sign bool, e int, r u128) d128 {
	if isZero_bool_128(r.dw0, r.dw1) {
		return d128_zero(sign, e)
	}
	return decimalFNZ_128(sign, e, r.dw0, r.dw1)
}

// remainderTruncFnzFnz: x - trunc(x/y) * y, exact.
func remainderTruncFnzFnz(x, y d128) d128 {
	if remQuotientIsZero(x, y) {
		return x
	}
	h := remDivisorAtMinExp(x, y)
	r2 := remCoeffMod2H(x, y, h)
	e := min(d128_qExp(x), d128_qExp(y))
	if cmp_int_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1) >= 0 {
		return remResult(d128_signFlag(x), e, sub_128_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1))
	}
	return remResult(d128_signFlag(x), e, r2)
}

// remainderNearFnzFnz: x - n * y, n = x/y rounded to the nearest integer, ties
// to even; exact.
func remainderNearFnzFnz(x, y d128) d128 {
	if remQuotientIsZero(x, y) {
		return x
	}
	h := remDivisorAtMinExp(x, y)
	r2 := remCoeffMod2H(x, y, h)
	e := min(d128_qExp(x), d128_qExp(y))
	truncIsOdd := (cmp_int_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1) >= 0)
	r := r2
	if truncIsOdd {
		r = sub_128_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1)
	}
	// Round the quotient away from zero when r > h/2, or r == h/2 and the
	// truncated quotient is odd: the remainder becomes -(h - r), 0 < h - r < h.
	twoR := shl_128_128(r.dw0, r.dw1, 1)
	cmp := cmp_int_128x128(twoR.dw0, twoR.dw1, h.dw0, h.dw1)
	if (cmp > 0) || ((cmp == 0) && truncIsOdd) {
		hr := sub_128_128x128(h.dw0, h.dw1, r.dw0, r.dw1)
		return decimalFNZ_128(!d128_signFlag(x), e, hr.dw0, hr.dw1)
	}
	return remResult(d128_signFlag(x), e, r)
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
	return finalize_128_ctxnull(resultSign, qZ, coefficient, const_RESIDUE_EXACT, rounding, ctxnull)
}

func divFnzFnz_rnd_ctx(x, y d128, resultSign bool, rounding round, ctxnull *context) d128 {
	// Power-of-ten divisor fast path (PT). When y is a power of ten,
	// x / y = Cx * 10^(qExp(x) - qExp(y) - m) is exact and coefficient-preserving
	// (m = digitLen(y) - 1), so skip the dividend scale + 256/128 divide and reuse
	// the exact trailing-zero strip + finalize. "Divisor coefficient is a power of ten"
	// is cohort-invariant, so one dword compare detects every representation of y.
	m := d128_digitLen(y) - 1
	yCoeff := d128_u128(y)
	qPreferred := d128_qExp(x) - d128_qExp(y)
	if yCoeff.dw0 == pow10_128_dw0(m) && yCoeff.dw1 == pow10_128_dw1(m) {
		return divPow10Divisor_rnd_ctx(x, resultSign, qPreferred, m, rounding, ctxnull)
	}

	// CD-width divisor (digitLen(y) <= 4, so m <= 3). Two sub-paths:
	//   (a) integer-exact: xCoeff / yCoeff has remainder 0, so qI already sits at the preferred
	//       exponent as the canonical cohort member -- return it directly, no scale and
	//       no trailing-zero strip. This is the ET band: one 128/64 divide.
	//   (b) inexact: the scaled dividend is exactly 34 + digitLen(y) <= 38 digits, so it
	//       fits 128 bits, and the divisor fits 64 bits -- the whole correctly-rounded
	//       result is a single 128/64 on the scaled numerator. This is the WD/XD
	//       scaled-numerator path (below) narrowed to 128-bit width; it stays 128 bits
	//       throughout rather than widening the dividend to 256 for a full-width divide.
	if m <= 3 {
		xCoeff := d128_u128(x)
		qint := divRem_q128r64_128x64(xCoeff.dw0, xCoeff.dw1, yCoeff.dw0)
		if qint.remDw0 == 0 && !isZero_bool_128(qint.quotDw0, qint.quotDw1) {
			return finalize_128_ctxnull(resultSign, qPreferred,
				fromLoHi_128_64x64(qint.quotDw0, qint.quotDw1), const_RESIDUE_EXACT, rounding, ctxnull)
		}
		digitLenDeltaCD := d128_digitLen(x) - (m + 1)
		numeratorScaleCD := 34 - digitLenDeltaCD
		verify("", func() bool { return numeratorScaleCD > 0 })
		scaledNumCD := mulPow10_128_128(x.ubdLo64, d128_hi49(x), numeratorScaleCD)
		qr := divRem_q128r64_128x64(scaledNumCD.dw0, scaledNumCD.dw1, yCoeff.dw0)
		residueCD := residueFromRemainderDivisor_64(qr.remDw0, yCoeff.dw0)
		qZcd := qPreferred - numeratorScaleCD
		coefficientCD := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		if residueCD == const_RESIDUE_EXACT && qZcd < qPreferred {
			ss := strip_s128sc_128(coefficientCD.dw0, coefficientCD.dw1, qPreferred-qZcd)
			coefficientCD = u128{ss.strippedDw0, ss.strippedDw1}
			qZcd += int(ss.stripCount)
		}
		return finalize_128_ctxnull(resultSign, qZcd, coefficientCD, residueCD, rounding, ctxnull)
	}

	digitLenDelta := d128_digitLen(x) - (m + 1)
	numeratorScale := 34 - digitLenDelta
	verify("", func() bool { return numeratorScale > 0 })

	scaledNumerator := mulPow10_256_128(x.ubdLo64, d128_hi49(x), numeratorScale)
	qr := u256DivResidue_128(scaledNumerator, yCoeff)
	coefficient := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	res := qr.residue

	qZ := qPreferred - numeratorScale

	if res == const_RESIDUE_EXACT && qZ < qPreferred {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, qPreferred-qZ)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctxnull(resultSign, qZ, coefficient, res, rounding, ctxnull)
}
