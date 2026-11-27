// Fused multiply-add: x * y + a with a single rounding (IEEE 754-2019).
// `d128_fma_rnd_ctxnull` is the general body; the `_tte`, `_rnd` and `_ctx` forms are
// thin specializations. The fast path (product fits in 128 bits, exponent in range, a
// finite) makes the product an exact D128 and reuses add. The slow path keeps the
// product as a wide U256, widens a to U256, aligns, adds and finalizes once, so the
// whole fma rounds exactly once.

package decimal128

import "math/bits"

// `d128_fma_rnd_ctxnull` is the general body; the `_tte` (bare), `_rnd`,
// and `_ctx` forms are thin specializations.
func d128_fma_rnd_ctxnull(x, y, a d128, rounding round, ctxnull *context) d128 {
	prod := mul_256_128x128(x.ubdLo64, d128_hi49(x), y.ubdLo64, d128_hi49(y))
	prodQExp := d128_qExp(x) + d128_qExp(y)
	prodSignFlag := d128_signFlag(x) != d128_signFlag(y)
	// Fast path: product fits in 128 bits, exponent in range, x, y and a finite.
	// x and y must be checked too: a special's qExp is an encoding code, and a
	// zero coefficient makes the product "fit", so the range test alone lets
	// Inf/NaN times a low-exponent zero through (CairoVectorsWorkOrder.md §3.2).
	if (prod.dw3|prod.dw2) == 0 && ushr_64_64(prod.dw1, 48) == 0 &&
		prodQExp >= -6176 && prodQExp <= 6111 &&
		d128_isFIN(x) && d128_isFIN(y) && d128_isFIN(a) {
		lo := prod.lo()
		product := d128Pack(prodSignFlag, prodQExp, lo.dw0, lo.dw1)
		return d128_add_rnd_ctxnull(product, a, rounding, ctxnull)
	}
	return fmaSlow_rnd_ctxnull(x, y, a, prod.hi(), prod.lo(), rounding, ctxnull)
}

