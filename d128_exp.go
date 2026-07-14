// exp(x) = e^x and exp10(x) = 10^x. The finite non-zero magnitude is computed
// in the 38-digit D38 domain by a [9,9] Padé approximation after range
// reduction, then lowered to D128 with the caller's rounding. A non-special
// exp/exp10 result is irrational, hence always INEXACT.

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
			return exp10ImplFNZ(x, rounding, ctxnull)
		}
		return expImplFNZ(x, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		return d128_zero(false, 0) // exp(−∞) = +0
	}
	return D128_POS_INFINITY // exp(+∞) = +∞
}

// --- exp magnitude ---

func expImplFNZ(x d128, rounding round, ctxnull *context) d128 {
	// |x| ≥ 10^6: exp definitely overflows (x > 0) or underflows to 0 (x < 0).
	// Catching it here also keeps n = round(x/ln10) inside int range below.
	if d128_sciExp(x) >= 6 {
		if d128_signFlag(x) {
			return transcendentalUnderflow(rounding, ctxnull)
		}
		return transcendentalOverflow(rounding, ctxnull)
	}
	xd := d38Pack(d128_signFlag(x), d128_qExp(x), x.ubdLo64, d128_hi49(x))

	// Range reduction: n = round(x / ln10), r = x − n·ln10 ∈ [−ln10/2, ln10/2].
	nDec := d38_roundToIntegralTiesToAway(d38_divide(xd, LN10))
	n := int(d38_toLongOrMin(nDec))
	r := xd
	if n != 0 {
		// n·ln10 with the 50-digit ln10; r = x − n·ln10 via the wide signed add.
		nMag := int64(absInt(n))
		nLn10 := mul_256_256x256(
			LN10_50_COEFF.dw0, LN10_50_COEFF.dw1, LN10_50_COEFF.dw2, LN10_50_COEFF.dw3,
			nMag, 0, 0, 0)
		r = d38_addWide(
			d38_signFlag(xd), d38_qExp(xd), u256FromU128(d38_u128(xd)),
			n > 0, LN10_50_QEXP, nLn10) // op2 = −(n·ln10)
	}

	// exp(r) = exp(r/8)^8 — three squarings.
	rPrime := d38_divide(r, D38_EIGHT)
	e := d38_padeEval(rPrime, padeExpPWeights, padeExpQWeights)
	e = d38_square(e)
	e = d38_square(e)
	e = d38_square(e)

	// exp(x) = exp(r) · 10^n — shift the exponent by n; finalize handles range.
	result := finalize_128_ctxnull(false, d38_qExp(e)+n, d38_u128(e), RESIDUE_EXACT, rounding, ctxnull)
	return forceInexact(result, ctxnull)
}

// --- exp10 magnitude ---

func exp10ImplFNZ(x d128, rounding round, ctxnull *context) d128 {
	xd := d38Pack(d128_signFlag(x), d128_qExp(x), x.ubdLo64, d128_hi49(x))
	return d128_exp10_from_d38(xd, rounding, ctxnull)
}

// d128_exp10_from_d38 computes 10^xd for a 38-digit argument, lowered to D128 with the
// caller's rounding. exp10ImplFNZ lifts a D128 into this; pow feeds the
// full-precision y·log10(x) product directly so the result is rounded once.
func d128_exp10_from_d38(xd d38, rounding round, ctxnull *context) d128 {
	// Exact fast path: 10^(integer) is exact (avoids transcendental error).
	l := d38_toLongOrMin(xd)
	if l != minInt64 {
		if l > 6144 {
			return transcendentalOverflow(rounding, ctxnull)
		}
		if l < -6176 {
			return transcendentalUnderflow(rounding, ctxnull)
		}
		return finalize_128_ctxnull(false, int(l), u128{1, 0}, RESIDUE_EXACT, rounding, ctxnull)
	}

	// Range reduction: n = round(x), r = x − n ∈ [−0.5, 0.5] (exact, no ln10).
	nDec := d38_roundToIntegralTiesToAway(xd)
	nLong := d38_toLongOrMin(nDec)
	if nLong == minInt64 {
		if d38_signFlag(xd) {
			return transcendentalUnderflow(rounding, ctxnull)
		}
		return transcendentalOverflow(rounding, ctxnull)
	}
	if nLong > 6200 {
		return transcendentalOverflow(rounding, ctxnull)
	}
	if nLong < -6200 {
		return transcendentalUnderflow(rounding, ctxnull)
	}
	n := int(nLong)
	r := d38_subtract(xd, nDec)

	// 10^r = 10^(r/8)^8 — three squarings.
	rPrime := d38_divide(r, D38_EIGHT)
	z := d38_padeEval(rPrime, padeExp10PWeights, padeExp10QWeights)
	z = d38_square(z)
	z = d38_square(z)
	z = d38_square(z)

	result := finalize_128_ctxnull(false, d38_qExp(z)+n, d38_u128(z), RESIDUE_EXACT, rounding, ctxnull)
	return forceInexact(result, ctxnull)
}

// --- shared helpers ---

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// forceInexact: a non-special exp/exp10 result is irrational, hence inexact.
// The final round usually sets the flag, but force it for the rare exact-looking
// 38-digit landing (same rationale as ln/log10).
func forceInexact(d d128, ctxnull *context) d128 {
	if ctxnull != nil {
		return ctxSignalInexact_ctxnull(ctxnull, d)
	}
	return d
}

// transcendentalOverflow: positive overflow → +∞ under round-to-nearest /
// toward-positive, else the largest finite value; OVERFLOW + INEXACT.
func transcendentalOverflow(rounding round, ctxnull *context) d128 {
	return finalizeOverflow_ctxnull(false, rounding, ctxnull)
}

// transcendentalUnderflow: positive underflow swamped below subnormal → +0
// under round-to-nearest / toward-zero / toward-negative, else the smallest
// finite value; UNDERFLOW + INEXACT.
func transcendentalUnderflow(rounding round, ctxnull *context) d128 {
	return finalizeUnderflowFullySwamped(false, rounding, ctxnull)
}
