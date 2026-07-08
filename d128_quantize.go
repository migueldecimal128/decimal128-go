// IEEE 754-2019 §5.3.2 quantize: rescale x to the quantum (encoded exponent) of
// y, keeping the sign of x. A finer target appends zeros (exact, but invalid if
// it would exceed 34 digits); a coarser target rounds and may signal inexact.
// quantize never signals overflow/underflow — the target exponent comes from a
// valid operand.

package decimal128

// _rnd_ctxnull is the canonical body; _tte/_rnd/_ctx forward into it — same
// factoring as d128_divide_* (d128_arith_div). quantize is the sole rounding op
// with a _rnd form; non-rounding neighbours (scaleB/logB/…) keep bare _ctxnull.
func d128_quantize_tte(x, y d128) d128            { return d128_quantize_rnd_ctxnull(x, y, TIES_TO_EVEN, nil) }
func d128_quantize_rnd(x, y d128, rnd round) d128 { return d128_quantize_rnd_ctxnull(x, y, rnd, nil) }
func d128_quantize_ctx(x, y d128, ctx *context) d128 {
	return d128_quantize_rnd_ctxnull(x, y, ctx.round, ctx)
}

func d128_quantize_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	xInf := d128_isINF(x)
	yInf := d128_isINF(y)
	if xInf || yInf {
		if xInf && yInf {
			return d128_infinity(d128_signFlag(x))
		}
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeQuantizeExactlyOneOperandIsInfinite)
	}

	// Both operands finite. The result takes its sign from x and its exponent
	// from y; only the coefficient is recomputed.
	sign := d128_signFlag(x)
	qTarget := d128_qExp(y)
	coeff := d128_u128(x)

	// Zero has no significant digits: representable at any exponent, exactly.
	if isZero_bool_128(coeff.dw0, coeff.dw1) {
		return d128_zero(sign, qTarget)
	}

	delta := d128_qExp(x) - qTarget
	if delta == 0 {
		return x
	}
	if delta > 0 {
		// Finer quantum: append delta zeros. Exact, unless the coefficient
		// would exceed 34 digits — which GDAS / IEEE 754 make invalid.
		if digitLen_int_128(coeff.dw0, coeff.dw1)+delta > 34 {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeQuantizeResultWouldExceedPrecision)
		}
		scaled := mulPow10_128_128(coeff.dw0, coeff.dw1, delta)
		return finalize_128_ctxnull(sign, qTarget, scaled, RESIDUE_EXACT, rounding, ctxnull)
	}
	// Coarser quantum: drop -delta low digits, rounding via finalize (which
	// raises inexact when the discarded digits are non-zero).
	qr := div_128_pow10_checked_q128res(coeff, -delta)
	reduced := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	return finalize_128_ctxnull(sign, qTarget, reduced, qr.residue, rounding, ctxnull)
}
