// IEEE 754-2019 §5.3.1 roundToIntegral{*} and §5.4.1 convertToInteger{*} (Int64).
// suppressInexact distinguishes the roundToIntegral{Rule} forms (no inexact
// signal) from the Exact/SignalInexact forms. The Int64 out-of-range / non-finite
// sentinel is Int64.min (Intel bid128_to_int64_* convention).

package decimal128

import "math"

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

func d128_roundToIntegral(x d128) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_EVEN, nil, true)
}
func d128_roundToIntegral_rnd(x d128, rounding round) d128 {
	return roundToIntegral128_ctxnull(x, rounding, nil, true)
}

func d128_roundToIntegralTiesToEven(x d128) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_EVEN, nil, true)
}
func d128_roundToIntegralTiesToAway(x d128) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_AWAY, nil, true)
}
func d128_roundToIntegralTowardZero(x d128) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_ZERO, nil, true)
}
func d128_roundToIntegralTowardPositive(x d128) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_POSITIVE, nil, true)
}
func d128_roundToIntegralTowardNegative(x d128) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_NEGATIVE, nil, true)
}

func d128_roundToIntegralTiesToEven_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_EVEN, ctx, true)
}
func d128_roundToIntegralTiesToAway_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_AWAY, ctx, true)
}
func d128_roundToIntegralTowardZero_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_ZERO, ctx, true)
}
func d128_roundToIntegralTowardPositive_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_POSITIVE, ctx, true)
}
func d128_roundToIntegralTowardNegative_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_NEGATIVE, ctx, true)
}

// --- roundToIntegralExact (SignalInexact form) ---

func d128_roundToIntegralTiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_EVEN, ctxnull, false)
}
func d128_roundToIntegralTiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TIES_TO_AWAY, ctxnull, false)
}
func d128_roundToIntegralTowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_ZERO, ctxnull, false)
}
func d128_roundToIntegralTowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_POSITIVE, ctxnull, false)
}
func d128_roundToIntegralTowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, ROUND_TOWARD_NEGATIVE, ctxnull, false)
}
func d128_roundToIntegralSignalInexact(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ctx.round, ctx, false)
}

// --- convertToInteger64 ---

func d128_toInt64TiesToEven_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TIES_TO_EVEN, true, ctxnull)
}
func d128_toInt64TiesToAway_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TIES_TO_AWAY, true, ctxnull)
}
func d128_toInt64TowardZero_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TOWARD_ZERO, true, ctxnull)
}
func d128_toInt64TowardPositive_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TOWARD_POSITIVE, true, ctxnull)
}
func d128_toInt64TowardNegative_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TOWARD_NEGATIVE, true, ctxnull)
}

func d128_toInt64TiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TIES_TO_EVEN, false, ctxnull)
}
func d128_toInt64TiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TIES_TO_AWAY, false, ctxnull)
}
func d128_toInt64TowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TOWARD_ZERO, false, ctxnull)
}
func d128_toInt64TowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TOWARD_POSITIVE, false, ctxnull)
}
func d128_toInt64TowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, ROUND_TOWARD_NEGATIVE, false, ctxnull)
}

// --- range predicates ---

// d128_fitsInt64 reports whether x is finite and lies in the closed real
// interval [Int64.min, Int64.max]. When true, converting x to int64 cannot
// overflow under any rounding direction (there is no rounding parameter).
// NaN and infinity return false.
func d128_fitsInt64(x d128) bool {
	if !d128_isFinite(x) {
		return false
	}
	lo := d128_fromInt64(math.MinInt64)
	hi := d128_fromInt64(math.MaxInt64)
	return d128_isLessEqual(lo, x) && d128_isLessEqual(x, hi)
}

