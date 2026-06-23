// IEEE 754-2019 §5.3.3 logB and scaleB.
//
// logB(x) returns the adjusted (scientific) exponent of x as a D128. scaleB(x,n)
// returns x × 10^n, rounded within ctx. Both treat infinities and NaNs per the
// standard; logB(±0) signals divisionByZero.

package decimal128

// d128_logB returns logB(x): finite non-zero → D128 of its sciExp; ±0 → −∞ and
// signals divisionByZero; ±∞ → +∞; qNaN propagates; sNaN signals invalid.
func d128_logB(x d128, ctxnull *context) d128 {
	if d128_isZER(x) {
		return ctxSignalDivByZero(ctxnull, true)
	}
	if d128_isFNZ(x) {
		return d128_fromInt(d128_sciExp(x))
	}
	if d128_isINF(x) {
		return D128_POS_INFINITY
	}
	// NaN: qNaN propagates, sNaN signals invalid_operation.
	return ctxNanOperand1(ctxnull, x)
}

// d128_scaleB returns x × 10^pow10Delta, rounded as needed within ctx.
// Infinities pass through unchanged; sNaNs signal invalid_operation.
func d128_scaleB(x d128, pow10Delta int, ctxnull *context) d128 {
	if d128_isFIN(x) {
		pow10DeltaCapped := max(min(pow10Delta, 100_000), -100_000)
		qNew := d128_qExp(x) + pow10DeltaCapped
		rounding := TIES_TO_EVEN
		if ctxnull != nil {
			rounding = ctxnull.round
		}
		return finalize_128_ctx(d128_signFlag(x), qNew, d128_u128(x), RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_isINF(x) {
		return x
	}
	return ctxNanOperand1(ctxnull, x)
}
