// Fused multiply-add in the 38-digit D38 domain: returns round38(x*y + c) with
// a SINGLE round-half-even rounding (no flags, no rounding mode). This is the
// Horner step of the Padé transcendental kernels, so fusion is required for
// bit-exact agreement with the sibling ports.
//
// All operands are finite non-zero in the kernels; the only degenerate inputs
// handled are a zero product (→ result is c) and a zero addend (→ rounded
// product). The wide-add core (addWide38) is the D128 FMA retargeted 34→38
// digits and finalized via finalize38_256; intermediates stay within U256
// (exact product ≤76 digits, every alignment shift bounded ≤77 digits).

package decimal128

func d38_fma(x, y, c d38) d38 {
	prod := mul_256_128x128(x.coeffLo64, x.coeffHi64, y.coeffLo64, y.coeffHi64)
	if isZero_bool_128(prod.dw0|prod.dw1, prod.dw2|prod.dw3) {
		return c // x*y == 0  →  result is exactly c
	}
	prodSign := d38_signFlag(x) != d38_signFlag(y)
	prodQ := d38_qExp(x) + d38_qExp(y)
	if d38_isZero(c) {
		return finalize38_256(prodSign, prodQ, prod, RESIDUE_EXACT)
	}
	return addWide38(prodSign, prodQ, prod, d38_signFlag(c), d38_qExp(c), u256FromU128(d38_u128(c)))
}

// addWide38 adds two wide signed values (sign, qExp, ≤77-digit U256 coefficient)
// and rounds the sum to a 38-digit D38 (round-half-even). Shared by the FMA
// product/addend path and the exp range reduction (x − n·ln10 with a 50-digit
// ln10).
func addWide38(prodSign bool, prodQ int, prodCoeff u256, cSign bool, cQ int, cCoeff u256) d38 {
	// m = side with the larger qExp; n = the other.
	var mSign, nSign bool
	var mQ, nQ int
	var mCoeff, nCoeff u256
	if prodQ >= cQ {
		mSign, mQ, mCoeff = prodSign, prodQ, prodCoeff
		nSign, nQ, nCoeff = cSign, cQ, cCoeff
	} else {
		mSign, mQ, mCoeff = cSign, cQ, cCoeff
		nSign, nQ, nCoeff = prodSign, prodQ, prodCoeff
	}
	if mSign == nSign {
		return addWide38LikeSign(mSign, mQ, mCoeff, nQ, nCoeff)
	}
	return addWide38UnlikeSign(mSign, mQ, mCoeff, nSign, nQ, nCoeff)
}

// addWide38LikeSign: |m| + |n| ≥ |m|, no cancellation. Keep m's digits, align n
// down (its truncated tail ≤ half a ULP at qAlign, captured as residue).
func addWide38LikeSign(sign bool, mQ int, mCoeff u256, nQ int, nCoeff u256) d38 {
	qDelta := mQ - nQ
	verify("", qDelta >= 0)
	mDigitLen := digitLen_int_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3)
	headroom := max(0, 38-mDigitLen)
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
	return finalize38_256(sign, qAlign, coeffSum, qr.residue)
}

// addWide38UnlikeSign: opposite signs ⇒ magnitude subtraction with possible
// cancellation. Shift m left as far as U256 capacity (77 digits) allows, which
// yields an EXACT subtraction (gap == 0) exactly when cancellation can matter;
// when operands are far apart in magnitude, n's tail collapses into the residue.
func addWide38UnlikeSign(mSign bool, mQ int, mCoeff u256, nSign bool, nQ int, nCoeff u256) d38 {
	qDelta := mQ - nQ
	verify("", qDelta >= 0)
	mDigitLen := digitLen_int_256(mCoeff.dw0, mCoeff.dw1, mCoeff.dw2, mCoeff.dw3)
	headroom := max(0, 77-mDigitLen)
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
	res := qr.residue

	ds := subAbs_d256swap_256x256(mAligned.dw0, mAligned.dw1, mAligned.dw2, mAligned.dw3, nAligned.dw0, nAligned.dw1, nAligned.dw2, nAligned.dw3)
	diff := u256{ds.diffDw0, ds.diffDw1, ds.diffDw2, ds.diffDw3}
	if diff == (u256{}) && res == RESIDUE_EXACT {
		return D38_ZERO
	}
	if ds.swap {
		// |n| > |m|: result takes n's sign; m was exact, no inversion.
		return finalize38_256(nSign, qAlign, diff, res)
	}
	// |m| ≥ |n|: result takes m's sign; a non-exact n-residue means diff is one
	// ULP too large by a sub-ULP amount — decrement and invert the residue.
	if res == RESIDUE_EXACT {
		return finalize38_256(mSign, qAlign, diff, RESIDUE_EXACT)
	}
	dec := sub_256_256x256(diff.dw0, diff.dw1, diff.dw2, diff.dw3, 1, 0, 0, 0)
	return finalize38_256(mSign, qAlign, dec, residueSubtractionInverse(res))
}