// d128_fitsUInt64 reports whether x is finite, non-negative, and at most
// UInt64.max. When true, converting x to uint64 cannot overflow under any
// rounding direction. Negative zero is accepted; any negative nonzero value
// is rejected. NaN and infinity return false.
func d128_fitsUInt64(x d128) bool {
	if !d128_isFinite(x) {
		return false
	}
	nonNegative := d128_isZER(x) || !d128_isNegative(x)
	hi := d128_fromUInt64(math.MaxUint64)
	return nonNegative && d128_isLessEqual(x, hi)
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

// --- convertToUnsignedInteger64 ---
//
// The unsigned family does NOT reimplement the signed kernel's inline range
// logic. It rounds QUIETLY in the requested direction, then range-checks the
// rounded result with fitsUInt64 — a single branch that covers NaN, infinity, a
// negative-nonzero rounded value, and magnitude >= 2^64. Rounding first makes the
// range test both exact and direction-dependent (e.g. -0.3 -> 0 is valid toward
// zero but -0.7 toward negative -> -1 is not; 2^64-0.5 fits toward zero but
// overflows toward positive). Mirrors Swift d128ConvertToUInt64_ctxnull and the
// signed d128ConvertToInt64_ctxnull entry-point set one-to-one.

func d128ConvertToUInt64_ctxnull(x d128, rounding round, suppressInexact bool, ctxnull *context) uint64 {
	// Round to integral QUIETLY in the requested direction (no flags yet), then
	// range-check the rounded result. NaN/infinity survive rounding as non-finite
	// and are rejected by fitsUInt64, so the out-of-range branch is the single
	// INVALID signal (an sNaN is quieted here without a separate signal — the
	// invalid it deserves is raised below). This is the ONLY place invalid is
	// raised, and inexact is NEVER raised on this path (IEEE mutual exclusion).
	t := d128_roundToIntegral_rnd(x, rounding)
	if !d128_fitsUInt64(t) {
		ctxSignalInvalidOperation_ctxnull(ctxnull, causeConvertNonFiniteToInteger)
		return 0
	}
	// In range. For the Exact variants, signal inexact iff a fraction was
	// discarded — i.e. the integral result differs in VALUE from the operand.
	// isEqual compares values (not cohorts), so 5.00 -> 5 is exact.
	if !suppressInexact && ctxnull != nil && !d128_isEqual(t, x) {
		ctxSignalInexact_ctxnull(ctxnull, x)
	}
	// t is an integral value in [0, MaxUint64]; magnitude coeff*10^qExp with
	// qExp >= 0 and < 2^64, so the product stays in the low dword.
	cu := d128_u128(t)
	qExp := d128_qExp(t)
	// A zero coefficient is zero at any exponent, and a zero can carry a large
	// qExp (e.g. 0E+100) that fitsUInt64 still accepts -> short-circuit before
	// scaling, since pow10_128_dw0 is only defined over the coefficient's digit
	// range. A NONZERO value that fitsUInt64 has qExp <= 19, always in range.
	if cu.dw0 == 0 && cu.dw1 == 0 {
		return 0
	}
	var mag uint64
	if qExp == 0 {
		mag = uint64(cu.dw0)
	} else {
		mag = uint64(cu.dw0) * uint64(pow10_128_dw0(qExp)) // wrapping; exact < 2^64
	}
	return mag
}

func d128_toUInt64TiesToEven_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TIES_TO_EVEN, true, ctxnull)
}
func d128_toUInt64TiesToAway_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TIES_TO_AWAY, true, ctxnull)
}

// d128_toUInt64TowardZero_ctxnull truncates x toward zero to a uint64.
// NaN/inf/negative/overflow -> 0 + invalidOperation; quiet on a discarded
// fraction. Delegates to the shared kernel with (ROUND_TOWARD_ZERO,
// suppressInexact=true); observable behavior is unchanged.
func d128_toUInt64TowardZero_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TOWARD_ZERO, true, ctxnull)
}
func d128_toUInt64TowardPositive_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TOWARD_POSITIVE, true, ctxnull)
}
func d128_toUInt64TowardNegative_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TOWARD_NEGATIVE, true, ctxnull)
}

func d128_toUInt64TiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TIES_TO_EVEN, false, ctxnull)
}
func d128_toUInt64TiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TIES_TO_AWAY, false, ctxnull)
}
func d128_toUInt64TowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TOWARD_ZERO, false, ctxnull)
}
func d128_toUInt64TowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TOWARD_POSITIVE, false, ctxnull)
}
func d128_toUInt64TowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, ROUND_TOWARD_NEGATIVE, false, ctxnull)
}
