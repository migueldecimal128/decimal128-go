// IEEE 754 multiply and square. `d128_multiply_rnd_ctxnull` is the general body; the
// `_tte` (bare), `_rnd`, and `_ctx` forms are thin specializations of it (rounding/ctx
// folded in). Fast/slow split (as in add): the fast entry carries the cheap 64x64
// compact-product guard inline and tail-calls the slow entry
// `d128_multiply_slow_rnd_ctxnull` for every other case. The slow entry does the full
// 128x128->256 product (with its own fits-in-112 early-out before finalize), plus the
// FNZ and non-finite cases. Square's fast entry carries an x^2-specialized guard and
// tail-calls the same slow entry.

package decimal128

import "math/bits"

// d128_multiply_tte is the ties-to-even multiply without a context, in one frame:
// d128_multiply_rnd_ctxnull's compact product first, then the 128x128 product and
// its rounding. The general rounding (finalize_256_ctxnull: a digit count, a
// reciprocal division that classifies the discarded digits as a residue, then
// roundAndPack) serves every rounding mode and the context flags; ties to even
// without a context needs neither. Half of the divisor is added first, so the
// rounded quotient is the truncated one, and a tie is exactly a zero remainder,
// whose quotient is made even. Anything that is not a plain finite result in range
// (a special operand, an exponent that overflows or reaches the subnormal range)
// goes to d128_multiply_slow_rnd_ctxnull, which gives the same value there and
// handles those. Measured: two frames (a fast entry calling a ties-to-even slow
// entry) cost 1.4 ns on wide products against this one.
func d128_multiply_tte(x, y d128) d128 {
	// The compact product: three tests as one, without a branch. Both
	// coefficients one word, the 64x64 product below 2^112, and the exponent sum
	// in [-4096, 4095] (inside the format: adding 4096 leaves it below 2^13); a
	// sum outside that range but inside the format is packed further down.
	prod := mul_128_64x64(x.ubdLo64, y.ubdLo64)
	prodQExp := d128_qExp(x) + d128_qExp(y)
	if ((x.ubdHi64|y.ubdHi64)<<15)|ushr_64_64(prod.dw1, 48)|ushr_64_64(int64(prodQExp+4096), 13) == 0 {
		// The sign is the XOR of the operands' sign bits (bit 49), taken in place.
		return d128{ubdLo64: prod.dw0, ubdHi64: int64(prodQExp)<<50 | (x.ubdHi64^y.ubdHi64)&(1<<49) | prod.dw1}
	}
	if !bothFinite(x, y) {
		return d128_multiply_slow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
	}
	signBit := (x.ubdHi64 ^ y.ubdHi64) & (1 << 49) // the product's sign, at bit 49
	qExp := prodQExp                               // the product's exponent

	// The product, four words.
	x0, x1 := uint64(x.ubdLo64), uint64(d128_hi49(x))
	y0, y1 := uint64(y.ubdLo64), uint64(d128_hi49(y))
	g00, p0 := bits.Mul64(x0, y0)
	g01, f01 := bits.Mul64(x0, y1)
	g10, f10 := bits.Mul64(x1, y0)
	g11, f11 := bits.Mul64(x1, y1)
	p1, k1 := bits.Add64(g00, f01, 0)
	p1, k2 := bits.Add64(p1, f10, 0)
	p2, k1 := bits.Add64(g01, g10, k1)
	p2, k2 = bits.Add64(p2, f11, k2)
	p3 := g11 + k1 + k2

	// The digits of p: the estimate t from the bit length is the count or one
	// short, and it is short exactly when p - 10^t does not borrow. The bit
	// length is taken by selects, not branches: the products of one band vary in
	// which word is the top one.
	n := bits.Len64(p0)
	if p1 != 0 {
		n = 64 + bits.Len64(p1)
	}
	if p2 != 0 {
		n = 128 + bits.Len64(p2)
	}
	if p3 != 0 {
		n = 192 + bits.Len64(p3)
	}
	t := (n * 1233) >> 12
	var w0, w1, w2, w3 uint64
	if t <= 38 {
		w0, w1 = uint64(pow10_128_dw0(t)), uint64(pow10_128_dw1(t))
	} else {
		w0, w1 = uint64(pow10_256_dw0(t-39)), uint64(pow10_256_dw1(t-39))
		w2, w3 = uint64(pow10_256_dw2(t-39)), uint64(pow10_256_dw3(t-39))
	}
	_, b := bits.Sub64(p0, w0, 0)
	_, b = bits.Sub64(p1, w1, b)
	_, b = bits.Sub64(p2, w2, b)
	_, b = bits.Sub64(p3, w3, b)
	j := t + 1 - int(b) - 34
	if j <= 0 {
		// Fewer than 35 digits: exact.
		if qExp < -6176 || qExp > 6111 {
			return d128_multiply_slow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
		}
		return d128{ubdLo64: int64(p0), ubdHi64: int64(qExp)<<50 | signBit | int64(p1)}
	}
	if qExp+j < -6176 {
		// The subnormal range removes more digits than the precision does.
		return d128_multiply_slow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
	}

	// Biased by half of d = 10^j.
	d0, d1 := uint64(pow10_128_dw0(j)), uint64(pow10_128_dw1(j))
	var c uint64
	p0, c = bits.Add64(p0, d0>>1|d1<<63, 0)
	p1, c = bits.Add64(p1, d1>>1, c)
	p2, c = bits.Add64(p2, 0, c)
	p3 += c

	// a = floor(p / 2^s), below 2^127.
	s := uint(const_RECIP10_128_S[j&63])
	u0, u1, u2 := p0, p1, p2
	if s >= 64 {
		u0, u1, u2 = p1, p2, p3
	}
	sh := s & 63
	a0 := u0>>sh | u1<<(64-sh)
	a1 := u1>>sh | u2<<(64-sh)

	// qe = floor(a * m / 2^128).
	m0 := uint64(const_RECIP10_128_M[(j<<1)&127])
	m1 := uint64(const_RECIP10_128_M[((j<<1)|1)&127])
	h00, _ := bits.Mul64(a0, m0)
	h01, l01 := bits.Mul64(a0, m1)
	h10, l10 := bits.Mul64(a1, m0)
	h11, l11 := bits.Mul64(a1, m1)
	mid, c1 := bits.Add64(h00, l01, 0)
	_, c2 := bits.Add64(mid, l10, 0)
	q0, c1 := bits.Add64(h01, h10, c1)
	q0, c2 = bits.Add64(q0, l11, c2)
	q1 := h11 + c1 + c2

	// r = p - qe * d, exact in 128 bits; one correction when r >= d.
	rh, rl := bits.Mul64(q0, d0)
	rh += q0*d1 + q1*d0
	rl, c = bits.Sub64(p0, rl, 0)
	rh, _ = bits.Sub64(p1, rh, c)
	_, lt := bits.Sub64(rl, d0, 0)
	_, lt = bits.Sub64(rh, d1, lt)
	ge := lt ^ 1 // 1 when r >= d
	q0, c = bits.Add64(q0, ge, 0)
	q1 += c
	rl, c = bits.Sub64(rl, d0&-ge, 0)
	rh, _ = bits.Sub64(rh, d1&-ge, c)
	if rl|rh == 0 {
		q0 &^= 1 // a tie: the even neighbour
	}

	exp := qExp + j
	if q0 == uint64(pow10_128_dw0(34)) && q1 == uint64(pow10_128_dw1(34)) {
		// Rounded up to 10^34: one digit too many.
		q0, q1, exp = uint64(pow10_128_dw0(33)), uint64(pow10_128_dw1(33)), exp+1
	}
	if exp > 6111 {
		return d128_multiply_slow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
	}
	return d128{ubdLo64: int64(q0), ubdHi64: int64(exp)<<50 | signBit | int64(q1)}
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
	if rounding == const_ROUND_05_UP {
		// Checked here, not in the slow entry, so that entry stays as it was.
		return d128_multiply05Up_ctxnull(x, y, ctxnull)
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
	if rounding == const_ROUND_05_UP {
		// Checked here, not in the slow entry, so that entry stays as it was.
		return d128_multiply05Up_ctxnull(x, x, ctxnull)
	}
	return d128_multiply_slow_rnd_ctxnull(x, x, rounding, ctxnull)
}

// d128_multiply05Up_ctxnull is the round-05up route for multiply (round05_up.go): toward
// zero, then fix the last digit.
//
//go:noinline
func d128_multiply05Up_ctxnull(x, y d128, ctxnull *context) d128 {
	tz := newContextRound(const_ROUND_TOWARD_ZERO)
	t := d128_multiply_slow_rnd_ctxnull(x, y, const_ROUND_TOWARD_ZERO, tz)
	return round05Up_finish_ctxnull(t, tz.flags, ctxnull)
}
