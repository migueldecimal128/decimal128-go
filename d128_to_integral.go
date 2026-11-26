// IEEE 754-2019 Section 5.3.1 roundToIntegral{*} and Section 5.4.1 convertToInteger{*}
// for int64 and uint64, plus the range predicates fitsInt64 / fitsUInt64.
// roundToIntegral comes in all five directions, each as a quiet form (no flags), a
// _ctx form, and a SignalInexact form that raises inexact when the result differs from
// the operand; in the shared body, suppressInexact tells them apart. The int64
// out-of-range / non-finite sentinel is math.MinInt64 (the Intel bid128_to_int64_*
// convention).

package decimal128

import "math"

const minInt64 int64 = -1 << 63

// --- roundToIntegral{Rule} (value form; no inexact signal) ---

// Returns the integral value with the same sign as `x`, rounded
// using `tiesToEven`. Equivalent to `roundToIntegral_rnd(.ROUND_TIES_TO_EVEN)`.
//
// IEEE 754-2019 Section 5.3.1 `roundToIntegralTiesToEven`.
func d128_roundToIntegral(x d128) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil, true)
}

// d128_roundToIntegral_rnd returns the integral value with the same sign as `x`, rounded
// in the given direction. No exception flags are raised.
//
// This is the "value" form of IEEE 754-2019 Section 5.3.1
// `roundToIntegralTiesToEven` etc.; the "exact" form (which signals
// `inexact`) is available on `Context`.
func d128_roundToIntegral_rnd(x d128, rounding round) d128 {
	return roundToIntegral128_ctxnull(x, rounding, nil, true)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralTiesToEven`.
func d128_roundToIntegralTiesToEven(x d128) d128 {
	return d128_roundToIntegral_rnd(x, const_ROUND_TIES_TO_EVEN)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralTiesToAway`.
func d128_roundToIntegralTiesToAway(x d128) d128 {
	return d128_roundToIntegral_rnd(x, const_ROUND_TIES_TO_AWAY)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralTowardZero`.
func d128_roundToIntegralTowardZero(x d128) d128 {
	return d128_roundToIntegral_rnd(x, const_ROUND_TOWARD_ZERO)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralTowardPositive`.
func d128_roundToIntegralTowardPositive(x d128) d128 {
	return d128_roundToIntegral_rnd(x, const_ROUND_TOWARD_POSITIVE)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralTowardNegative`.
func d128_roundToIntegralTowardNegative(x d128) d128 {
	return d128_roundToIntegral_rnd(x, const_ROUND_TOWARD_NEGATIVE)
}

func d128_roundToIntegralTiesToEven_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TIES_TO_EVEN, ctx, true)
}
func d128_roundToIntegralTiesToAway_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TIES_TO_AWAY, ctx, true)
}
func d128_roundToIntegralTowardZero_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TOWARD_ZERO, ctx, true)
}
func d128_roundToIntegralTowardPositive_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TOWARD_POSITIVE, ctx, true)
}
func d128_roundToIntegralTowardNegative_ctx(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TOWARD_NEGATIVE, ctx, true)
}

// --- roundToIntegralExact (SignalInexact form) ---

// IEEE 754-2019 Section 5.3.1 `roundToIntegralExact` under `tiesToEven`,
// exposed here as `roundToIntegralTiesToEvenSignalInexact`.
func d128_roundToIntegralTiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TIES_TO_EVEN, ctxnull, false)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralExact` under `tiesToAway`,
// exposed here as `roundToIntegralTiesToAwaySignalInexact`.
func d128_roundToIntegralTiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TIES_TO_AWAY, ctxnull, false)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralExact` under `towardZero`,
// exposed here as `roundToIntegralTowardZeroSignalInexact`.
func d128_roundToIntegralTowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TOWARD_ZERO, ctxnull, false)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralExact` under `towardPositive`,
// exposed here as `roundToIntegralTowardPositiveSignalInexact`.
func d128_roundToIntegralTowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TOWARD_POSITIVE, ctxnull, false)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralExact` under `towardNegative`,
// exposed here as `roundToIntegralTowardNegativeSignalInexact`.
func d128_roundToIntegralTowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) d128 {
	return roundToIntegral128_ctxnull(x, const_ROUND_TOWARD_NEGATIVE, ctxnull, false)
}

// IEEE 754-2019 Section 5.3.1 `roundToIntegralExact`, exposed here as
// `roundToIntegralSignalInexact`. Rounds to integral using the
// supplied context's ambient rounding direction and signals
// `inexact` when the result differs from `x`.
func d128_roundToIntegralSignalInexact(x d128, ctx *context) d128 {
	return roundToIntegral128_ctxnull(x, ctx.round, ctx, false)
}

// Round `x` to an integral value in the given direction.
//
// - Parameters:
//   - x: The operand. May be any `D128` (finite, zero, infinity, NaN).
//   - rounding: The rounding direction to apply.
//   - ctx: If non-nil, flags are signaled into `ctx.flags`.
//     If nil, no flags are signaled (Quiet).
//   - suppressInexact: If `true`, the `inexact` flag is not raised even
//     when rounding discards non-zero fractional digits. Matches the
//     IEEE 754-2019 Section 5.3.1 distinction between `roundToIntegral{Rule}`
//     (no inexact signal) and the `Exact` variants (signals inexact).
//
// - Returns: The rounded integral value, preserving the sign of `x`.
//
// NaN propagation honors signaling NaNs: when `ctx` is provided, an sNaN
// operand raises `invalidOperation` and the quieted NaN is returned;
// under Quiet, the same quieting happens but no flag is set.
//
// Non-finite inputs and inputs that are already integral (`qExp >= 0`)
// are returned unchanged.
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
			res = const_RESIDUE_LT_HALF
		} else {
			xU128 := d128_u128(x)
			res = residueFromDecadeWithDigitLen_128(xU128.dw0, xU128.dw1, xDigitLen)
			verify("", func() bool { return res != const_RESIDUE_EXACT })
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
	ctxForFinalize := ctxnull
	if suppressInexact {
		ctxForFinalize = nil
	}
	return finalize_128_ctxnull(xSign, 0, coeff, qr.residue, rounding, ctxForFinalize)
}

// --- convertToInteger64 ---

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTiesToEven`.
func d128_toInt64TiesToEven_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TIES_TO_EVEN, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTiesToAway`.
func d128_toInt64TiesToAway_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TIES_TO_AWAY, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTowardZero`.
func d128_toInt64TowardZero_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TOWARD_ZERO, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTowardPositive`.
func d128_toInt64TowardPositive_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TOWARD_POSITIVE, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTowardNegative`.
func d128_toInt64TowardNegative_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TOWARD_NEGATIVE, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTiesToEven`.
func d128_toInt64TiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TIES_TO_EVEN, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTiesToAway`.
func d128_toInt64TiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TIES_TO_AWAY, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTowardZero`.
func d128_toInt64TowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TOWARD_ZERO, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTowardPositive`.
func d128_toInt64TowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TOWARD_POSITIVE, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTowardNegative`.
func d128_toInt64TowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) int64 {
	return d128ConvertToInt64_ctxnull(x, const_ROUND_TOWARD_NEGATIVE, false, ctxnull)
}

// Core implementation for all decimal-to-Int64 conversions.
//
// Implements the IEEE 754-2019 `convertToInteger` family for signed
// 64-bit integers. The invalid sentinel `Int64.min` matches Intel's
// `bid128_to_int64_*` convention.
//
// - Parameters:
//   - x: The operand.
//   - rounding: Round mode applied when `x` has a fractional part.
//   - suppressInexact: If `true`, the `inexact` flag is not raised even
//     when rounding discards non-zero fractional digits (used by the
//     non-`Exact` IEEE 754 variants). `invalidOperation` is always
//     raised regardless of this flag.
//   - ctx: If non-nil, flags are accumulated into `ctx.flags`. If nil,
//     no flags are signaled (Quiet).
//   - Returns: The converted `Int64`, or `Int64.min` if `x` is NaN,
//     infinite, or out of range.
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
					res = const_RESIDUE_LT_HALF
				} else {
					xU128 := d128_u128(x)
					res = residueFromDecadeWithDigitLen_128(xU128.dw0, xU128.dw1, xDigitLen)
					verify("", func() bool { return res != const_RESIDUE_EXACT })
				}
				roundUp := needsRoundUp(rounding, sign, false, res)
				var ret int64
				if !roundUp {
					ret = 0
				} else {
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
					if !suppressInexact && res != const_RESIDUE_EXACT && ctxnull != nil {
						ctxSignalInexact_ctxnull(ctxnull, x)
					}
					return ret
				}
			}
		}
	}
	ctxSignalInvalidOperation_ctxnull(ctxnull, causeConvertNonFiniteToInteger, const_D128_QNAN0)
	return minInt64
}

// --- range predicates ---

// `true` iff `x` is finite and `math.MinInt64 <= x <= math.MaxInt64`.
func d128_fitsInt64(x d128) bool {
	if !d128_isFinite(x) {
		return false
	}
	lo := d128_fromInt64(math.MinInt64)
	hi := d128_fromInt64(math.MaxInt64)
	return d128_isLessEqual(lo, x) && d128_isLessEqual(x, hi)
}

// `true` iff `x` is finite, non-negative in value, and `x <= math.MaxUint64`.
// Non-negativity is `isZero || !isNegative` (cheap bit tests -- no compare
// against a constructed zero), so negative zero (`-0`) fits (it converts to
// `0`); only a negative *nonzero* value is rejected.
func d128_fitsUInt64(x d128) bool {
	if !d128_isFinite(x) {
		return false
	}
	nonNegative := d128_isZER(x) || !d128_isNegative(x)
	hi := d128_fromUInt64(math.MaxUint64)
	return nonNegative && d128_isLessEqual(x, hi)
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

// IEEE 754-2019 Section 5.4.1 `convertToInteger` for unsigned 64-bit integers.
//
// Rounds `x` to an integral value in `rounding`'s direction, then requires the
// result to lie in `[0, UInt64.max]`. Rounding first makes the range test
// exact and *direction-dependent*: e.g. a value in `(-1, 0]` rounds to `0`
// (valid) while `-0.7` toward-negative rounds to `-1` (invalid), and
// `2^64 - 0.5` fits toward-zero but overflows toward-positive.
//
// - Parameters:
//   - x: The operand.
//   - rounding: Round mode applied when `x` has a fractional part.
//   - suppressInexact: If `true`, `inexact` is not raised even when rounding
//     discards a non-zero fraction (the non-`Exact` IEEE 754 variants).
//     `invalidOperation` is always raised on the out-of-range/non-finite path,
//     and -- matching IEEE's mutual exclusion -- `inexact` is never raised there.
//   - ctxnull: If non-nil, flags accumulate into `ctxnull.flags`; if nil, no
//     flags are signaled (Quiet).
//   - Returns: The converted `UInt64`, or `0` if `x` is NaN, infinite, or out of
//     range.
func d128ConvertToUInt64_ctxnull(x d128, rounding round, suppressInexact bool, ctxnull *context) uint64 {
	// Round to integral QUIETLY in the requested direction (no flags yet), then
	// range-check the rounded result. NaN/infinity survive rounding as non-finite
	// and are rejected by fitsUInt64, so the out-of-range branch is the single
	// INVALID signal (an sNaN is quieted here without a separate signal — the
	// invalid it deserves is raised below). This is the ONLY place invalid is
	// raised, and inexact is NEVER raised on this path (IEEE mutual exclusion).
	t := d128_roundToIntegral_rnd(x, rounding)
	if !d128_fitsUInt64(t) {
		ctxSignalInvalidOperation_ctxnull(ctxnull, causeConvertNonFiniteToInteger, const_D128_QNAN0)
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

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTiesToEven` (unsigned).
func d128_toUInt64TiesToEven_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TIES_TO_EVEN, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTiesToAway` (unsigned).
func d128_toUInt64TiesToAway_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TIES_TO_AWAY, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTowardZero` (unsigned).
//
// Truncate `x` toward zero to a `uint64`; `0` + `invalidOperation` when NaN,
// infinite, negative (magnitude >= 1), or `>= 2^64`.
func d128_toUInt64TowardZero_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TOWARD_ZERO, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTowardPositive` (unsigned).
func d128_toUInt64TowardPositive_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TOWARD_POSITIVE, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerTowardNegative` (unsigned).
func d128_toUInt64TowardNegative_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TOWARD_NEGATIVE, true, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTiesToEven` (unsigned).
func d128_toUInt64TiesToEvenSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TIES_TO_EVEN, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTiesToAway` (unsigned).
func d128_toUInt64TiesToAwaySignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TIES_TO_AWAY, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTowardZero` (unsigned).
func d128_toUInt64TowardZeroSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TOWARD_ZERO, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTowardPositive` (unsigned).
func d128_toUInt64TowardPositiveSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TOWARD_POSITIVE, false, ctxnull)
}

// IEEE 754-2019 Section 5.4.1 `convertToIntegerExactTowardNegative` (unsigned).
func d128_toUInt64TowardNegativeSignalInexact_ctxnull(x d128, ctxnull *context) uint64 {
	return d128ConvertToUInt64_ctxnull(x, const_ROUND_TOWARD_NEGATIVE, false, ctxnull)
}
