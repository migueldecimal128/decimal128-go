// Subtract. The general body (d128_subtract_rnd_ctxnull) underlies the
// _tte/_rnd/_ctx forms. Aligned same-exponent differences take a flag-free fast
// path; otherwise subtract negates y and delegates to add's FNZ×FNZ helper
// (addFnzFnz_rnd_ctxnull, see d128_arith_add.go), except the unaligned
// unlike-sign magnitude subtract handled here by subMagUnalignedFnzFnz.

package decimal128

// Fast path: same exponent. Opposite signs -> magnitude add; same signs ->
// magnitude subtract. A zero magnitude-subtract result falls through to the
// (cold) d128_subtract_slow_rnd_ctxnull, as does any non-fast case, so this
// body stays tiny and inlines (via _tte/_rnd/_ctx) into the - operator.
func d128_subtract_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	xHi49 := d128_hi49(x)
	yHi49 := d128_hi49(y)
	hi49Sum := xHi49 + yHi49
	if ushr_64_64(hi49Sum, 48) == 0 && d128_qExp(x) == d128_qExp(y) {
		if d128_qExpAndSignBit(x) != d128_qExpAndSignBit(y) {
			// Opposite signs, same exponent: this is a magnitude add. Only the sign
			// differs, and x's full UBD word already carries the correct qExp+sign,
			// so add y's coefficient (its top 15 bits are zero) straight onto x's
			// full word and return raw -- no re-extract/rebuild. The guard above
			// (hi49Sum < 2^48) keeps the coefficient carry out of the qExp/sign field.
			sum := add_128_128x128(x.ubdLo64, x.ubdHi64, y.ubdLo64, yHi49)
			return d128{ubdLo64: sum.dw0, ubdHi64: sum.dw1}
		}
		ds := subAbs_d128swap_128x128(x.ubdLo64, xHi49, y.ubdLo64, yHi49)
		if !isZero_bool_128(ds.diffDw0, ds.diffDw1) {
			resultQExpAndSignBit := d128_qExpAndSignBit(x) ^ boolToInt64(ds.swap)
			return d128FromQExpSignBit(resultQExpAndSignBit, ds.diffDw0, ds.diffDw1)
		}
	}
	return d128_subtract_slow_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Cold path, kept out of line (see d128_add_slow_rnd_ctxnull). //go:noinline
// pins it out of line so the fast entry stays the small hot function.
//
//go:noinline
func d128_subtract_slow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	yNeg := d128_negate(y)
	if d128_isNonFinite(x) || d128_isNonFinite(yNeg) || d128_isZER(x) || d128_isZER(yNeg) {
		return d128_add_rnd_ctxnull(x, yNeg, rounding, ctxnull)
	}
	return addFnzFnz_rnd_ctxnull(x, yNeg, rounding, ctxnull)
}

func d128_subtract_tte(x, y d128) d128 {
	return d128_subtract_rnd_ctxnull(x, y, ROUND_TIES_TO_EVEN, nil)
}
func d128_subtract_rnd(x, y d128, rnd round) d128 { return d128_subtract_rnd_ctxnull(x, y, rnd, nil) }
func d128_subtract_ctx(x, y d128, ctx *context) d128 {
	return d128_subtract_rnd_ctxnull(x, y, ctx.round, ctx)
}

func subMagUnalignedFnzFnz_rnd_ctxnull(zSign bool, m, s d128, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return d128_isFNZ(m) && d128_isFNZ(s) })
	verify("", func() bool { return d128_qExp(m) != d128_qExp(s) })
	verify("", func() bool { return cmpMagFnzFnz(m, s) > 0 })
	mQ := d128_qExp(m)
	mCoeff := d128_u128(m)
	mDigitLen := digitLen_int_128(mCoeff.dw0, mCoeff.dw1)
	sQ := d128_qExp(s)
	sCoeff := d128_u128(s)
	sDigitLen := -1
	gap := mQ - sQ
	if gap < 0 {
		prod := mulPow10_128_128(sCoeff.dw0, sCoeff.dw1, -gap)
		diff := sub_128_128x128(mCoeff.dw0, mCoeff.dw1, prod.dw0, prod.dw1)
		return finalize_128_ctxnull(zSign, mQ, diff, RESIDUE_EXACT, rounding, ctxnull)
	}
	verify("", func() bool { return mDigitLen <= 34 })

	// Fully/Borderline swamped fast path at 34-digit scale.
	headroom := 34 - mDigitLen
	shiftM34 := min(gap, headroom)
	qAlign34 := mQ - shiftM34
	shiftS34 := qAlign34 - sQ
	isSwamped := (shiftS34 >= 35)
	if !isSwamped {
		sDigitLen = digitLen_int_128(sCoeff.dw0, sCoeff.dw1)
		isSwamped = (shiftS34 >= sDigitLen)
	}
	if isSwamped {
		var mScaled u128
		if shiftM34 == 0 {
			mScaled = mCoeff
		} else {
			mScaled = mulPow10_128_128(mCoeff.dw0, mCoeff.dw1, shiftM34)
		}
		if cmp_int_128x128_pow10(mScaled.dw0, mScaled.dw1, 33) != 0 { // not 10^33
			raw := RESIDUE_LT_HALF
			if !(shiftS34 >= 35 || shiftS34 > sDigitLen) {
				raw = residueFromDecadeWithDigitLen_128(sCoeff.dw0, sCoeff.dw1, sDigitLen)
			}
			z := sub_128_128x128(mScaled.dw0, mScaled.dw1, 1, 0)
			return finalizeSwampedFnz_128_ctxnull(zSign, qAlign34, z, residueSubtractionInverse(raw), rounding, ctxnull)
		}
		// mScaled == 10^33: fall through to the guard path.
	}

	// Guard-digit path (the 10^33 exception and partial overlap).
	headroomWithGuard := 1 + headroom
	shiftMLeft := min(gap, headroomWithGuard)
	qAlign := mQ - shiftMLeft
	shiftSRight := qAlign - sQ
	verify("", func() bool { return shiftSRight >= 0 })

	zCoeff := mulPow10_128_128(mCoeff.dw0, mCoeff.dw1, shiftMLeft)
	if sDigitLen == -1 {
		sDigitLen = digitLen_int_128(sCoeff.dw0, sCoeff.dw1)
	}
	if shiftSRight > sDigitLen {
		borrowed := sub_128_128x128(zCoeff.dw0, zCoeff.dw1, 1, 0)
		return finalizeAddSubFnz_128_ctxnull(zSign, qAlign, borrowed, residueSubtractionInverse(RESIDUE_LT_HALF), rounding, ctxnull)
	}
	qr := div_128_pow10_checked_q128res(sCoeff, shiftSRight)
	rawResidue := qr.residue
	zCoeff = sub_128_128x128(zCoeff.dw0, zCoeff.dw1, qr.quotDw0, qr.quotDw1)
	res := RESIDUE_EXACT
	if rawResidue != RESIDUE_EXACT {
		res = residueSubtractionInverse(rawResidue)
		zCoeff = sub_128_128x128(zCoeff.dw0, zCoeff.dw1, 1, 0)
	}
	return finalizeAddSubFnz_128_ctxnull(zSign, qAlign, zCoeff, res, rounding, ctxnull)
}
