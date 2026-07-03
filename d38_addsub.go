// Addition / subtraction in the 38-digit D38 domain (round-half-even, no flags,
// no rounding mode), used by the transcendental kernels. Four alignment paths
// mirror the D128 addsub structure but target 38 digits.

package decimal128

func d38_add(x, y d38) d38 {
	// A zero operand must short-circuit: the alignment paths below treat it as a
	// value at its qExp, so adding a far-exponent zero would shift the non-zero
	// operand down and shred its precision (e.g. r + 0 with r ≈ 1e-33).
	if d38_isZero(y) {
		return x
	}
	if d38_isZero(x) {
		return y
	}
	if x.qExpAndSignBit == y.qExpAndSignBit {
		return addAlignedLikeSignD38(x, y)
	}
	if d38_qExp(x) == d38_qExp(y) {
		return addAlignedUnlikeSignD38(x, y)
	}
	if d38_signFlag(x) == d38_signFlag(y) {
		return addUnalignedLikeSignD38(x, y)
	}
	return addUnalignedUnlikeSignD38(x, y)
}

func d38_subtract(x, y d38) d38 {
	return d38_add(x, d38_negated(y))
}

func addAlignedLikeSignD38(x, y d38) d38 {
	// same qExp and sign — unscaled addition
	sum := add_128_128x128(x.coeffLo64, x.coeffHi64, y.coeffLo64, y.coeffHi64)
	if cmp_int_128x128_pow10(sum.dw0, sum.dw1, 38) < 0 {
		return d38Raw(x.qExpAndSignBit, sum.dw0, sum.dw1)
	}
	return finalize38_128(d38_signFlag(x), d38_qExp(x), sum, RESIDUE_EXACT)
}

func addAlignedUnlikeSignD38(x, y d38) d38 {
	verify("", func() bool { return d38_qExp(x) == d38_qExp(y) })
	verify("", func() bool { return d38_signFlag(x) != d38_signFlag(y) })
	ds := subAbs_d128swap_128x128(x.coeffLo64, x.coeffHi64, y.coeffLo64, y.coeffHi64)
	if isZero_bool_128(ds.diffDw0, ds.diffDw1) {
		return D38_ZERO
	}
	// swapped means |x| < |y|: flip the sign
	var flip int32
	if ds.swap {
		flip = 1
	}
	return d38Raw(x.qExpAndSignBit^flip, ds.diffDw0, ds.diffDw1)
}

func addUnalignedLikeSignD38(x, y d38) d38 {
	verify("", func() bool { return d38_digitLen(x) <= 38 && d38_digitLen(y) <= 38 })
	verify("", func() bool { return d38_qExp(x) != d38_qExp(y) })
	verify("", func() bool { return d38_signFlag(x) == d38_signFlag(y) })

	var m, n d38
	if d38_qExp(x) > d38_qExp(y) {
		m, n = x, y
	} else {
		m, n = y, x
	}
	mQ := d38_qExp(m)
	nQ := d38_qExp(n)
	mDigitLen := d38_digitLen(m)
	nDigitLen := d38_digitLen(n)
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta > 0 })
	headroom := 38 - mDigitLen
	shiftLeft := min(headroom, qDelta)
	zU128 := mulPow10_128_128(m.coeffLo64, m.coeffHi64, shiftLeft)
	qAlign := mQ - shiftLeft
	shiftRight := qAlign - nQ
	var res residue
	if shiftRight == 0 {
		verify("", func() bool { return shiftLeft > 0 })
		zU128 = add_128_128x128(zU128.dw0, zU128.dw1, n.coeffLo64, n.coeffHi64)
		res = RESIDUE_EXACT
	} else if shiftRight > nDigitLen {
		if d38_isZero(n) {
			res = RESIDUE_EXACT
		} else {
			res = RESIDUE_LT_HALF
		}
	} else if shiftRight == nDigitLen {
		nU128 := d38_u128(n)
		res = residueFromDecadeWithDigitLen_128(nU128.dw0, nU128.dw1, nDigitLen)
	} else {
		qr := div_128_pow10_q128res(d38_u128(n), shiftRight)
		q := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		res = qr.residue
		zU128 = add_128_128x128(zU128.dw0, zU128.dw1, q.dw0, q.dw1)
	}
	return finalize38_128(d38_signFlag(m), qAlign, zU128, res)
}

