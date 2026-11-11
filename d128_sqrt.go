// IEEE 754-2019 §5.4.1 squareRoot. Special classes per §6.3; the finite nonzero
// magnitude is an exact integer square root in the unsigned-integer domain
// (sqrtFloorScaled), correctly rounded for every input.

package decimal128

func d128_sqrt_tte(x d128) d128            { return d128_sqrt_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_sqrt_rnd(x d128, rnd round) d128 { return d128_sqrt_rnd_ctxnull(x, rnd, nil) }
func d128_sqrt_ctx(x d128, ctx *context) d128 {
	return d128_sqrt_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_sqrt_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			// §6.3: squareRoot(±0) = ±0, sign preserved; qExp = floor(Q/2).
			return d128_zero(d128_signFlag(x), d128_qExp(x)>>1)
		}
		if d128_signFlag(x) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeSquareRootOfNegFiniteNonZero)
		}
		return sqrtMagnitudeFnz_ctxnull(x, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeSquareRootOfNegInfinity)
	}
	return const_D128_POS_INFINITY
}

func sqrtMagnitudeFnz_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	f := sqrtFloorScaled(d128_u128(x), d128_qExp(x), d128_digitLen(x))
	result := finalize_256_ctxnull(false, f.resultQExp, f.s, f.residue, rounding, ctxnull)

	// GDAS ideal exponent floor(Q/2): reduce toward it ONLY for exact results
	// (a perfect square whose root fits in <=34 digits). Inexact results keep
	// full precision — their trailing zeros are significant.
	isExact := f.residue == const_RESIDUE_EXACT && isMultipleOfPow10_4(f.s)
	if isExact {
		maxToStrip := (d128_qExp(x) >> 1) - d128_qExp(result)
		if maxToStrip > 0 {
			return d128_stripTrailingZeros(result, maxToStrip)
		}
	}
	return result
}

// isMultipleOfPow10_4 is true iff s (always 38 digits) is divisible by 10^4 —
// i.e. its root reduces to <=34 significant digits.
func isMultipleOfPow10_4(s u256) bool {
	qr := u256DivRem128(s, u128{10000, 0})
	return isZero_bool_128(qr.remDw0, qr.remDw1)
}
