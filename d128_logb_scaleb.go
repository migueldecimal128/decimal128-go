// IEEE 754-2019 §5.3.3 logB and scaleB.
//
// logB(x) returns the adjusted (scientific) exponent of x as a D128. scaleB(x,n)
// returns x × 10^n, rounded within ctx. Both treat infinities and NaNs per the
// standard; logB(±0) signals divisionByZero.

package decimal128

import "math"

// d128_logB_ctxnull returns logB(x): finite non-zero → D128 of its sciExp; ±0 → −∞ and
// signals divisionByZero; ±∞ → +∞; qNaN propagates; sNaN signals invalid.
func d128_logB_ctxnull(x d128, ctxnull *context) d128 {
	if d128_isZER(x) {
		return ctxSignalDivByZero_ctxnull(ctxnull, true)
	}
	if d128_isFNZ(x) {
		return d128_fromInt64(d128_sciExp(x))
	}
	if d128_isINF(x) {
		return D128_POS_INFINITY
	}
	// NaN: qNaN propagates, sNaN signals invalid_operation.
	return ctxNanOperand1_ctxnull(ctxnull, x)
}

// d128_isSameQuantum reports whether x and other share the same quantum (the
// same encoded exponent qExp), per IEEE 754-2019 §5.7.3 — regardless of value.
// It is quiet: two infinities share a quantum; a NaN shares one only with
// another NaN; no flags are ever raised (sNaN does not signal here).
func d128_isSameQuantum(x, other d128) bool {
	if d128_isFIN(x) {
		return d128_isFIN(other) && d128_qExp(x) == d128_qExp(other)
	}
	if d128_isINF(x) {
		return d128_isINF(other)
	}
	return d128_isNAN(other)
}

// d128_quantumInt_ctxnull returns the encoded exponent qExp of a finite x; for a
// non-finite x it signals invalidOperation and returns the int sentinel
// math.MinInt.
func d128_quantumInt_ctxnull(x d128, ctxnull *context) int {
	if d128_isFIN(x) {
		return d128_qExp(x)
	}
	_ = ctxSignalInvalidOperation_ctxnull(ctxnull, causeQuantumOfNonFinite)
	return math.MinInt
}

// d128_scaleB_ctxnull returns x × 10^pow10Delta, rounded as needed within ctx.
// Infinities pass through unchanged; sNaNs signal invalid_operation.
func d128_scaleB_ctxnull(x d128, pow10Delta int, ctxnull *context) d128 {
	if d128_isFIN(x) {
		pow10DeltaCapped := max(min(pow10Delta, 100_000), -100_000)
		qNew := d128_qExp(x) + pow10DeltaCapped
		rounding := TIES_TO_EVEN
		if ctxnull != nil {
			rounding = ctxnull.round
		}
		return finalize_128_ctxnull(d128_signFlag(x), qNew, d128_u128(x), RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_isINF(x) {
		return x
	}
	return ctxNanOperand1_ctxnull(ctxnull, x)
}
