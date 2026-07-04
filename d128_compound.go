// IEEE 754-2019 §9.2 compound(x, n) = (1+x)^n for x ≥ -1 and integer n. Form
// t = 1+x, reject x < -1, then delegate to pown(t, n). x = -1 (t = 0) and the
// special classes are handled directly; a zero result carries the preferred
// exponent ⌊n·min(0, Q(x))⌋.

package decimal128

var d128CompoundOne = d128_fromInt64(1)

func d128_compound_tte(x d128, n int) d128 { return d128_compound_rnd_ctxnull(x, n, TIES_TO_EVEN, nil) }
func d128_compound_rnd(x d128, n int, rnd round) d128 {
	return d128_compound_rnd_ctxnull(x, n, rnd, nil)
}
func d128_compound_ctx(x d128, n int, ctx *context) d128 {
	return d128_compound_rnd_ctxnull(x, n, ctx.round, ctx)
}

func d128_compound_rnd_ctxnull(x d128, n int, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return d128_fromInt64(1) // (1 ± 0)^n = 1
		}
		return compoundFNZ(x, n, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		// compound(x, 0) is 1 for a quiet NaN (§9.2).
		if n == 0 && d128_isQNAN(x) {
			return d128_fromInt64(1)
		}
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		// -∞ is below -1 → invalid (even for n == 0).
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeCompoundXLtNegOne)
	}
	if n > 0 {
		return D128_POS_INFINITY // (1 + ∞)^(n>0) = +∞
	}
	if n == 0 {
		return d128_fromInt64(1) // compound(+∞, 0) = 1
	}
	return d128_zero(false, 0) // n < 0 → +0
}

func compoundFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	// t = 1 + x.
	t := d128_add_rnd_ctxnull(d128CompoundOne, x, rounding, ctxnull)
	if d128_signFlag(t) && !d128_isZER(t) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeCompoundXLtNegOne) // x < -1
	}
	if n == 0 {
		return d128_fromInt64(1)
	}
	if d128_isZER(t) {
		// x = -1 exactly.
		if n < 0 {
			return ctxSignalDivByZero_ctxnull(ctxnull, false) // +∞
		}
		// n > 0 → +0 with the preferred exponent ⌊n·min(0, Q(x))⌋.
		nCapped := min(n, 10_000) // keep the product inside int
		return d128_zero(false, nCapped*min(0, d128_qExp(x)))
	}
	result := d128_pown_rnd_ctxnull(t, n, rounding, ctxnull)
	if d128_isZER(result) {
		absN := int64(n)
		if absN < 0 {
			absN = -absN
		}
		qPreferred := int(maxI64(minI64(absN*int64(min(0, d128_qExp(x))), 6111), -6176))
		return d128_zero(false, qPreferred)
	}
	return result
}