func addUnalignedUnlikeSignD38(x, y d38) d38 {
	verify("", func() bool { return d38_qExp(x) != d38_qExp(y) })
	verify("", func() bool { return d38_signFlag(x) != d38_signFlag(y) })
	cmp := cmpMagD38(x, y)
	if cmp > 0 {
		return subMagUnalignedFnzFnzD38(d38_signFlag(x), x, y)
	} else if cmp < 0 {
		return subMagUnalignedFnzFnzD38(d38_signFlag(y), y, x)
	}
	return D38_ZERO
}

func subMagUnalignedFnzFnzD38(zSign bool, m, s d38) d38 {
	verify("", func() bool { return d38_qExp(m) != d38_qExp(s) })
	verify("", func() bool { return cmpMagD38(m, s) > 0 })
	mQ := d38_qExp(m)
	mU128 := d38_u128(m)
	mDigitLen := digitLen_int_128(mU128.dw0, mU128.dw1)
	sQ := d38_qExp(s)
	sU128 := d38_u128(s)
	sDigitLen := digitLen_int_128(sU128.dw0, sU128.dw1)
	gap := mQ - sQ

	if gap < 0 {
		// |m| > |s| but qExp(m) < qExp(s): only possible when m's coefficient is
		// much larger than s's. Scale s up to m's qExp using U256.
		sScaled := mulPow10_256_128(sU128.dw0, sU128.dw1, -gap)
		diff := sub_256_256x256(mU128.dw0, mU128.dw1, 0, 0, sScaled.dw0, sScaled.dw1, sScaled.dw2, sScaled.dw3)
		verify("", func() bool { return isZero_bool_128(diff.dw2, diff.dw3) })
		verify("", func() bool { return cmp_int_128x128(diff.dw0, diff.dw1, POW10_38.dw0, POW10_38.dw1) < 0 })
		return d38Pack(zSign, mQ, diff.dw0, diff.dw1)
	}

	verify("", func() bool { return mDigitLen <= 38 })
	verify("", func() bool { return sDigitLen <= 38 })
	headroomWithGuard := 1 + 38 - mDigitLen
	shiftMLeft := min(gap, headroomWithGuard)
	zCoeff := mulPow10_256_128(mU128.dw0, mU128.dw1, shiftMLeft)

	qAlign := mQ - shiftMLeft
	shiftSRight := qAlign - sQ
	verify("", func() bool { return shiftSRight >= 0 })

	var res residue
	if shiftSRight == 0 {
		zCoeff = sub_256_256x256(zCoeff.dw0, zCoeff.dw1, zCoeff.dw2, zCoeff.dw3, sU128.dw0, sU128.dw1, 0, 0)
		res = RESIDUE_EXACT
	} else if shiftSRight >= sDigitLen {
		if shiftSRight > sDigitLen {
			res = RESIDUE_GT_HALF
		} else {
			res = residueSubtractionInverse(residueFromDecadeWithDigitLen_128(sU128.dw0, sU128.dw1, sDigitLen))
		}
		verify("", func() bool { return res != RESIDUE_EXACT })
		zCoeff = sub_256_256x256(zCoeff.dw0, zCoeff.dw1, zCoeff.dw2, zCoeff.dw3, 1, 0, 0, 0)
	} else {
		qr := div_128_pow10_q128res(sU128, shiftSRight)
		t := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		residueT := qr.residue
		zCoeff = sub_256_256x256(zCoeff.dw0, zCoeff.dw1, zCoeff.dw2, zCoeff.dw3, t.dw0, t.dw1, 0, 0)
		res = residueSubtractionInverse(residueT)
		if residueT != RESIDUE_EXACT {
			zCoeff = sub_256_256x256(zCoeff.dw0, zCoeff.dw1, zCoeff.dw2, zCoeff.dw3, 1, 0, 0, 0)
		}
	}
	return finalize38_256(zSign, qAlign, zCoeff, res)
}
