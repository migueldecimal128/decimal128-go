// IEEE 754 add. `d128_add_rnd_ctxnull` is the general body; the `_tte` (bare), `_rnd`,
// and `_ctx` forms are thin specializations of it (rounding/ctx folded in). Its fast
// entry returns directly, with no rounding and no flags, for the same sign, the same
// exponent and coefficients that each fit in 64 bits; every other case tail-calls
// the out-of-line `d128_add_slow_rnd_ctxnull`. Subtract reaches the shared
// `addFnzFnz_rnd_ctxnull` for its aligned magnitude-add subcase.

package decimal128

func d128_add_tte(x, y d128) d128            { return d128_add_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil) }
func d128_add_rnd(x, y d128, rnd round) d128 { return d128_add_rnd_ctxnull(x, y, rnd, nil) }
func d128_add_ctx(x, y d128, ctx *context) d128 {
	return d128_add_rnd_ctxnull(x, y, ctx.round, ctx)
}

// Fast entry: same sign and exponent, and both coefficients below 2^64,
// returns directly (no rounding, no flags). Every other case tail-calls the
// out-of-line d128_add_slow_rnd_ctxnull.
func d128_add_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// One test: (xHi ^ yHi) | xHi << 15 == 0 holds exactly when the high words
	// are equal (same qExp and sign) and hold no coefficient bits, which also
	// excludes the non-finite values (their coefficient bits are set).
	if ((x.ubdHi64 ^ y.ubdHi64) | (x.ubdHi64 << 15)) == 0 {
		// The sum is below 2^65, so canonical; its carry goes into x's high
		// word, whose qExp and sign carry through unchanged.
		sum := add_128_128x128(x.ubdLo64, x.ubdHi64, y.ubdLo64, 0)
		return d128{ubdLo64: sum.dw0, ubdHi64: sum.dw1}
	}
	return d128_add_slow_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Slow entry: the non-finite, zero, and general finite-nonzero cases. Kept out
// of line so the fast entry stays small.
//
//go:noinline
func d128_add_slow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Kept out of line so the fast entry stays a small, i-cache-friendly hot function.
	// Go has no always_inline, and the fast entry's guard + inline arithmetic already
	// exceeds the ~80 inline budget, so it does not collapse into Add (unlike the
	// native ports' operators); the split's Go payoff is hot/cold code separation.
	// The go:noinline directive pins that contract.
	// Aligned operands whose coefficients sum below 2^112, ahead of the cold
	// ladder. hi49Sum fitting in 48 bits proves BOTH operands finite (nonfinite
	// coefficients are >= 10^34, whose bit 48 is set). The same sign adds the
	// two-word coefficients; otherwise the signs differ and the magnitudes
	// subtract. Both coefficients are canonical and unscaled, so the
	// difference is < 10^34 and packs raw; swapped means |x| < |y|, and the
	// result takes the larger magnitude's sign. Mirror of the subtract slow
	// entry's first test.
	xHi49 := d128_hi49(x)
	yHi49 := d128_hi49(y)
	hi49Sum := xHi49 + yHi49
	if ushr_64_64(hi49Sum, 48) == 0 && d128_qExp(x) == d128_qExp(y) {
		if d128_qExpAndSignBit(x) == d128_qExpAndSignBit(y) {
			// Same sign, coefficients of up to 112 bits: the two-word sum.
			// hi49Sum < 2^48, so the sum never carries into the qExp field and
			// x's qExp and sign carry through in its high word.
			sum := add_128_128x128(x.ubdLo64, x.ubdHi64, y.ubdLo64, yHi49)
			return d128{ubdLo64: sum.dw0, ubdHi64: sum.dw1}
		}
		verify("", func() bool { return d128_signFlag(x) != d128_signFlag(y) })
		ds := subAbs_d128swap_128x128(x.ubdLo64, xHi49, y.ubdLo64, yHi49)
		if isZero_bool_128(ds.diffDw0, ds.diffDw1) {
			return d128_zero(rounding == const_ROUND_TOWARD_NEGATIVE, d128_qExp(x))
		}
		resultQExpAndSignBit := d128_qExpAndSignBit(x) ^ boolToInt64(ds.swap)
		return d128FromQExpSignBit(resultQExpAndSignBit, ds.diffDw0, ds.diffDw1)
	}
	if d128_isNonFinite(x) || d128_isNonFinite(y) {
		return addNonFinite_rnd_ctxnull(x, y, ctxnull)
	}
	if d128_isZER(x) {
		if d128_isZER(y) {
			resultSign := (d128_signFlag(x) && d128_signFlag(y)) ||
				(d128_signFlag(x) != d128_signFlag(y) && rounding == const_ROUND_TOWARD_NEGATIVE)
			return d128_zero(resultSign, min(d128_qExp(x), d128_qExp(y)))
		}
		return preferMinimumQExp_exact(y, d128_qExp(x))
	}
	if d128_isZER(y) {
		return preferMinimumQExp_exact(x, d128_qExp(y))
	}
	return addFnzFnz_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Finite-nonzero + finite-nonzero add core: nearly aligned operands are
