// exp(x) = e^x and exp10(x) = 10^x. The finite non-zero magnitude is computed in the
// 48-digit D48 domain (D128ExpD48: range reduction, a [12/12] Pade approximation,
// three squarings), then rounded once into D128 with the caller's rounding. A
// non-special exp/exp10 result is irrational, hence always INEXACT. Correctly rounded
// except in the documented 1-ulp gap for arguments very close to 0 in directed modes
// (CrossPlatformArchitecture.md Section 12, "Transcendental Accuracy").

package decimal128

// --- exp ladder ---

func d128_exp_tte(x d128) d128            { return d128_exp_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_exp_rnd(x d128, rnd round) d128 { return d128_exp_rnd_ctxnull(x, rnd, nil) }
func d128_exp_ctx(x d128, ctx *context) d128 {
	return d128_exp_rnd_ctxnull(x, ctx.round, ctx)
}

// IEEE 754-2019 Section 9.2 exp(x) = e^x. The finite nonzero magnitude is
// computed in the 48-digit D48 domain (range reduction, a [12/12] Pade
// approximant, three squarings), then rounded once with the caller's rounding.
// A non-special result is irrational, hence always inexact. Correctly rounded,
// except within the documented 1-ulp gap (CrossPlatformArchitecture.md,
// "Transcendental accuracy").
func d128_exp_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	if rounding == const_ROUND_05_UP {
		// Cold route: toward zero, then fix the last digit (round05_up.go).
		tz := newContextRound(const_ROUND_TOWARD_ZERO)
		t := d128_exp_rnd_ctxnull(x, const_ROUND_TOWARD_ZERO, tz)
		return round05Up_finish_ctxnull(t, tz.flags, ctxnull)
	}
	return expDispatch(x, false, rounding, ctxnull)
}

// --- exp10 ladder ---

func d128_exp10_tte(x d128) d128            { return d128_exp10_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_exp10_rnd(x d128, rnd round) d128 { return d128_exp10_rnd_ctxnull(x, rnd, nil) }
func d128_exp10_ctx(x d128, ctx *context) d128 {
	return d128_exp10_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_exp10_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	if rounding == const_ROUND_05_UP {
		// Cold route: toward zero, then fix the last digit (round05_up.go).
		tz := newContextRound(const_ROUND_TOWARD_ZERO)
		t := d128_exp10_rnd_ctxnull(x, const_ROUND_TOWARD_ZERO, tz)
		return round05Up_finish_ctxnull(t, tz.flags, ctxnull)
	}
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
	return const_D128_POS_INFINITY // exp(+∞) = +∞
}
