// exp(x) = e^x and exp10(x) = 10^x. The finite non-zero magnitude is computed
// in the 48-digit D48 domain (d128_exp_d48.go: range reduction, a [12/12] Padé
// approximation, three squarings), then rounded once into D128 with the
// caller's rounding. A non-special exp/exp10 result is irrational, hence always
// INEXACT. Correctly rounded except in the documented 1-ulp gap for arguments
// very close to 0 in directed modes (CrossPlatformArchitecture.md §12,
// "Transcendental accuracy").

package decimal128

// --- exp ladder ---

func d128_exp_tte(x d128) d128            { return d128_exp_rnd_ctxnull(x, ROUND_TIES_TO_EVEN, nil) }
func d128_exp_rnd(x d128, rnd round) d128 { return d128_exp_rnd_ctxnull(x, rnd, nil) }
func d128_exp_ctx(x d128, ctx *context) d128 {
	return d128_exp_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_exp_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return expDispatch(x, false, rounding, ctxnull)
}

// --- exp10 ladder ---

func d128_exp10_tte(x d128) d128            { return d128_exp10_rnd_ctxnull(x, ROUND_TIES_TO_EVEN, nil) }
func d128_exp10_rnd(x d128, rnd round) d128 { return d128_exp10_rnd_ctxnull(x, rnd, nil) }
func d128_exp10_ctx(x d128, ctx *context) d128 {
	return d128_exp10_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_exp10_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return expDispatch(x, true, rounding, ctxnull)
}

// --- dispatch ---

func expDispatch(x d128, isExp10 bool, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return d128_fromInt64(1) // exp(±0) = 1 exactly
		}
		if isExp10 {
			return exp10ImplFNZ_d48(x, rounding, ctxnull)
		}
		return expImplFNZ_d48(x, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		return d128_zero(false, 0) // exp(−∞) = +0
	}
	return d128_POS_INFINITY() // exp(+∞) = +∞
}