// aligned exactly, the rest go to the unaligned kernels. Subtract delegates
// here through subtract(x, y) = add(x, negate(y)), so it is module-internal
// rather than file-private.
func addFnzFnz_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	qAligned := min(d128_qExp(x), d128_qExp(y))
	absXShift := d128_qExp(x) - qAligned // one of these is zero
	absYShift := d128_qExp(y) - qAligned
	if absXShift+absYShift <= 4 {
		// Nearly aligned: align to qAligned without overflowing 128 bits.
		var xCoeffAligned u128
		if absXShift == 0 {
			xCoeffAligned = d128_u128(x)
		} else {
			xCoeffAligned = mulPow10_128_128(x.ubdLo64, d128_hi49(x), absXShift)
		}
		var yCoeffAligned u128
		if absYShift == 0 {
			yCoeffAligned = d128_u128(y)
		} else {
			yCoeffAligned = mulPow10_128_128(y.ubdLo64, d128_hi49(y), absYShift)
		}

		var resultSign bool
		var resultCoeff u128
		if d128_signFlag(x) == d128_signFlag(y) {
			resultSign = d128_signFlag(x)
			resultCoeff = add_128_128x128(xCoeffAligned.dw0, xCoeffAligned.dw1, yCoeffAligned.dw0, yCoeffAligned.dw1)
		} else {
			ds := subAbs_d128swap_128x128(xCoeffAligned.dw0, xCoeffAligned.dw1, yCoeffAligned.dw0, yCoeffAligned.dw1)
			if isZero_bool_128(ds.diffDw0, ds.diffDw1) {
				return d128_zero(rounding == const_ROUND_TOWARD_NEGATIVE, qAligned)
			}
			resultSign = d128_signFlag(x) != ds.swap
			resultCoeff = u128{ds.diffDw0, ds.diffDw1}
		}
		if isLt_pow10_34_bool_128(resultCoeff.dw0, resultCoeff.dw1) {
			return d128Pack(resultSign, qAligned, resultCoeff.dw0, resultCoeff.dw1)
		}
		return finalize_128_ctxnull(resultSign, qAligned, resultCoeff, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_signFlag(x) == d128_signFlag(y) {
		return addFnzFnzUnalignedLikeSign_rnd_ctxnull(x, y, rounding, ctxnull)
	}
	return addFnzFnzUnalignedUnlikeSign_rnd_ctxnull(x, y, rounding, ctxnull)
}

func addFnzFnzUnalignedLikeSign_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	xQ := d128_qExp(x)
	yQ := d128_qExp(y)
	verify("", func() bool { return xQ != yQ })
	verify("", func() bool { return d128_signFlag(x) == d128_signFlag(y) })

	var m, n d128
	var mQ, nQ int
	if xQ > yQ {
		m, n = x, y
		mQ = xQ
		nQ = yQ
	} else {
		m, n = y, x
		mQ = yQ
		nQ = xQ
	}
	mDigitLen := d128_digitLen(m)
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta > 0 })
	headroom := 34 - mDigitLen
	shiftLeft := min(max(headroom, 0), qDelta)
	var zCoeff u128
	if shiftLeft == 0 {
		zCoeff = d128_u128(m)
	} else {
		zCoeff = mulPow10_128_128(m.ubdLo64, d128_hi49(m), shiftLeft)
	}
	qAlign := mQ - shiftLeft
	shiftRight := qAlign - nQ
	// MQ FAST TAIL: gap fully absorbed into m's headroom (shiftRight == 0), so n
	// already sits at the result scale and, being the same sign, its coefficient
	// simply adds. The sum is EXACT: skip the div-by-zero and residue finalize.
	if shiftRight == 0 {
		nCoeff := d128_u128(n)
		sum := add_128_128x128(zCoeff.dw0, zCoeff.dw1, nCoeff.dw0, nCoeff.dw1)
		if isLt_pow10_34_bool_128(sum.dw0, sum.dw1) {
			return d128Pack(d128_signFlag(m), qAlign, sum.dw0, sum.dw1)
		}
		return finalize_128_ctxnull(d128_signFlag(m), qAlign, sum, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	// Swamped fast path: n lies strictly below the result's LSD (LT_HALF residue).
	if shiftRight >= 35 || shiftRight > d128_digitLen(n) {
		return finalizeSwampedFnz_128_ctxnull(d128_signFlag(m), qAlign, zCoeff, const_RESIDUE_LT_HALF, rounding, ctxnull)
	}
	qr := div_128_pow10_checked_q128res(d128_u128(n), shiftRight)
	res := qr.residue
	zCoeff = add_128_128x128(zCoeff.dw0, zCoeff.dw1, qr.quotDw0, qr.quotDw1)
	return finalizeAddSubFnz_128_ctxnull(d128_signFlag(m), qAlign, zCoeff, res, rounding, ctxnull)
}

func addFnzFnzUnalignedUnlikeSign_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return d128_qExp(x) != d128_qExp(y) })
	verify("", func() bool { return d128_signFlag(x) != d128_signFlag(y) })
	cmp := cmpMagFnzFnz(x, y)
	if cmp > 0 {
		return subMagUnalignedFnzFnz_rnd_ctxnull(d128_signFlag(x), x, y, rounding, ctxnull)
	}
	if cmp < 0 {
		return subMagUnalignedFnzFnz_rnd_ctxnull(d128_signFlag(y), y, x, rounding, ctxnull)
	}
	return d128_zero(rounding == const_ROUND_TOWARD_NEGATIVE, min(d128_qExp(x), d128_qExp(y)))
}

