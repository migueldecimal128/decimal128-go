// IEEE 754-2019 §9.2 pown(x, n): x raised to a signed integer power.
// Left-to-right binary exponentiation in decimal128 precision, with exact fast
// paths for n ∈ {0, 1, 2} and |x| ∈ {1, 10} plus the special classes.

package decimal128

import "math/bits"

var d128PownOne = d128_fromInt(1)
var d128PownTen = d128_fromInt(10)

// --- ladder ---

func d128_pown_tte(x d128, n int) d128            { return d128_pown_rnd_ctxnull(x, n, TIES_TO_EVEN, nil) }
func d128_pown_rnd(x d128, n int, rnd round) d128 { return d128_pown_rnd_ctxnull(x, n, rnd, nil) }
func d128_pown_ctx(x d128, n int, ctx *context) d128 {
	return d128_pown_rnd_ctxnull(x, n, ctx.round, ctx)
}

func d128_pown_rnd_ctxnull(x d128, n int, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return pownZER(x, n, ctxnull)
		}
		return pownFNZ(x, n, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		// pown(x, 0) is 1 unless x is a signaling NaN (§9.2).
		if n == 0 && d128_isQNAN(x) {
			return d128_fromInt(1)
		}
		return ctxNanOperand1(ctxnull, x)
	}
	verify("", d128_isINF(x))
	return pownINF(x, n)
}

// --- special classes ---

func pownINF(x d128, n int) d128 {
	resultSign := d128_signFlag(x) && (n&1) != 0
	if n > 0 {
		return d128_infinity(resultSign)
	}
	if n < 0 {
		return d128_zero(resultSign, 0)
	}
	return d128_fromInt(1)
}

func pownZER(x d128, n int, ctxnull *context) d128 {
	resultSign := d128_signFlag(x) && (n&1) != 0
	if n > 0 {
		// ±0, exponent scaled by n; clamp the multiplier so the product stays
		// well inside int (d128_zero clamps the exponent into range).
		pClamp := min(n, 9999)
		return d128_zero(resultSign, d128_qExp(x)*pClamp)
	}
	if n < 0 {
		return ctxSignalDivByZero(ctxnull, resultSign)
	}
	return d128_fromInt(1)
}

// --- finite non-zero magnitude ---

func pownFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	if n == 0 {
		return d128_fromInt(1)
	}
	if n == 1 {
		return x
	}
	if n == 2 {
		return d128_square_rnd_ctxnull(x, rounding, ctxnull)
	}

	negative := d128_signFlag(x) && (n&1) != 0

	// |x| == 1: result is ±1 carrying the preferred exponent n·Q(x), clamped to [-33, 0].
	if cmpMagFnzFnz(x, d128PownOne) == 0 {
		resultQExp := int(maxI64(minI64(int64(n)*int64(d128_qExp(x)), 0), -33))
		return d128_oneWithQExp(negative, resultQExp)
	}

	// |x| == 10: an exact power of ten, carrying the preferred cohort exponent.
	if cmpMagFnzFnz(x, d128PownTen) == 0 {
		if n > 6144 {
			return finalizeOverflow_ctx(negative, rounding, ctxnull)
		}
		if n < -6176 {
			return finalizeUnderflowFullySwamped(negative, rounding, ctxnull)
		}
		preferredQExpLong := int64(n) * int64(d128_qExp(x))
		minPreferredQExp := minI64(maxI64(int64(n)-33, -32), 0)
		preferredQExp := int(maxI64(preferredQExpLong, minPreferredQExp))
		return tenPowN(negative, n, preferredQExp, rounding, ctxnull)
	}

	// Negative power: raise to |n| then reciprocate. Choosing raise-first vs
	// reciprocate-first by the estimated result exponent avoids a spurious
	// subnormal underflow on the intermediate.
	if n < 0 {
		absPow := -n
		estimatedResultExp := int64(d128_sciExp(x)) * int64(absPow)
		if estimatedResultExp < 6100 {
			raised := pownFNZ(x, absPow, rounding, ctxnull)
			return d128_divide_rnd_ctxnull(d128PownOne, raised, rounding, ctxnull)
		}
		reciprocal := d128_divide_rnd_ctxnull(d128PownOne, x, rounding, ctxnull)
		return pownFNZ(reciprocal, absPow, rounding, ctxnull)
	}

	return pownFNZ_powGE3(x, n, rounding, ctxnull)
}

func pownFNZ_powGE3(x d128, n int, rounding round, ctxnull *context) d128 {
	verify("", n >= 3)
	resultSign := d128_signFlag(x) && (n&1) != 0
	estimatedResultExp := int64(d128_sciExp(x)) * int64(n)
	if estimatedResultExp > 6200 {
		return finalizeOverflow_ctx(resultSign, rounding, ctxnull)
	}
	if estimatedResultExp < -6250 {
		return finalizeUnderflowFullySwamped(resultSign, rounding, ctxnull)
	}
	// Left-to-right binary exponentiation; the result sign falls out of the
	// multiplies, so no separate sign bookkeeping in the loop.
	m := x
	z := m
	bit := int64(1) << (62 - bits.LeadingZeros64(uint64(n)))
	for bit != 0 {
		z = d128_square_rnd_ctxnull(z, rounding, ctxnull)
		if (int64(n) & bit) != 0 {
			z = d128_multiply_rnd_ctxnull(z, m, rounding, ctxnull)
		}
		bit >>= 1
	}
	return z
}

// --- exact builders ---

// d128_oneWithQExp builds ±1 carrying quantum exponent qExp (in [-33, 0]); the
// coefficient is 10^(-qExp).
func d128_oneWithQExp(sign bool, qExp int) d128 {
	p := -qExp
	return decimalFNZ_128(sign, qExp, pow10_128_dw0(p), pow10_128_dw1(p))
}

// tenPowN builds ±10^n with the preferred cohort exponent preferredQExp;
// finalize clamps the exponent into range / overflows in the large-n branch.
func tenPowN(sign bool, n, preferredQExp int, rounding round, ctxnull *context) d128 {
	coeffExp := n - preferredQExp
	if coeffExp <= 0 || coeffExp > 33 {
		// coefficient 1, qExp = n
		return finalize_128_ctx(sign, n, u128{1, 0}, RESIDUE_EXACT, rounding, ctxnull)
	}
	coeff := fromLoHi_128_64x64(pow10_128_dw0(coeffExp), pow10_128_dw1(coeffExp))
	return finalize_128_ctx(sign, preferredQExp, coeff, RESIDUE_EXACT, rounding, ctxnull)
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
