// D48 fused multiply-add (round-half-even): x·y + c in one rounding step (Swift
// D48Fma.swift). The product is an exact u320 (<= 96 digits); d48_addWide
// aligns it against c and rounds once. Sizing (D48ZigPrototypeWorkOrder.md §2):
// the like-sign sum stays < 10^96, the unlike-sign shift fills u320 to 96
// digits. No FMA exact-alignment shortcut (G-6).

package decimal128

func d48_addWideLikeSign(sign bool, mQ int, mCoeff u320, nQ int, nCoeff u320) d48 {
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta >= 0 })
	mDigitLen := digitLen_320(mCoeff)
	headroom := max(0, 48-mDigitLen)
	shiftLeft := min(qDelta, headroom)

	mAligned := mCoeff
	if shiftLeft != 0 {
		mAligned = mulPow10_320(mCoeff, shiftLeft)
	}
	qAlign := mQ - shiftLeft
	gap := qAlign - nQ
	verify("", func() bool { return gap >= 0 })

	qr := divPow10_checked_q320res(nCoeff, gap)
	coeffSum := add_320_320x320(mAligned, qr.quot()) // < 10^96 (sizing §2)
	return d48_finalize_320(sign, qAlign, coeffSum, qr.residue)
}

func d48_addWideUnlikeSign(mSign bool, mQ int, mCoeff u320, nSign bool, nQ int, nCoeff u320) d48 {
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta >= 0 })
	mDigitLen := digitLen_320(mCoeff)
	// Shift m left as far as u320 capacity (96 digits) allows: exact when
	// cancellation can matter.
	headroom := max(0, 96-mDigitLen)
	shiftLeft := min(qDelta, headroom)

	mAligned := mCoeff
	if shiftLeft != 0 {
		mAligned = mulPow10_320(mCoeff, shiftLeft)
	}
	qAlign := mQ - shiftLeft
	gap := qAlign - nQ
	verify("", func() bool { return gap >= 0 })

	qr := divPow10_checked_q320res(nCoeff, gap)
	nAligned := qr.quot()
	res := qr.residue

	c := cmp_int_320x320(mAligned, nAligned)
	if (c == 0) && (res == RESIDUE_EXACT) {
		return d48_ZERO
	}
	if c < 0 {
		// |n| > |m|: the result takes n's sign; m was exact, no inversion.
		return d48_finalize_320(nSign, qAlign, sub_320_320x320(nAligned, mAligned), res)
	}
	diff := sub_320_320x320(mAligned, nAligned)
	// |m| >= |n|: the result takes m's sign; a non-exact n-residue means diff is
	// one ULP too large by a sub-ULP amount -- decrement and invert the residue.
	if res == RESIDUE_EXACT {
		return d48_finalize_320(mSign, qAlign, diff, RESIDUE_EXACT)
	}
	return d48_finalize_320(mSign, qAlign, sub_320_320x320(diff, u320FromDw0(1)), residueSubtractionInverse(res))
}

// d48_addWide is prod + c for an exact wide product (sign, qExp, u320
// coefficient) and a value c, rounded once. Also exp's range reduction
// x − n·ln10.
func d48_addWide(prodSign bool, prodQ int, prodCoeff u320, cSign bool, cQ int, cCoeff u320) d48 {
	// m = the side with the larger qExp; n = the other.
	if prodQ >= cQ {
		if prodSign == cSign {
			return d48_addWideLikeSign(prodSign, prodQ, prodCoeff, cQ, cCoeff)
		}
		return d48_addWideUnlikeSign(prodSign, prodQ, prodCoeff, cSign, cQ, cCoeff)
	}
	if prodSign == cSign {
		return d48_addWideLikeSign(cSign, cQ, cCoeff, prodQ, prodCoeff)
	}
	return d48_addWideUnlikeSign(cSign, cQ, cCoeff, prodSign, prodQ, prodCoeff)
}

func d48_fma(x, y, c d48) d48 {
	prod := mul_320_192x192(d48_u160(x), d48_u160(y))
	if prod.isZero() {
		return c // x·y == 0: the result is exactly c
	}
	prodSign := d48_signFlag(x) != d48_signFlag(y)
	prodQ := d48_qExp(x) + d48_qExp(y)
	if d48_isZero(c) {
		return d48_finalize_320(prodSign, prodQ, prod, RESIDUE_EXACT)
	}
	return d48_addWide(prodSign, prodQ, prod, d48_signFlag(c), d48_qExp(c), d48_u320(c))
}