// d128_fma_tte is the ties-to-even x * y + a without a context, in one frame for
// finite nonzero operands (GoUbdKernelGapWorkOrder.md, FMA). The product p is
// formed exactly (up to 68 digits); m is the operand with the larger exponent and
// n the other, G apart.
//
//   - Exact alignment (digits(m) + G <= 76): m * 10^G +- n is exact in four words
//     at n's exponent, and is rounded once to 34 digits: half of 10^j is added and
//     one reciprocal division by 10^j (divRemPow10_256) gives the quotient; a zero
//     remainder is a tie (made even). An exact cancellation is +0.
//   - The addend far above the product (m = a, past exact alignment): a is scaled
//     to 34 digits and p / 10^d, rounded half up the same way, is added or
//     subtracted, as in d128_add_tte's rounding path; a sum of 35 digits is
//     rounded once more from the truncated sum and whether anything below is
//     nonzero.
//
// A product of at most 112 bits is packed and added by d128_add_tte. Everything
// else (specials, zeros, a product far above the addend, results that leave the
// format or lose a digit) goes to d128_fma_rnd_ctxnull, which gives the same value
// there.
func d128_fma_tte(x, y, a d128) d128 {
	xl, xh, yl, yh := uint64(x.ubdLo64), uint64(d128_hi49(x)), uint64(y.ubdLo64), uint64(d128_hi49(y))
	al, ah := uint64(a.ubdLo64), uint64(d128_hi49(a))
	if !bothFinite(x, y) || !d128_isFIN(a) || xl|xh == 0 || yl|yh == 0 || al|ah == 0 {
		return d128_fma_rnd_ctxnull(x, y, a, const_ROUND_TIES_TO_EVEN, nil)
	}
	// The product, four words.
	g00, p0 := bits.Mul64(xl, yl)
	g01, f01 := bits.Mul64(xl, yh)
	g10, f10 := bits.Mul64(xh, yl)
	g11, f11 := bits.Mul64(xh, yh)
	p1, k1 := bits.Add64(g00, f01, 0)
	p1, k2 := bits.Add64(p1, f10, 0)
	p2, k1 := bits.Add64(g01, g10, k1)
	p2, k2 = bits.Add64(p2, f11, k2)
	p3 := g11 + k1 + k2
	ep := d128_qExp(x) + d128_qExp(y)
	ea := d128_qExp(a)
	pSign := (x.ubdHi64 ^ y.ubdHi64) & (1 << 49)
	aSign := a.ubdHi64 & (1 << 49)
	if p3|p2 == 0 && p1>>48 == 0 && ep >= -6176 && ep <= 6111 {
		return d128_add_tte(d128{ubdLo64: int64(p0), ubdHi64: int64(ep)<<50 | pSign | int64(p1)}, a)
	}
	dp := digitLen_u256(p0, p1, p2, p3)
	da := digitLen_u128(al, ah)

	var s0, s1, s2, s3 uint64 // the exact sum, or a scaled to 34 digits
	var exp int
	var sign int64
	if ep >= ea && dp+ep-ea <= 76 || ea > ep && da+ea-ep <= 76 {
		// Exact alignment: m * 10^G at n's exponent, then +- n.
		var m0, m1, m2, m3, n0, n1, n2, n3 uint64
		var g int
		var mSign int64
		if ep >= ea {
			m0, m1, m2, m3, n0, n1, mSign, g, exp = p0, p1, p2, p3, al, ah, pSign, ep-ea, ea
		} else {
			m0, m1, n0, n1, n2, n3, mSign, g, exp = al, ah, p0, p1, p2, p3, aSign, ea-ep, ep
		}
		nSign := (pSign ^ aSign) ^ mSign
		if g > 0 {
			sc := mulPow10_256_256(int64(m0), int64(m1), int64(m2), int64(m3), g)
			m0, m1, m2, m3 = uint64(sc.dw0), uint64(sc.dw1), uint64(sc.dw2), uint64(sc.dw3)
		}
		var c uint64
		sign = mSign
		if pSign == aSign {
			s0, c = bits.Add64(m0, n0, 0)
			s1, c = bits.Add64(m1, n1, c)
			s2, c = bits.Add64(m2, n2, c)
			s3 = m3 + n3 + c
		} else {
			s0, c = bits.Sub64(m0, n0, 0)
			s1, c = bits.Sub64(m1, n1, c)
			s2, c = bits.Sub64(m2, n2, c)
			s3, c = bits.Sub64(m3, n3, c)
			if c != 0 {
				// n was the larger: negate, take its sign.
				s0, c = bits.Sub64(0, s0, 0)
				s1, c = bits.Sub64(0, s1, c)
				s2, c = bits.Sub64(0, s2, c)
				s3, _ = bits.Sub64(0, s3, c)
				sign = nSign
			}
			if s0|s1|s2|s3 == 0 && exp >= -6176 && exp <= 6111 {
				return d128{ubdHi64: int64(exp) << 50} // an exact cancellation: +0
			}
		}
		// Rounded once to 34 digits.
		if j := digitLen_u256(s0, s1, s2, s3) - 34; j > 0 {
			if exp+j < -6176 {
				// The subnormal range removes more digits than the precision does.
				return d128_fma_rnd_ctxnull(x, y, a, const_ROUND_TIES_TO_EVEN, nil)
			}
			h0, h1, h2, h3 := pow10_4(j)
			h0, h1, h2, h3 = h0>>1|h1<<63, h1>>1|h2<<63, h2>>1|h3<<63, h3>>1
			s0, c = bits.Add64(s0, h0, 0)
			s1, c = bits.Add64(s1, h1, c)
			s2, c = bits.Add64(s2, h2, c)
			s3 += h3 + c
			var r0, r1, r2, r3 uint64
			s0, s1, r0, r1, r2, r3 = divRemPow10_256(s0, s1, s2, s3, j)
			s2, s3 = 0, 0 // the quotient is below 10^34 + 1
			if r0|r1|r2|r3 == 0 {
				s0 &^= 1 // a tie: the even neighbour
			}
			exp += j
			if s1 == uint64(pow10_128_dw1(34)) && s0 == uint64(pow10_128_dw0(34)) {
				s0, s1, exp = uint64(pow10_128_dw0(33)), uint64(pow10_128_dw1(33)), exp+1
			}
		}
	} else if ea > ep {
		// The addend far above the product: a scaled to 34 digits, p / 10^d
		// rounded half up beside it (d > 42 here, so the quotient is small; past
		// 76, p is below half a unit and only says the result is inexact).
		m := 34 - da
		t0, t1 := uint64(pow10_128_dw0(m)), uint64(pow10_128_dw1(m))
		s1, s0 = bits.Mul64(al, t0)
		s1 += al*t1 + ah*t0
		exp = ea - m
		d := exp - ep
		var q0, q1, h0, h1, h2, h3 uint64
		rbZero, rbBelowHalf, rbHalf := false, false, false
		if d <= 76 {
			h0, h1, h2, h3 = pow10_4(d)
			h0, h1, h2, h3 = h0>>1|h1<<63, h1>>1|h2<<63, h2>>1|h3<<63, h3>>1
			var c uint64
			t0, c := bits.Add64(p0, h0, 0)
			t1, c := bits.Add64(p1, h1, c)
			t2, c := bits.Add64(p2, h2, c)
			t3 := p3 + h3 + c
			var r0, r1, r2, r3 uint64
			q0, q1, r0, r1, r2, r3 = divRemPow10_256(t0, t1, t2, t3, d)
			rbZero = r0|r1|r2|r3 == 0
			_, lt := bits.Sub64(r0, h0, 0)
			_, lt = bits.Sub64(r1, h1, lt)
			_, lt = bits.Sub64(r2, h2, lt)
			_, lt = bits.Sub64(r3, h3, lt)
			rbBelowHalf = lt != 0
			rbHalf = r0 == h0 && r1 == h1 && r2 == h2 && r3 == h3
		}
		sign = aSign // a scaled is the larger magnitude
		var c uint64
		if pSign == aSign {
			s0, c = bits.Add64(s0, q0, 0)
			s1 += q1 + c
			if _, bw := bits.Sub64(s0, uint64(pow10_128_dw0(34)), 0); true {
				if _, bw = bits.Sub64(s1, uint64(pow10_128_dw1(34)), bw); bw == 0 {
					// 35 digits: rounded once more, at exp + 1, from the
					// truncated sum and whether anything below it is nonzero.
					if d <= 76 && rbBelowHalf {
						s0, c = bits.Sub64(s0, 1, 0)
						s1 -= c
					}
					sticky := d > 76 || !rbHalf
					var u uint64
					s0, s1, u, _ = divRemPow10_recip128(s0, s1, 1)
					if u > 5 || u == 5 && (sticky || s0&1 != 0) {
						s0, c = bits.Add64(s0, 1, 0)
						s1 += c
					}
					exp++
				} else if rbZero {
					s0 &^= 1 // a tie: the even neighbour
				}
			}
		} else {
			s0, c = bits.Sub64(s0, q0, 0)
			s1 -= q1 + c
			if rbZero {
				s0, c = bits.Add64(s0, s0&1, 0) // halfway: the even one
				s1 += c
			}
			_, bw := bits.Sub64(uint64(pow10_128_dw0(33)), s0, 0)
			if _, bw = bits.Sub64(uint64(pow10_128_dw1(33)), s1, bw); bw == 0 {
				// At most 10^33: a digit was lost to the scale.
				return d128_fma_rnd_ctxnull(x, y, a, const_ROUND_TIES_TO_EVEN, nil)
			}
		}
	} else {
		return d128_fma_rnd_ctxnull(x, y, a, const_ROUND_TIES_TO_EVEN, nil)
	}
	if s3|s2 != 0 || exp < -6176 || exp > 6111 {
		return d128_fma_rnd_ctxnull(x, y, a, const_ROUND_TIES_TO_EVEN, nil)
	}
	if _, bw := bits.Sub64(s0, uint64(pow10_128_dw0(34)), 0); true {
		if _, bw = bits.Sub64(s1, uint64(pow10_128_dw1(34)), bw); bw == 0 {
			return d128_fma_rnd_ctxnull(x, y, a, const_ROUND_TIES_TO_EVEN, nil)
		}
	}
	return d128{ubdLo64: int64(s0), ubdHi64: int64(exp)<<50 | sign | int64(s1)}
}
func d128_fma_rnd(x, y, a d128, rnd round) d128 { return d128_fma_rnd_ctxnull(x, y, a, rnd, nil) }
func d128_fma_ctx(x, y, a d128, ctx *context) d128 {
	return d128_fma_rnd_ctxnull(x, y, a, ctx.round, ctx)
}