func addNonFinite_rnd_ctxnull(x, y d128, ctxnull *context) d128 {
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if d128_isFIN(x) {
		return y
	}
	if d128_isFIN(y) {
		return x
	}
	if d128_signFlag(x) == d128_signFlag(y) {
		return x
	}
	return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMagnitudeSubtractionOfInfinities, const_D128_QNAN0)
}

// preferMinimumQExp_exact lowers x's exponent toward otherExp by scaling its coefficient up into the
// free digits, exactly (the value is unchanged). Returns x unchanged when
// there is no headroom or otherExp >= qExp(x). Shared with fma.
func preferMinimumQExp_exact(x d128, otherExp int) d128 {
	xSign := d128_signFlag(x)
	xQ := d128_qExp(x)
	xCoeff := d128_u128(x)
	delta := xQ - otherExp
	if delta <= 0 {
		return x
	}
	headroom := max(0, 34-d128_digitLen(x))
	if headroom == 0 {
		return x
	}
	expRoom := xQ + 6176
	shiftLeft := min(min(headroom, delta), expRoom)
	var zCoeff u128
	if shiftLeft == 0 {
		zCoeff = xCoeff
	} else {
		zCoeff = mulPow10_128_128(xCoeff.dw0, xCoeff.dw1, shiftLeft)
	}
	zQ := xQ - shiftLeft
	verify("", func() bool { return isCanonical34_128(zQ, zCoeff) })
	return d128Pack(xSign, zQ, zCoeff.dw0, zCoeff.dw1)
}
