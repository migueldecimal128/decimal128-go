// IEEE 754-2019 §5.3.1 roundToIntegral{*} and §5.4.1 convertToInteger{*} (Int64).
// suppressInexact distinguishes the roundToIntegral{Rule} forms (no inexact
// signal) from the Exact/SignalInexact forms. The Int64 out-of-range / non-finite
// sentinel is Int64.min (Intel bid128_to_int64_* convention).

package decimal128

const minInt64 int64 = -1 << 63

// --- roundToIntegral core ---

func roundToIntegral128_ctxnull(x d128, rounding round, ctxnull *context, suppressInexact bool) d128 {
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	if !d128_isFIN(x) || d128_qExp(x) >= 0 {
		return x
	}
	xSign := d128_signFlag(x)
	if d128_isZER(x) {
		return d128_zero(xSign, 0)
	}
	xDigitLen := d128_digitLen(x)
	fracDigitLen := -d128_qExp(x)

	// Case A: all digits fractional (|x| < 1) → 0 or ±1.
	if fracDigitLen >= xDigitLen {
		var res residue
		if fracDigitLen > xDigitLen {
			res = RESIDUE_LT_HALF
		} else {
			xU128 := d128_u128(x)
			res = residueFromDecadeWithDigitLen_128(xU128.dw0, xU128.dw1, xDigitLen)
			verify("", func() bool { return res != RESIDUE_EXACT })
		}
		var z d128
		if needsRoundUp(rounding, xSign, false, res) {
			z = decimalFNZ_64x64(xSign, 0, 1, 0)
		} else {
			z = d128_zero(xSign, 0)
		}
		if !suppressInexact && ctxnull != nil {
			return ctxSignalInexact_ctxnull(ctxnull, z)
		}
		return z
	}

	// Case B: mixed integral/fractional — scale to qExp 0, finalize rounds.
	qr := div_128_pow10_q128res(d128_u128(x), fracDigitLen)
	coeff := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	var ctxForFinalize *context
	if !suppressInexact {
		ctxForFinalize = ctxnull
	}
	return finalize_128_ctxnull(xSign, 0, coeff, qr.residue, rounding, ctxForFinalize)
}

// --- roundToIntegral{Rule} (value form; no inexact signal) ---

func d128_roundedToIntegral(x d128) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_EVEN, nil, true)
}
func d128_roundedToIntegralRnd(x d128, rounding round) d128 {
	return roundToIntegral128_ctxnull(x, rounding, nil, true)
}

func d128_roundToIntegralTiesToEven(x d128) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_EVEN, nil, true)
}
func d128_roundToIntegralTiesToAway(x d128) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_AWAY, nil, true)
}
func d128_roundToIntegralTowardZero(x d128) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_ZERO, nil, true)
}
func d128_roundToIntegralTowardPositive(x d128) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_POSITIVE, nil, true)
}
func d128_roundToIntegralTowardNegative(x d128) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_NEGATIVE, nil, true)
}

func d128_roundToIntegralTiesToEven_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_EVEN, ctx, true)
}
func d128_roundToIntegralTiesToAway_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_AWAY, ctx, true)
}
func d128_roundToIntegralTowardZero_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_ZERO, ctx, true)
}
func d128_roundToIntegralTowardPositive_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_POSITIVE, ctx, true)
}
func d128_roundToIntegralTowardNegative_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_NEGATIVE, ctx, true)
}

// --- roundToIntegralExact (SignalInexact form) ---

func d128_roundToIntegralTiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_EVEN, ctxnull, false)
}
func d128_roundToIntegralTiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, TIES_TO_AWAY, ctxnull, false)
}
func d128_roundToIntegralTowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_ZERO, ctxnull, false)
}
func d128_roundToIntegralTowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_POSITIVE, ctxnull, false)
}
func d128_roundToIntegralTowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, TOWARD_NEGATIVE, ctxnull, false)
}
func d128_roundToIntegralSignalInexact(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ctx.round, ctx, false)
}

// --- convertToInteger64 ---

func d128_toInt64TiesToEven_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TIES_TO_EVEN, true, ctxnull)
}
func d128_toInt64TiesToAway_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TIES_TO_AWAY, true, ctxnull)
}
func d128_toInt64TowardZero_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TOWARD_ZERO, true, ctxnull)
}
func d128_toInt64TowardPositive_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TOWARD_POSITIVE, true, ctxnull)
}
func d128_toInt64TowardNegative_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TOWARD_NEGATIVE, true, ctxnull)
}

func d128_toInt64TiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TIES_TO_EVEN, false, ctxnull)
}
func d128_toInt64TiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TIES_TO_AWAY, false, ctxnull)
}
func d128_toInt64TowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TOWARD_ZERO, false, ctxnull)
}
func d128_toInt64TowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TOWARD_POSITIVE, false, ctxnull)
}
func d128_toInt64TowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, TOWARD_NEGATIVE, false, ctxnull)
}

func d128ConvertToInt64_ctxnull(x d128, rounding round, suppressInexact bool, ctxnull *context) int64 {
	if d128_isFIN(x) {
		signMask := int64(d128_signMask(x))
		sign := d128_signFlag(x)
		qExp := d128_qExp(x)
		dw0 := x.ubdLo64

		if d128_isZER(x) {
			return 0
		}

		if qExp == 0 {
			coeff := d128_u128(x)
			if cmp_int_128x128(coeff.dw0, coeff.dw1, 0, 1) < 0 { // < 2^64
				w0 := coeff.dw0
				if w0 >= 0 {
					return (w0 ^ signMask) - signMask
				}
				if w0 == minInt64 && sign {
					return minInt64
				}
			}
		} else if qExp > 0 {
			digitLen := d128_digitLen(x)
			if digitLen+qExp <= 19 {
				scaled := dw0 * pow10_128_dw0(qExp)
				if scaled >= 0 {
					return (scaled ^ signMask) - signMask
				}
			}
		} else {
			xDigitLen := d128_digitLen(x)
			fracDigitLen := -qExp
			if fracDigitLen >= xDigitLen {
				var res residue
				if fracDigitLen > xDigitLen {
					res = RESIDUE_LT_HALF
				} else {
					xU128 := d128_u128(x)
					res = residueFromDecadeWithDigitLen_128(xU128.dw0, xU128.dw1, xDigitLen)
					verify("", func() bool { return res != RESIDUE_EXACT })
				}
				var ret int64 = 0
				if needsRoundUp(rounding, sign, false, res) {
					ret = (signMask << 1) | 1 // +1 or -1
				}
				if !suppressInexact && ctxnull != nil {
					ctxSignalInexact_ctxnull(ctxnull, x)
				}
				return ret
			}
			intDigitLen := xDigitLen - fracDigitLen
			if intDigitLen <= 19 {
				qr := div_128_pow10_q128res(d128_u128(x), fracDigitLen)
				res := qr.residue
				verify("", func() bool { return qr.quotDw1 == 0 })
				r0Initial := qr.quotDw0
				isOdd := (r0Initial & 1) != 0
				r0 := r0Initial
				if needsRoundUp(rounding, sign, isOdd, res) {
					r0 = r0Initial + 1
				}
				if r0 >= 0 || (r0 == minInt64 && sign) {
					ret := (r0 ^ signMask) - signMask
					if !suppressInexact && res != RESIDUE_EXACT && ctxnull != nil {
						ctxSignalInexact_ctxnull(ctxnull, x)
					}
					return ret
				}
			}
		}
	}
	ctxSignalInvalidOperation_ctxnull(ctxnull, causeConvertNonFiniteToInteger)
	return minInt64
}