func fmaSlow_rnd_ctxnull(x, y, a d128, prodHi, prodLo u128, rounding round, ctxnull *context) d128 {
	if d128_isNAN(x) || d128_isNAN(y) || d128_isNAN(a) {
		return fmaNanOperand_ctxnull(x, y, a, ctxnull)
	}
	if d128_isINF(a) {
		if (d128_isZER(x) && d128_isINF(y)) || (d128_isINF(x) && d128_isZER(y)) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity, const_D128_QNAN0)
		}
		if d128_isINF(x) || d128_isINF(y) {
			prodSign := d128_signFlag(x) != d128_signFlag(y)
			if prodSign != d128_signFlag(a) {
				return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMagnitudeSubtractionOfInfinities, const_D128_QNAN0)
			}
		}
		return a
	}
	if (d128_isZER(x) && d128_isINF(y)) || (d128_isINF(x) && d128_isZER(y)) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity, const_D128_QNAN0)
	}
	if d128_isZER(x) || d128_isZER(y) {
		return fmaZeroProd_ctxnull(x, y, a, rounding, ctxnull)
	}
	if d128_isINF(x) || d128_isINF(y) {
		prodSign := d128_signFlag(x) != d128_signFlag(y)
		return fmaInfProdCtx(prodSign, a)
	}

	prodSign := d128_signFlag(x) != d128_signFlag(y)
	prodQ := d128_qExp(x) + d128_qExp(y)
	if d128_isZER(a) {
		return fmaFnzFnzZer_ctxnull(prodSign, prodQ, prodHi, prodLo, a, rounding, ctxnull)
	}

	prodU256 := u256FromHiLo(prodHi, prodLo)
	aCoeff := d128_u128(a)
	zeroSign := rounding == const_ROUND_TOWARD_NEGATIVE
	return addWideAndFinalize_rnd_ctxnull(
		prodSign, prodQ, prodU256,
		d128_signFlag(a), d128_qExp(a), u256{aCoeff.dw0, aCoeff.dw1, 0, 0},
		zeroSign, rounding, ctxnull)
}

