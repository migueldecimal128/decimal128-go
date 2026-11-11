// D48 add/subtract (round-half-even, no flags; Swift D48AddSub.swift). Mirrors
// d38 add/sub: aligns operands by exponent, folding the smaller operand's
// shifted-out digits into a residue, then routes through d48_finalize_320.
// Intermediates are u320.

package decimal128

func d48_add(x, y d48) d48 {
	// A zero operand must short-circuit: the alignment paths would shift the
	// non-zero operand down and shred its precision.
	if d48_isZero(y) {
		return x
	}
	if d48_isZero(x) {
		return y
	}
	if x.qExpAndSignBit == y.qExpAndSignBit {
		return d48_addAlignedLikeSign(x, y)
	}
	if d48_qExp(x) == d48_qExp(y) {
		return d48_addAlignedUnlikeSign(x, y)
	}
	if d48_signFlag(x) == d48_signFlag(y) {
		return d48_addUnalignedLikeSign(x, y)
	}
	return d48_addUnalignedUnlikeSign(x, y)
}

func d48_subtract(x, y d48) d48 {
	return d48_add(x, d48_negated(y))
}

func d48_addAlignedLikeSign(x, y d48) d48 {
	sum := add_320_320x320(d48_u320(x), d48_u320(y))
	if cmp_int_320_pow10(sum, 48) < 0 {
		return d48_fromQExpAndSignBit_160(x.qExpAndSignBit, sum.u192())
	}
	return d48_finalize_320(d48_signFlag(x), d48_qExp(x), sum, const_RESIDUE_EXACT)
}

func d48_addAlignedUnlikeSign(x, y d48) d48 {
	verify("", func() bool { return d48_qExp(x) == d48_qExp(y) })
	verify("", func() bool { return d48_signFlag(x) != d48_signFlag(y) })
	xc := d48_u320(x)
	yc := d48_u320(y)
	c := cmp_int_320x320(xc, yc)
	if c == 0 {
		return const_D48_ZERO
	}
	if c < 0 { // |x| < |y|: flip the sign
		return d48_fromQExpAndSignBit_160(x.qExpAndSignBit^1, sub_320_320x320(yc, xc).u192())
	}
	return d48_fromQExpAndSignBit_160(x.qExpAndSignBit, sub_320_320x320(xc, yc).u192())
}

//go:noinline
func d48_addUnalignedLikeSign(x, y d48) d48 {
	verify("", func() bool { return (d48_digitLen(x) <= 48) && (d48_digitLen(y) <= 48) })
	verify("", func() bool { return d48_qExp(x) != d48_qExp(y) })
	verify("", func() bool { return d48_signFlag(x) == d48_signFlag(y) })

	m, n := y, x
	if d48_qExp(x) > d48_qExp(y) {
		m, n = x, y
	}
	mQ := d48_qExp(m)
	nQ := d48_qExp(n)
	mDigitLen := d48_digitLen(m)
	nDigitLen := d48_digitLen(n)
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta > 0 })
	headroom := 48 - mDigitLen
	shiftLeft := min(headroom, qDelta)
	z := mulPow10_320(d48_u320(m), shiftLeft)
	qAlign := mQ - shiftLeft
	shiftRight := qAlign - nQ
	var res residue
	if shiftRight == 0 {
		verify("", func() bool { return shiftLeft > 0 })
		z = add_320_320x320(z, d48_u320(n))
		res = const_RESIDUE_EXACT
	} else if shiftRight > nDigitLen {
		if d48_isZero(n) {
			res = const_RESIDUE_EXACT
		} else {
			res = const_RESIDUE_LT_HALF
		}
	} else if shiftRight == nDigitLen {
		// n drops entirely: its residue against 10^nDigitLen (<= 10^48).
		res = residueFromRemainderPow10_320(d48_u320(n), nDigitLen)
	} else {
		qr := divPow10_q320res(d48_u320(n), shiftRight)
		res = qr.residue
		z = add_320_320x320(z, qr.quot())
	}
	return d48_finalize_320(d48_signFlag(m), qAlign, z, res)
}

func d48_addUnalignedUnlikeSign(x, y d48) d48 {
	verify("", func() bool { return d48_qExp(x) != d48_qExp(y) })
	verify("", func() bool { return d48_signFlag(x) != d48_signFlag(y) })
	cmp := d48_cmpMag(x, y)
	if cmp > 0 {
		return d48_subMagUnalignedFnzFnz(d48_signFlag(x), x, y)
	} else if cmp < 0 {
		return d48_subMagUnalignedFnzFnz(d48_signFlag(y), y, x)
	}
	return const_D48_ZERO
}

func d48_subMagUnalignedFnzFnz(zSign bool, m, s d48) d48 {
	verify("", func() bool { return d48_qExp(m) != d48_qExp(s) })
	verify("", func() bool { return d48_cmpMag(m, s) > 0 })
	mQ := d48_qExp(m)
	mc := d48_u320(m)
	mDigitLen := digitLen_320(mc)
	sQ := d48_qExp(s)
	sc := d48_u320(s)
	sDigitLen := digitLen_320(sc)
	gap := mQ - sQ

	if gap < 0 {
		// |m| > |s| but qExp(m) < qExp(s): scale s up to m's qExp (exact, fits).
		diff := sub_320_320x320(mc, mulPow10_320(sc, -gap))
		verify("", func() bool { return cmp_int_320_pow10(diff, 48) < 0 })
		return d48_fromSignQExp_160(zSign, mQ, diff.u192())
	}

	verify("", func() bool { return mDigitLen <= 48 })
	verify("", func() bool { return sDigitLen <= 48 })
	headroomWithGuard := 1 + 48 - mDigitLen
	shiftMLeft := min(gap, headroomWithGuard)
	z := mulPow10_320(mc, shiftMLeft)

	qAlign := mQ - shiftMLeft
	shiftSRight := qAlign - sQ
	verify("", func() bool { return shiftSRight >= 0 })

	var res residue
	if shiftSRight == 0 {
		z = sub_320_320x320(z, sc)
		res = const_RESIDUE_EXACT
	} else if shiftSRight >= sDigitLen {
		if shiftSRight > sDigitLen {
			res = const_RESIDUE_GT_HALF
		} else {
			// s drops entirely: its residue against 10^sDigitLen (<= 10^48).
			res = residueSubtractionInverse(residueFromRemainderPow10_320(sc, sDigitLen))
		}
		verify("", func() bool { return res != const_RESIDUE_EXACT })
		z = sub_320_320x320(z, u320FromDw0(1))
	} else {
		qr := divPow10_q320res(sc, shiftSRight)
		z = sub_320_320x320(z, qr.quot())
		res = residueSubtractionInverse(qr.residue)
		if qr.residue != const_RESIDUE_EXACT {
			z = sub_320_320x320(z, u320FromDw0(1))
		}
	}
	return d48_finalize_320(zSign, qAlign, z, res)
}
