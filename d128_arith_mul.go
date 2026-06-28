// Multiply and square. d128_multiply_rnd_ctxnull is the general body; the bare
// _tte forms keep a cheaper 64×64 fast-path prologue. A square forwards to
// multiply (x·x), inheriting the same fast path and finalization.

package decimal128

func d128_multiply_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	prod := mul_256_128x128(x.ubdLo64, d128_hi49(x), y.ubdLo64, d128_hi49(y))
	hi49Sum := d128_hi49(x) + d128_hi49(y)
	prodQExp := d128_qExp(x) + d128_qExp(y)
	prodSignFlag := d128_signFlag(x) != d128_signFlag(y)
	if ushr_64_64(hi49Sum, 48) == 0 && prodQExp >= -6176 && prodQExp <= 6111 &&
		(prod.dw3|prod.dw2) == 0 && ushr_64_64(prod.dw1, 48) == 0 {
		lo := prod.lo()
		return d128Pack(prodSignFlag, prodQExp, lo.dw0, lo.dw1)
	}
	if bothFinite(x, y) {
		return mulFnzFnz_rnd_ctxnull(x, y, rounding, ctxnull)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", d128_isINF(x) || d128_isINF(y))
	if (d128_isINF(x) && d128_isZER(y)) || (d128_isINF(y) && d128_isZER(x)) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity)
	}
	return d128_infinity(prodSignFlag)
}

func d128_multiply_tte(x, y d128) d128 {
	x128 := d128_u128(x)
	y128 := d128_u128(y)
	allHiBits := x128.dw1 | y128.dw1
	prodQExp := d128_qExp(x) + d128_qExp(y)
	if allHiBits == 0 && prodQExp >= -6176 && prodQExp <= 6111 {
		prodSignFlag := d128_signFlag(x) != d128_signFlag(y)
		prod := mul_128_64x64(x128.dw0, y128.dw0)
		if ushr_64_64(prod.dw1, 48) == 0 {
			return d128Pack(prodSignFlag, prodQExp, prod.dw0, prod.dw1)
		}
	}
	return d128_multiply_rnd_ctxnull(x, y, TIES_TO_EVEN, nil)
}

func d128_multiply_rnd(x, y d128, rnd round) d128 { return d128_multiply_rnd_ctxnull(x, y, rnd, nil) }
func d128_multiply_ctx(x, y d128, ctx *context) d128 {
	return d128_multiply_rnd_ctxnull(x, y, ctx.round, ctx)
}

func d128_square_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return d128_multiply_rnd_ctxnull(x, x, rounding, ctxnull)
}

func d128_square_tte(x d128) d128 {
	x128 := d128_u128(x)
	prodQExp := d128_qExp(x) + d128_qExp(x)
	if x128.dw1 == 0 && prodQExp >= -6176 && prodQExp <= 6111 {
		prod := mul_128_64x64(x128.dw0, x128.dw0)
		if ushr_64_64(prod.dw1, 48) == 0 {
			return d128Pack(false, prodQExp, prod.dw0, prod.dw1) // a square is non-negative
		}
	}
	return d128_multiply_rnd_ctxnull(x, x, TIES_TO_EVEN, nil)
}

func d128_square_rnd(x d128, rnd round) d128 { return d128_multiply_rnd_ctxnull(x, x, rnd, nil) }
func d128_square_ctx(x d128, ctx *context) d128 {
	return d128_multiply_rnd_ctxnull(x, x, ctx.round, ctx)
}

func mulFnzFnz_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	prod := mul_256_128x128(x.ubdLo64, d128_hi49(x), y.ubdLo64, d128_hi49(y))
	prodSign := d128_signFlag(x) != d128_signFlag(y)
	prodQExp := d128_qExp(x) + d128_qExp(y)
	return finalize_256_ctxnull(prodSign, prodQExp, prod, RESIDUE_EXACT, rounding, ctxnull)
}
