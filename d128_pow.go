// IEEE 754-2019 §9.2 pow(x, y): x raised to a decimal power. Integer exponents
// delegate to pown; the finite-nonzero transcendental core computes
// x^y = exp10(y · log10(x)) entirely in the 38-digit D38 intermediate (a single
// final round), then applies the GDAS preferred exponent ⌊y·Q(x)⌋.

package decimal128

import "math"

var d128PowOne = d128_fromInt64(1)

const (
	qMaxPow  = 6111
	qTinyPow = -6176
)

// --- ladder ---

func d128_pow_tte(x, y d128) d128            { return d128_pow_rnd_ctxnull(x, y, ROUND_TIES_TO_EVEN, nil) }
func d128_pow_rnd(x, y d128, rnd round) d128 { return d128_pow_rnd_ctxnull(x, y, rnd, nil) }
func d128_pow_ctx(x, y d128, ctx *context) d128 {
	return d128_pow_rnd_ctxnull(x, y, ctx.round, ctx)
}

func d128_pow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Integer exponent → pown. Covers y = 0 (→ 1 for any x), every integer
	// power, and zero/inf/nan bases with integer y (pown owns those classes).
	if d128_isExactIntegral(y) {
		n := d128_toInt64TowardZero_ctxnull(y, nil)
		if n > math.MinInt32 && n <= math.MaxInt32 {
			return d128_pown_rnd_ctxnull(x, int(n), rounding, ctxnull)
		}
		// |y| too large for pown: only a magnitude-1 base stays finite.
		if d128_isFNZ(x) && cmpMagFnzFnz(x, d128PowOne) == 0 {
			if d128_signFlag(x) && d128_isOddIntegral(y) {
				return d128_fromInt64(-1)
			}
			return d128_fromInt64(1)
		}
		// else fall through: a huge integer y overflows/underflows in the matrix.
	}

	// NaN operands (y is not an exact integer here).
	if d128_isNAN(x) || d128_isNAN(y) {
		// pow(1, qNaN) = 1 with no exception (§9.2).
		if d128_isFNZ(x) && !d128_signFlag(x) && d128_isQNAN(y) && cmpMagFnzFnz(x, d128PowOne) == 0 {
			return d128_fromInt64(1)
		}
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}

	// x, y finite or infinite; y is non-integer (y may be ∞).
	if d128_isZER(x) {
		if d128_isINF(y) {
			// pow(±0, ±∞): y > 0 → +0, y < 0 → +∞ (no divideByZero, per §9.2).
			if d128_signFlag(y) {
				return D128_POS_INFINITY
			}
			return d128_zero(false, 0)
		}
		return powZerFnz(y, ctxnull)
	}
	if d128_isINF(x) {
		if d128_isINF(y) {
			return powFnzInf(x, y)
		}
		return powInfFnz(y)
	}
	// x finite non-zero.
	if d128_isINF(y) {
		return powFnzInf(x, y)
	}
	return powFnzFnz(x, y, rounding, ctxnull)
}

// --- special classes (non-integer y) ---

// powZerFnz: pow(±0, y), y finite non-integer. Sign is always positive.
func powZerFnz(y d128, ctxnull *context) d128 {
	if d128_signFlag(y) {
		return ctxSignalDivByZero_ctxnull(ctxnull, false) // y < 0 → +∞, divideByZero
	}
	return d128_zero(false, 0) // y > 0 → +0
}

// powInfFnz: pow(±∞, y), y finite non-integer.
func powInfFnz(y d128) d128 {
	if d128_signFlag(y) {
		return d128_zero(false, 0) // y < 0 → +0
	}
	return D128_POS_INFINITY // y > 0 → +∞
}

// powFnzInf: pow(x, ±∞), x finite non-zero or ±∞.
func powFnzInf(x, y d128) d128 {
	if d128_isFNZ(x) && cmpMagFnzFnz(x, d128PowOne) == 0 {
		return d128_fromInt64(1) // pow(±1, ±∞) = 1
	}
	xLessThanOne := d128_isFNZ(x) && d128_sciExp(x) < 0
	yNeg := d128_signFlag(y)
	if xLessThanOne != yNeg {
		return d128_zero(false, 0)
	}
	return D128_POS_INFINITY
}

