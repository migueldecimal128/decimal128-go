// Add. The general body (d128_add_rnd_ctxnull) underlies the _tte/_rnd/_ctx
// forms. Aligned same-exponent sums take a flag-free fast path; otherwise the
// FNZ×FNZ paths align (near) or use the unaligned far-exponent helpers with the
// FE finalize entries. The shared addFnzFnz_rnd_ctxnull helper is also reached
// from subtract's negate-and-add path (see d128_arith_sub.go).

package decimal128

func d128_add_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	xHi49 := d128_hi49(x)
	yHi49 := d128_hi49(y)
	hi49Sum := xHi49 + yHi49
	if ushr_64_64(hi49Sum, 48) == 0 && d128_qExpAndSignBit(x) == d128_qExpAndSignBit(y) {
		// Fast path: same sign and exponent, sum fits in 112 bits. No rounding.
		coeffSum := add_128_128x128(x.ubdLo64, xHi49, y.ubdLo64, yHi49)
		return d128FromQExpSignBit(d128_qExpAndSignBit(x), coeffSum.dw0, coeffSum.dw1)
	}
	if d128_isNonFinite(x) || d128_isNonFinite(y) {
		return addNonFinite_rnd_ctxnull(x, y, ctxnull)
	}
	if d128_isZER(x) {
		if d128_isZER(y) {
			resultSign := (d128_signFlag(x) && d128_signFlag(y)) ||
				(d128_signFlag(x) != d128_signFlag(y) && rounding == ROUND_TOWARD_NEGATIVE)
			return d128_zero(resultSign, min(d128_qExp(x), d128_qExp(y)))
		}
		return preferMinimumQExp_exact(y, d128_qExp(x))
	}
	if d128_isZER(y) {
		return preferMinimumQExp_exact(x, d128_qExp(y))
	}
	return addFnzFnz_rnd_ctxnull(x, y, rounding, ctxnull)
}

func d128_add_tte(x, y d128) d128            { return d128_add_rnd_ctxnull(x, y, ROUND_TIES_TO_EVEN, nil) }
func d128_add_rnd(x, y d128, rnd round) d128 { return d128_add_rnd_ctxnull(x, y, rnd, nil) }
func d128_add_ctx(x, y d128, ctx *context) d128 {
	return d128_add_rnd_ctxnull(x, y, ctx.round, ctx)
}

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
				return d128_zero(rounding == ROUND_TOWARD_NEGATIVE, qAligned)
			}
			resultSign = d128_signFlag(x) != ds.swap
			resultCoeff = u128{ds.diffDw0, ds.diffDw1}
		}
		if isLt_pow10_34_bool_128(resultCoeff.dw0, resultCoeff.dw1) {
			return d128Pack(resultSign, qAligned, resultCoeff.dw0, resultCoeff.dw1)
		}
		return finalize_128_ctxnull(resultSign, qAligned, resultCoeff, RESIDUE_EXACT, rounding, ctxnull)
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
	// Swamped fast path: n lies strictly below the result's LSD (LT_HALF residue).
	if shiftRight >= 35 || shiftRight > d128_digitLen(n) {
		return finalizeSwampedFnz_128_ctxnull(d128_signFlag(m), qAlign, zCoeff, RESIDUE_LT_HALF, rounding, ctxnull)
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
	return d128_zero(rounding == ROUND_TOWARD_NEGATIVE, min(d128_qExp(x), d128_qExp(y)))
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
	return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMagnitudeSubtractionOfInfinities)
}

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