// NaN propagation across three operands. sNaN priority: if any operand is
// sNaN, signal invalid-op and pick an sNaN to quiet; otherwise pick the
// first qNaN in (x, y, a) order. IEEE 754 also requires invalid-op to be
// raised if the product would have been 0*INF, even when a NaN is being
// propagated.
func fmaNanOperand_ctxnull(x, y, a d128, ctxnull *context) d128 {
	hasSNAN := d128_isSNAN(x) || d128_isSNAN(y) || d128_isSNAN(a)
	var pickedNaN d128
	if hasSNAN {
		if d128_isSNAN(x) {
			pickedNaN = x
		} else if d128_isSNAN(y) {
			pickedNaN = y
		} else {
			pickedNaN = a
		}
	} else {
		if d128_isNAN(x) {
			pickedNaN = x
		} else if d128_isNAN(y) {
			pickedNaN = y
		} else {
			pickedNaN = a
		}
	}
	quieted := d128_quieted(pickedNaN)
	if hasSNAN {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeSnanOperand, quieted)
	}
	aIsTheOnlyNaN := !d128_isNAN(x) && !d128_isNAN(y) && d128_isNAN(a)
	if aIsTheOnlyNaN {
		if (d128_isZER(x) && d128_isINF(y)) || (d128_isINF(x) && d128_isZER(y)) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity, quieted)
		}
	}
	return quieted
}