// --- finite non-zero core ---

func powFnzFnz(x, y d128, rounding round, ctxnull *context) d128 {
	// Negative base with non-integer exponent → invalid.
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causePowerOfNegBaseNonIntegerExponent)
	}
	xd := d38Pack(false, d128_qExp(x), x.ubdLo64, d128_hi49(x))
	yd := d38Pack(d128_signFlag(y), d128_qExp(y), y.ubdLo64, d128_hi49(y))

	// x^y = exp10(y · log10(x)), full 38-digit precision through the product.
	log10X := d38_logMagnitude(xd, true)
	yLog10X := d38_multiply(yd, log10X)
	result := d128_exp10_from_d38(yLog10X, rounding, ctxnull)

	// GDAS preferred exponent ⌊y·Q(x)⌋ (only for a finite-nonzero result).
	if d128_isFNZ(result) {
		return applyPreferredScale(result, powPreferredQExp(x, y, yd))
	}
	return result
}

func powPreferredQExp(x, y d128, yd d38) int {
	qX := d128_qExp(x)
	if qX == 0 {
		return 0
	}
	if d128_sciExp(y) >= 5 {
		if d128_signFlag(y) {
			return qTinyPow
		}
		return qMaxPow
	}
	prod := d38_multiply(yd, d38_fromInt32(qX))
	floored := d38_roundToIntegralTowardNegative(prod)
	l := d38_toLongOrMin(floored)
	return int(maxI64(minI64(l, qMaxPow), qTinyPow))
}

// applyPreferredScale re-expresses a finite-nonzero result with quantum exponent
// preferredQExp when the cohort allows — pad trailing zeros (if there is
// precision headroom) or strip them. A no-op on a full-precision inexact result.
// Value-preserving, raises no flags.
func applyPreferredScale(x d128, preferredQExp int) d128 {
	xQ := d128_qExp(x)
	qDelta := xQ - preferredQExp
	if qDelta == 0 {
		return x
	}
	if qDelta > 0 {
		// Lower the exponent toward preferred by padding trailing zeros.
		digitLen := d128_digitLen(x)
		headroom := min(34-digitLen, xQ-qTinyPow)
		if qDelta <= headroom {
			c := d128_u128(x)
			scaled := mulPow10_128_128(c.dw0, c.dw1, qDelta)
			return decimalFNZ_64x64(d128_signFlag(x), xQ-qDelta, scaled.dw0, scaled.dw1)
		}
		return x
	}
	// qDelta < 0: raise the exponent toward preferred by stripping trailing zeros.
	return d128_stripTrailingZeros(x, -qDelta)
}

// --- integrality predicates ---

// d128_isExactIntegral reports whether x is an exact integer (finite, no
// fractional part).
func d128_isExactIntegral(x d128) bool {
	if !d128_isFIN(x) {
		return false
	}
	if d128_isZER(x) {
		return true
	}
	q := d128_qExp(x)
	if q >= 0 {
		return true
	}
	drop := -q
	digitLen := d128_digitLen(x)
	if drop >= digitLen {
		return false // |x| < 1, nonzero
	}
	qr := div_128_pow10_q128res(d128_u128(x), drop)
	return qr.residue == RESIDUE_EXACT
}

// d128_isOddIntegral reports whether x is an odd integer. The caller guarantees
// x is an exact integer.
func d128_isOddIntegral(x d128) bool {
	q := d128_qExp(x)
	if q > 0 {
		return false // a multiple of 10
	}
	c := d128_u128(x)
	if q == 0 {
		return (c.dw0 & 1) != 0
	}
	qr := div_128_pow10_q128res(c, -q)
	return (qr.quotDw0 & 1) != 0
}
