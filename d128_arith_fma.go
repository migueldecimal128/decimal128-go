// Fused multiply-add: x*y + a with a single rounding. The fast path (product
// fits 128 bits, in range, a finite) makes the product an exact D128 and reuses
// add. The slow path keeps the product as a wide U256, widens a to U256, aligns,
// adds, and finalizes once — so the whole fma rounds exactly once.

package decimal128

func d128_fma_rnd_ctxnull(x, y, a d128, rounding round, ctxnull *context) d128 {
	prod := mul_256_128x128(x.ubdLo64, d128_hi49(x), y.ubdLo64, d128_hi49(y))
	prodQExp := d128_qExp(x) + d128_qExp(y)
	prodSignFlag := d128_signFlag(x) != d128_signFlag(y)
	if (prod.dw3|prod.dw2) == 0 && ushr_64_64(prod.dw1, 48) == 0 &&
		prodQExp >= -6176 && prodQExp <= 6111 && d128_isFIN(a) {
		lo := prod.lo()
		product := d128Pack(prodSignFlag, prodQExp, lo.dw0, lo.dw1)
		return d128_add_rnd_ctxnull(product, a, rounding, ctxnull)
	}
	return fmaSlow_rnd_ctxnull(x, y, a, prod.hi(), prod.lo(), rounding, ctxnull)
}

func d128_fma_tte(x, y, a d128) d128            { return d128_fma_rnd_ctxnull(x, y, a, TIES_TO_EVEN, nil) }
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
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity)
		}
		if d128_isINF(x) || d128_isINF(y) {
			prodSign := d128_signFlag(x) != d128_signFlag(y)
			if prodSign != d128_signFlag(a) {
				return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMagnitudeSubtractionOfInfinities)
			}
		}
		return a
	}
	if (d128_isZER(x) && d128_isINF(y)) || (d128_isINF(x) && d128_isZER(y)) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMulZeroByInfinity)
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
	zeroSign := rounding == TOWARD_NEGATIVE
	return addWideAndFinalize_rnd_ctxnull(
		prodSign, prodQ, prodU256,
		d128_signFlag(a), d128_qExp(a), u256{aCoeff.dw0, aCoeff.dw1, 0, 0},
		zeroSign, rounding, ctxnull)
}

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
		return ctxSignalInvalidOperationResult(ctxnull, causeSnanOperand, quieted)
	}
	aIsTheOnlyNaN := !d128_isNAN(x) && !d128_isNAN(y) && d128_isNAN(a)
	if aIsTheOnlyNaN {
		if (d128_isZER(x) && d128_isINF(y)) || (d128_isINF(x) && d128_isZER(y)) {
			return ctxSignalInvalidOperationResult(ctxnull, causeMulZeroByInfinity, quieted)
		}
	}
	return quieted
}

func fmaZeroProd_ctxnull(x, y, a d128, rounding round, ctxnull *context) d128 {
	prodSign := d128_signFlag(x) != d128_signFlag(y)
	prodQ := d128_qExp(x) + d128_qExp(y)
	if d128_isZER(a) {
		var fmaSign bool
		if prodSign == d128_signFlag(a) {
			fmaSign = prodSign
		} else {
			fmaSign = rounding == TOWARD_NEGATIVE
		}
		return d128_zero(fmaSign, min(prodQ, d128_qExp(a)))
	}
	if prodQ < d128_qExp(a) {
		return preferMinimumQExp_exact(a, prodQ)
	}
	return a
}

func fmaInfProdCtx(infSign bool, a d128) d128 {
	verify("", !d128_isNAN(a) && d128_isFIN(a))
	return d128_infinity(infSign)
}

func fmaFnzFnzZer_ctxnull(prodSign bool, prodQ int, prodHi, prodLo u128, a d128, rounding round, ctxnull *context) d128 {
	prodU256 := u256FromHiLo(prodHi, prodLo)
	productD128 := finalize_256_ctxnull(prodSign, prodQ, prodU256, RESIDUE_EXACT, rounding, ctxnull)
	if d128_isFIN(productD128) && !d128_isZER(productD128) {
		return preferMinimumQExp_exact(productD128, d128_qExp(a))
	}
	return productD128
}

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

func addWideAndFinalizeLikeSign_rnd_ctxnull(sign bool, mQ int, mCoeff u256, nQ int, nCoeff u256, rounding round, ctxnull *context) d128 {
	qDelta := mQ - nQ
	verify("", qDelta >= 0)
	mDigitLen := digitLen_int_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3)
	headroom := max(0, 34-mDigitLen)
	shiftLeft := min(qDelta, headroom)
	mAligned := mCoeff
	if shiftLeft != 0 {
		mAligned = mulPow10_256_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3, shiftLeft)
	}
	qAlign := mQ - shiftLeft
	gap := qAlign - nQ
	verify("", gap >= 0)
	qr := div_256_pow10_checked_q256res(nCoeff, gap)
	nAligned := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
	coeffSum := add_256_256x256(mAligned.dw0, mAligned.dw1, mAligned.dw2, mAligned.dw3, nAligned.dw0, nAligned.dw1, nAligned.dw2, nAligned.dw3)
	return finalize_256_ctxnull(sign, qAlign, coeffSum, qr.residue, rounding, ctxnull)
}

func addWideAndFinalizeUnlikeSign_rnd_ctxnull(mSign bool, mQ int, mCoeff u256, nSign bool, nQ int, nCoeff u256, zeroSign bool, rounding round, ctxnull *context) d128 {
	qDelta := mQ - nQ
	verify("", qDelta >= 0)
	mDigitLen := digitLen_int_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3)
	nDigitLen := digitLen_int_256(nCoeff.dw0, nCoeff.dw1, nCoeff.dw2, nCoeff.dw3)
	maxCancellation := min(mDigitLen, nDigitLen)
	headroom := max(0, 34+maxCancellation+1-mDigitLen)
	shiftLeft := min(qDelta, headroom)
	mAligned := mCoeff
	if shiftLeft != 0 {
		mAligned = mulPow10_256_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3, shiftLeft)
	}
	qAlign := mQ - shiftLeft
	gap := qAlign - nQ
	verify("", gap >= 0)
	qr := div_256_pow10_checked_q256res(nCoeff, gap)
	nAligned := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
	residue := qr.residue

	ds := subAbs_d256swap_256x256(mAligned.dw0, mAligned.dw1, mAligned.dw2, mAligned.dw3, nAligned.dw0, nAligned.dw1, nAligned.dw2, nAligned.dw3)
	diff := u256{ds.diffDw0, ds.diffDw1, ds.diffDw2, ds.diffDw3}
	if diff == (u256{}) && residue == RESIDUE_EXACT {
		return d128_zero(zeroSign, qAlign)
	}
	if ds.swap {
		return finalize_256_ctxnull(nSign, qAlign, diff, residue, rounding, ctxnull)
	}
	if residue == RESIDUE_EXACT {
		return finalize_256_ctxnull(mSign, qAlign, diff, RESIDUE_EXACT, rounding, ctxnull)
	}
	diffMinus1 := sub_256_256x256(diff.dw0, diff.dw1, diff.dw2, diff.dw3, 1, 0, 0, 0)
	return finalize_256_ctxnull(mSign, qAlign, diffMinus1, residueSubtractionInverse(residue), rounding, ctxnull)
}
