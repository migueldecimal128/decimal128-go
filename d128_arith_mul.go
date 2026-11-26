// IEEE 754 multiply and square. `d128_multiply_rnd_ctxnull` is the general body; the
// `_tte` (bare), `_rnd`, and `_ctx` forms are thin specializations of it (rounding/ctx
// folded in). Fast/slow split (as in add): the fast entry carries the cheap 64x64
// compact-product guard inline and tail-calls the slow entry
// `d128_multiply_slow_rnd_ctxnull` for every other case. The slow entry does the full
// 128x128->256 product (with its own fits-in-112 early-out before finalize), plus the
// FNZ and non-finite cases. Square's fast entry carries an x^2-specialized guard and
// tail-calls the same slow entry.

package decimal128

func d128_multiply_tte(x, y d128) d128 {
	return d128_multiply_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
}
func d128_multiply_rnd(x, y d128, rnd round) d128 { return d128_multiply_rnd_ctxnull(x, y, rnd, nil) }
func d128_multiply_ctx(x, y d128, ctx *context) d128 {
	return d128_multiply_rnd_ctxnull(x, y, ctx.round, ctx)
}

// Fast entry: both coefficients fit in 64 bits, the exponent sum is in range,
// and the 64*64 product fits in 112 bits (a canonical coefficient): returns
// directly, no rounding, no flags. Every other case tail-calls the out-of-line
// d128_multiply_slow_rnd_ctxnull.
func d128_multiply_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Go has no always_inline, and the fast entry's guard + inline arithmetic exceeds
	// the ~80 inline budget, so (as with add) it does not collapse into Multiply; the
	// split's Go payoff is hot/cold code separation.
	// ubdHi64 is [ qExp:14 | sign:1 | coeffHi:49 ]; a 15-bit left shift drops
	// sign+qExp, so the OR is zero iff both coeffHi49 fields are zero (both
	// coefficients fit in 64 bits) -> no per-operand mask needed.
	allHiBits := (x.ubdHi64 | y.ubdHi64) << 15
	prodQExp := d128_qExp(x) + d128_qExp(y)
	if allHiBits == 0 && prodQExp >= -6176 && prodQExp <= 6111 {
		prodSignFlag := d128_signFlag(x) != d128_signFlag(y)
		prod := mul_128_64x64(x.ubdLo64, y.ubdLo64)
		if ushr_64_64(prod.dw1, 48) == 0 {
			return d128Pack(prodSignFlag, prodQExp, prod.dw0, prod.dw1)
		}
	}
	return d128_multiply_slow_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Slow entry: the wide 128*128->256 product (with its own fits-in-112
// early-out before finalize), plus the zero and non-finite cases. Kept out of
// line so the fast entry stays small; square's fast entry also tail-calls
// it.
//
//go:noinline
func d128_multiply_slow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// The go:noinline directive keeps this out of line so the fast entry stays a
	// small, i-cache-friendly hot function.
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
		return finalize_256_ctxnull(prodSignFlag, prodQExp, prod, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if (d128_isINF(x) && d128_isZER(y)) || (d128_isINF(y) && d128_isZER(x)) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity, const_D128_QNAN0)
	}
	return d128_infinity(prodSignFlag)
}

func d128_square_tte(x d128) d128 {
	return d128_square_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil)
}
func d128_square_rnd(x d128, rnd round) d128 { return d128_square_rnd_ctxnull(x, rnd, nil) }
func d128_square_ctx(x d128, ctx *context) d128 {
	return d128_square_rnd_ctxnull(x, ctx.round, ctx)
}

// Square: x*x. The fast entry carries a square-specialized compact guard (one
// operand to check; the sign is always non-negative) and tail-calls the
// multiply slow entry for everything else.
func d128_square_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	// See d128_multiply_rnd_ctxnull: the 15-bit left shift drops sign+qExp, so
	// this is zero iff x's coeffHi49 is zero (x fits in 64 bits).
	allHiBits := x.ubdHi64 << 15
	xQ := d128_qExp(x)
	prodQExp := xQ + xQ
	if allHiBits == 0 && prodQExp >= -6176 && prodQExp <= 6111 {
		prod := mul_128_64x64(x.ubdLo64, x.ubdLo64)
		if ushr_64_64(prod.dw1, 48) == 0 {
			return d128Pack(false, prodQExp, prod.dw0, prod.dw1) // a square is non-negative
		}
	}
	return d128_multiply_slow_rnd_ctxnull(x, x, rounding, ctxnull)
}
