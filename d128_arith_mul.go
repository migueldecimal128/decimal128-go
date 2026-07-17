// Multiply and square. The compact 64×64 hot path lives in the fast entry
// d128_multiply_rnd_ctxnull; every other case tail-calls the (cold)
// d128_multiply_slow_rnd_ctxnull. The bare _tte / _rnd / _ctx forms are thin
// wrappers that all share the compact fast path. A square's fast entry carries an
// x²-specialized guard and tail-calls the same cold body.

package decimal128

// Fast path: both coefficients fit in 64 bits (compact product), exponent sum in
// range, and the 64×64 product fits in 112 bits (canonical significand) -> return
// directly, no rounding, no flags. Everything else tail-calls the cold slow entry.
// Go has no always_inline, and the fast entry's guard + inline arithmetic exceeds
// the ~80 inline budget, so (as with add) it does not collapse into Multiply; the
// split's Go payoff is hot/cold code separation.
func d128_multiply_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
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
	return d128_multiply_slow_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Cold path, kept out of line via //go:noinline so the fast entry stays a small,
// i-cache-friendly hot function. Wide 128×128→256 product with a fits-in-112
// early-out before finalize, plus the FNZ and non-finite cases. Also reached from
// square's fast entry.
//
//go:noinline
func d128_multiply_slow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	xHi49 := d128_hi49(x)
	yHi49 := d128_hi49(y)
	prod := mul_256_128x128(x.ubdLo64, xHi49, y.ubdLo64, yHi49)
	hi49Sum := xHi49 + yHi49
	prodQExp := d128_qExp(x) + d128_qExp(y)
	prodSignFlag := d128_signFlag(x) != d128_signFlag(y)
	if ushr_64_64(hi49Sum, 48) == 0 && prodQExp >= -6176 && prodQExp <= 6111 &&
		(prod.dw3|prod.dw2) == 0 && ushr_64_64(prod.dw1, 48) == 0 {
		// Secondary fast return: coefficients too wide for the 64×64 guard, but
		// the full product still fits in 112 bits canonical -> skip finalize.
		lo := prod.lo()
		return d128Pack(prodSignFlag, prodQExp, lo.dw0, lo.dw1)
	}
	if bothFinite(x, y) {
		// Reuse prod computed above: the finite×finite slow path is just
		// "finalize the 256-bit product". Recomputing it in a mulFnzFnz helper
		// would be a redundant 128×128→256 multiply on every full-width multiply.
		return finalize_256_ctxnull(prodSignFlag, prodQExp, prod, RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if (d128_isINF(x) && d128_isZER(y)) || (d128_isINF(y) && d128_isZER(x)) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity)
	}
	return d128_infinity(prodSignFlag)
}

func d128_multiply_tte(x, y d128) d128 {
	return d128_multiply_rnd_ctxnull(x, y, ROUND_TIES_TO_EVEN, nil)
}
func d128_multiply_rnd(x, y d128, rnd round) d128 { return d128_multiply_rnd_ctxnull(x, y, rnd, nil) }
func d128_multiply_ctx(x, y d128, ctx *context) d128 {
	return d128_multiply_rnd_ctxnull(x, y, ctx.round, ctx)
}

// Square: x*x. The fast entry carries an x²-specialized compact guard (single
// operand check, sign always non-negative) and tail-calls the shared multiply
// cold body for everything else.
func d128_square_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	x128 := d128_u128(x)
	xQ := d128_qExp(x)
	prodQExp := xQ + xQ
	if x128.dw1 == 0 && prodQExp >= -6176 && prodQExp <= 6111 {
		prod := mul_128_64x64(x128.dw0, x128.dw0)
		if ushr_64_64(prod.dw1, 48) == 0 {
			return d128Pack(false, prodQExp, prod.dw0, prod.dw1) // a square is non-negative
		}
	}
	return d128_multiply_slow_rnd_ctxnull(x, x, rounding, ctxnull)
}

func d128_square_tte(x d128) d128 {
	return d128_square_rnd_ctxnull(x, ROUND_TIES_TO_EVEN, nil)
}
func d128_square_rnd(x d128, rnd round) d128 { return d128_square_rnd_ctxnull(x, rnd, nil) }
func d128_square_ctx(x d128, ctx *context) d128 {
	return d128_square_rnd_ctxnull(x, ctx.round, ctx)
}