// Product is zero (x or y is ZER, both finite). a is FIN. Determine the
// signed-zero or scaled-FNZ result per IEEE 754.
func fmaZeroProd_ctxnull(x, y, a d128, rounding round, ctxnull *context) d128 {
	prodSign := d128_signFlag(x) != d128_signFlag(y)
	prodQ := d128_qExp(x) + d128_qExp(y)
	if d128_isZER(a) {
		var fmaSign bool
		if prodSign == d128_signFlag(a) {
			fmaSign = prodSign
		} else {
			fmaSign = rounding == const_ROUND_TOWARD_NEGATIVE
		}
		return d128_zero(fmaSign, min(prodQ, d128_qExp(a)))
	}
	if prodQ < d128_qExp(a) {
		return preferMinimumQExp_exact(a, prodQ)
	}
	return a
}

// Product is infinite. a is FIN. The product dominates; result is the
// signed infinity.
func fmaInfProdCtx(infSign bool, a d128) d128 {
	verify("", func() bool { return !d128_isNAN(a) && d128_isFIN(a) })
	return d128_infinity(infSign)
}

// FNZ * FNZ + (signed zero). Result equals the product with preferred-
// exponent rule: result qExp = min(prodQ, aQ) when achievable.
func fmaFnzFnzZer_ctxnull(prodSign bool, prodQ int, prodHi, prodLo u128, a d128, rounding round, ctxnull *context) d128 {
	prodU256 := u256FromHiLo(prodHi, prodLo)
	productD128 := finalize_256_ctxnull(prodSign, prodQ, prodU256, const_RESIDUE_EXACT, rounding, ctxnull)
	if d128_isFIN(productD128) && !d128_isZER(productD128) {
		return preferMinimumQExp_exact(productD128, d128_qExp(a))
	}
	return productD128
}

// const_FMA_EXACT_ALIGN_DIGITS is how far the larger-exponent operand may be scaled up so
// that alignment is exact. If scaling m by the whole exponent gap keeps it within
// this many digits, m is scaled all the way, nothing is shifted off n, and
// finalize_256 does the one and only divide. Any bound up to 76 is correct (both
// aligned operands are then below 10^76, so their sum is below 2*10^76 < 2^256);
// 69 covers the product's 68 digits plus one, and measured the same as 71.
// Past it, m is scaled only toward the 34-digit result precision and n is shifted
// right with a residue, which keeps the far-addend case on finalize's 128-bit path.
const const_FMA_EXACT_ALIGN_DIGITS = 69

func addWideAndFinalize_rnd_ctxnull(prodSign bool, prodQ int, prodCoeff u256, aSign bool, aQ int, aCoeff u256, zeroSign bool, rounding round, ctxnull *context) d128 {
	var mSign, nSign bool
	var mQ, nQ int
	var mCoeff, nCoeff u256
	if prodQ >= aQ {
		mSign, mQ, mCoeff = prodSign, prodQ, prodCoeff
		nSign, nQ, nCoeff = aSign, aQ, aCoeff
	} else {
		mSign, mQ, mCoeff = aSign, aQ, aCoeff
		nSign, nQ, nCoeff = prodSign, prodQ, prodCoeff
	}
	if mSign == nSign {
		return addWideAndFinalizeLikeSign_rnd_ctxnull(mSign, mQ, mCoeff, nQ, nCoeff, rounding, ctxnull)
	}
	return addWideAndFinalizeUnlikeSign_rnd_ctxnull(mSign, mQ, mCoeff, nSign, nQ, nCoeff, zeroSign, rounding, ctxnull)
}

