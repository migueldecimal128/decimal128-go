// Natural log and base-10 log. The finite non-zero magnitude is computed in the
// 48-digit D48 domain (d128_log_d48.go: argument reduction, square-root
// reduction, an [11/12] Padé approximation), then rounded once into D128 with
// the caller's rounding. Correctly rounded except in the documented 1-ulp gap
// for arguments very close to 1 in directed modes (CrossPlatformArchitecture.md
// §12, "Transcendental accuracy").

package decimal128

// --- ln ladder ---

func d128_ln_tte(x d128) d128            { return d128_ln_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_ln_rnd(x d128, rnd round) d128 { return d128_ln_rnd_ctxnull(x, rnd, nil) }
func d128_ln_ctx(x d128, ctx *context) d128 {
	return d128_ln_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_ln_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logDispatch(x, false, rounding, ctxnull)
}

// --- log10 ladder ---

func d128_log10_tte(x d128) d128            { return d128_log10_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_log10_rnd(x d128, rnd round) d128 { return d128_log10_rnd_ctxnull(x, rnd, nil) }
func d128_log10_ctx(x d128, ctx *context) d128 {
	return d128_log10_rnd_ctxnull(x, ctx.round, ctx)
}

func d128_log10_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logDispatch(x, true, rounding, ctxnull)
}

// --- dispatch (IEEE 754-2019 §9.2 special cases) ---

func logDispatch(x d128, isLog10 bool, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			// log(±0) = −∞, signalling division-by-zero.
			return ctxSignalDivByZero_ctxnull(ctxnull, true)
		}
		if d128_signFlag(x) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeLogOfNegNumber)
		}
		if isLog10 && isExactPowerOf10(x) {
			// log10(10^n) = n exactly — no rounding, no inexact flag.
			return d128_fromInt64(d128_sciExp(x))
		}
		return logImplFNZ(x, isLog10, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeLogOfNegNumber)
	}
	return const_D128_POS_INFINITY // log(+∞) = +∞
}

// --- magnitude ---

func logImplFNZ(x d128, isLog10 bool, rounding round, ctxnull *context) d128 {
	r := d48_logMagnitude(d48_fromD128(x, false), isLog10)
	result := finalizeD48_ctxnull(r, rounding, ctxnull)

	// A non-zero ln/log10 of a finite value is irrational -- always INEXACT. The
	// final round usually sets it, but when the 48-digit value lands on an
	// exact-looking value the round sees no residue, so force the flag. (r == 0
	// is the lone exact case, ln(1) = 0; log10 exacts are power-of-10
	// fast-pathed in logDispatch.) ln / log10 have no exact rule beyond those,
	// so they don't run the boundary test: a result near a rounding boundary is
	// rounded as is (the documented 1-ulp gap near 1, CrossPlatformArchitecture.md).
	if d48_isZero(r) {
		return result
	}
	return d48_forceInexact(result, ctxnull)
}

// isExactPowerOf10 reports whether x is exactly an integer power of ten (1·10^n)
// — the log10 fast path.
func isExactPowerOf10(x d128) bool {
	digitLen := d128_digitLen(x)
	c := d128_u128(x)
	return cmp_int_128x128_pow10(c.dw0, c.dw1, digitLen-1) == 0
}
