// IEEE 754-2019 §5.3.3 logB and scaleB.
//
// logB(x) returns the adjusted (scientific) exponent of x as a D128. scaleB(x,n)
// returns x × 10^n, rounded within ctx. Both treat infinities and NaNs per the
// standard; logB(±0) signals divisionByZero.

package decimal128

import "math"

// d128_logB_ctxnull returns `logB(x)` -- the adjusted (scientific) exponent as a `D128`,
// following IEEE 754-2019 Section 5.3.3.
//
// | Input           | Result                                    |
// |-----------------|-------------------------------------------|
// | finite non-zero | `D128.from(sciExp)`                 |
// | +/-0              | -inf, signals `divisionByZero`              |
// | +/-inf              | +inf                                        |
// | qNaN            | NaN (propagates, no flag)                 |
// | sNaN            | NaN, signals `invalidOperation`           |
func d128_logB_ctxnull(x d128, ctxnull *context) d128 {
	if d128_isZER(x) {
		return ctxSignalDivByZero_ctxnull(ctxnull, true)
	}
	if d128_isFNZ(x) {
		return d128_fromInt64(d128_sciExp(x))
	}
	if d128_isINF(x) {
		return const_D128_POS_INFINITY
	}
	// NaN: qNaN propagates, sNaN signals invalid_operation.
	return ctxNanOperand1_ctxnull(ctxnull, x)
}

// d128_scaleB_ctxnull returns `x * 10^pow10Delta`, rounded as needed within `ctx`.
//
// Equivalent to IEEE 754-2019 Section 5.3.3 `scaleB`.
//
// Infinities pass through unchanged; sNaNs signal `invalid_operation`.
func d128_scaleB_ctxnull(x d128, pow10Delta int, ctxnull *context) d128 {
	if d128_isFIN(x) {
		pow10DeltaCapped := max(min(pow10Delta, 100_000), -100_000)
		qNew := d128_qExp(x) + pow10DeltaCapped
		rounding := const_ROUND_TIES_TO_EVEN
		if ctxnull != nil {
			rounding = ctxnull.round
		}
		return finalize_128_ctxnull(d128_signFlag(x), qNew, d128_u128(x), const_RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_isINF(x) {
		return x
	}
	return ctxNanOperand1_ctxnull(ctxnull, x)
}

// d128_isSameQuantum returns `true` if `x` and `other` share the same *quantum* (same
// encoded exponent `qExp`), per IEEE 754-2019 Section 5.7.3.
//
// Two values have the same quantum when their exponents are identical
// after encoding, regardless of whether their numerical values are
// equal. For non-finite values: two infinities share a quantum; a NaN
// shares a quantum only with another NaN.
func d128_isSameQuantum(x, other d128) bool {
	if d128_isFIN(x) {
		return d128_isFIN(other) && d128_qExp(x) == d128_qExp(other)
	}
	if d128_isINF(x) {
		return d128_isINF(other)
	}
	return d128_isNAN(other)
}

// The quantum (encoded exponent) of a finite x. A non-finite x signals
// invalidOperation and returns the minimum integer as a sentinel.
func d128_quantumInt_ctxnull(x d128, ctxnull *context) int {
	if d128_isFIN(x) {
		return d128_qExp(x)
	}
	_ = ctxSignalInvalidOperation_ctxnull(ctxnull, causeQuantumOfNonFinite, x)
	return math.MinInt
}

// d128_quantize_tte returns `x` rescaled to the quantum (encoded exponent) of `y`, per
// IEEE 754-2019 Section 5.3.2 `quantize`.
//
// The numerical value of `x` is rounded -- under `ctx`'s rounding -- to the
// exponent of `y`, keeping the sign of `x`. Moving to a *finer* quantum
// (`qExp(y) < qExp(x)`) appends zeros and is always exact; moving to
// a *coarser* quantum rounds and may signal `inexact`.
//
// | Inputs                          | Result                              |
// |---------------------------------|-------------------------------------|
// | both finite, fits in 34 digits  | value of `x` at `qExp(y)`           |
// | both finite, needs > 34 digits  | NaN, signals `invalidOperation`     |
// | both +/-inf                         | +/-inf with the sign of `x`             |
// | exactly one +/-inf                  | NaN, signals `invalidOperation`     |
// | any NaN                         | propagated NaN (sNaN => invalid)     |
//
// `quantize` never signals `overflow` or `underflow`: the target exponent
// comes from a valid operand, and the coefficient either shrinks (coarser)
// or is checked against precision up front (finer).
//
// `_rnd_ctxnull` is the canonical body (rounding as a value, sink optional);
// the `_tte` / `_rnd` / `_ctx` rungs below forward into it -- the same factoring
// as `d128_divide_*` (D128ArithDiv). `quantize` is the sole rounding op with a
// `_rnd` form; the non-rounding neighbours (scaleB/logB/...) keep bare `_ctxnull`.
func d128_quantize_tte(x, y d128) d128 {
	return d128_quantize_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
}

func d128_quantize_rnd(x, y d128, rnd round) d128 { return d128_quantize_rnd_ctxnull(x, y, rnd, nil) }

func d128_quantize_ctx(x, y d128, ctx *context) d128 {
	return d128_quantize_rnd_ctxnull(x, y, ctx.round, ctx)
}

// quantize(x, y): x rescaled to the quantum (encoded exponent) of y, keeping
// x's sign (IEEE 754-2019 Section 5.3.2). The general body.
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
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeQuantizeExactlyOneOperandIsInfinite, const_D128_QNAN0)
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
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeQuantizeResultWouldExceedPrecision, const_D128_QNAN0)
		}
		scaled := mulPow10_128_128(coeff.dw0, coeff.dw1, delta)
		return finalize_128_ctxnull(sign, qTarget, scaled, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	// Coarser quantum: drop -delta low digits, rounding via finalize (which
	// raises inexact when the discarded digits are non-zero).
	qr := div_128_pow10_checked_q128res(coeff, -delta)
	reduced := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	return finalize_128_ctxnull(sign, qTarget, reduced, qr.residue, rounding, ctxnull)
}