// Like-sign path. Result magnitude is |m| + |n| >= |m|.
// shiftLeft cap = max(0, targetDigits - m.digitLen()). Truncated bits of n
// are at most half an ULP at qAlign, absorbed by residue.
func addWideAndFinalizeLikeSign_rnd_ctxnull(sign bool, mQ int, mCoeff u256, nQ int, nCoeff u256, rounding round, ctxnull *context) d128 {
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta >= 0 })
	mDigitLen := digitLen_int_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3)
	shiftLeft := qDelta
	if qDelta > const_FMA_EXACT_ALIGN_DIGITS-mDigitLen {
		shiftLeft = min(qDelta, max(0, 34-mDigitLen))
	}
	var mAligned u256
	if shiftLeft == 0 {
		mAligned = mCoeff
	} else {
		mAligned = mulPow10_256_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3, shiftLeft)
	}
	qAlign := mQ - shiftLeft
	gap := qAlign - nQ
	verify("", func() bool { return gap >= 0 })
	qr := div_256_pow10_checked_q256res(nCoeff, gap)
	nAligned := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
	coeffSum := add_256_256x256(mAligned.dw0, mAligned.dw1, mAligned.dw2, mAligned.dw3, nAligned.dw0, nAligned.dw1, nAligned.dw2, nAligned.dw3)
	return finalize_256_ctxnull(sign, qAlign, coeffSum, qr.residue, rounding, ctxnull)
}

func addWideAndFinalizeUnlikeSign_rnd_ctxnull(mSign bool, mQ int, mCoeff u256, nSign bool, nQ int, nCoeff u256, zeroSign bool, rounding round, ctxnull *context) d128 {
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta >= 0 })
	mDigitLen := digitLen_int_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3)
	shiftLeft := qDelta
	if qDelta > const_FMA_EXACT_ALIGN_DIGITS-mDigitLen {
		nDigitLen := digitLen_int_256(nCoeff.dw0, nCoeff.dw1, nCoeff.dw2, nCoeff.dw3)
		maxCancellation := min(mDigitLen, nDigitLen)
		shiftLeft = min(qDelta, max(0, 34+maxCancellation+1-mDigitLen))
	}
	var mAligned u256
	if shiftLeft == 0 {
		mAligned = mCoeff
	} else {
		mAligned = mulPow10_256_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3, shiftLeft)
	}
	qAlign := mQ - shiftLeft
	gap := qAlign - nQ
	verify("", func() bool { return gap >= 0 })
	qr := div_256_pow10_checked_q256res(nCoeff, gap)
	nAligned := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
	residue := qr.residue

	ds := subAbs_d256swap_256x256(mAligned.dw0, mAligned.dw1, mAligned.dw2, mAligned.dw3, nAligned.dw0, nAligned.dw1, nAligned.dw2, nAligned.dw3)
	diff := u256{ds.diffDw0, ds.diffDw1, ds.diffDw2, ds.diffDw3}
	if diff == (u256{}) && residue == const_RESIDUE_EXACT {
		return d128_zero(zeroSign, qAlign)
	}
	if ds.swap {
		return finalize_256_ctxnull(nSign, qAlign, diff, residue, rounding, ctxnull)
	}
	if residue == const_RESIDUE_EXACT {
		return finalize_256_ctxnull(mSign, qAlign, diff, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	diffMinus1 := sub_256_256x256(diff.dw0, diff.dw1, diff.dw2, diff.dw3, 1, 0, 0, 0)
	return finalize_256_ctxnull(mSign, qAlign, diffMinus1, residueSubtractionInverse(residue), rounding, ctxnull)
}
