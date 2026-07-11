// The finalize/rounding engine: turn an (sign, qExp, coefficient, residue) tuple
// into a canonical decimal128, applying precision rounding, the underflow/
// subnormal diversion, overflow, and clamping. finalize_128_ctxnull/finalize_256_ctxnull
// are the general entries; finalizeAddSubFnz/finalizeSwampedFnz are the FE
// fast-path entries with narrowed, provable contracts.

package decimal128

func finalize_128_ctxnull(sign bool, qExpIn int, coeffIn u128, residueIn residue, rounding round, ctxnull *context) d128 {
	// Fast path: EXACT residue and already in valid decimal128 range.
	if residueIn == RESIDUE_EXACT {
		if isCanonical34_128(qExpIn, coeffIn) {
			return d128Pack(sign, qExpIn, coeffIn.dw0, coeffIn.dw1)
		}
		if isZero_bool_128(coeffIn.dw0, coeffIn.dw1) {
			return d128_zero(sign, qExpIn)
		}
	}
	if isZero_bool_128(coeffIn.dw0, coeffIn.dw1) {
		return finalizeZero_ctxnull(sign, qExpIn, residueIn, rounding, ctxnull)
	}
	return finalize_128_nonTrivial_ctxnull(sign, qExpIn, coeffIn, residueIn, rounding, ctxnull)
}

// finalizeAddSubFnz_128_ctxnull — narrowed contract: qExp >= -6176 and coeff < 10^35.
// Skips the underflow diversion and full digit-length scan.
func finalizeAddSubFnz_128_ctxnull(sign bool, qExpIn int, coeffIn u128, residueIn residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return qExpIn >= -6176 })
	verify("", func() bool { return cmp_int_128x128_pow10(coeffIn.dw0, coeffIn.dw1, 35) < 0 })
	if isZero_bool_128(coeffIn.dw0, coeffIn.dw1) {
		return finalizeZero_ctxnull(sign, qExpIn, residueIn, rounding, ctxnull)
	}
	coeff0 := coeffIn.dw0
	coeff1 := coeffIn.dw1
	qExp := qExpIn
	res := residueIn
	if cmp_int_128x128_pow10(coeff0, coeff1, 34) >= 0 {
		qr := div_128_pow10_q128res(coeffIn, 1)
		coeff0 = qr.quotDw0
		coeff1 = qr.quotDw1
		res = residueMerge(qr.residue, res)
		qExp += 1
	}
	return roundAndPack_128_ctxnull(sign, qExp, coeff0, coeff1, res, rounding, ctxnull)
}

// finalizeSwampedFnz_128_ctxnull — narrower still: coeff != 0, coeff < 10^34,
// qExp in [-6176, 6111], residue != EXACT. Round, pack, signal inexact.
func finalizeSwampedFnz_128_ctxnull(sign bool, qExpIn int, coeffIn u128, residueIn residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return qExpIn >= -6176 && qExpIn <= 6111 })
	verify("", func() bool { return !isZero_bool_128(coeffIn.dw0, coeffIn.dw1) })
	verify("", func() bool { return cmp_int_128x128_pow10(coeffIn.dw0, coeffIn.dw1, 34) < 0 })
	verify("", func() bool { return residueIn != RESIDUE_EXACT })
	coeff0 := coeffIn.dw0
	coeff1 := coeffIn.dw1
	qExp := qExpIn
	isOdd := (coeff0 & 1) != 0
	if needsRoundUp(rounding, sign, isOdd, residueIn) {
		incremented := add_128_128x128(coeff0, coeff1, 1, 0)
		coeff0 = incremented.dw0
		coeff1 = incremented.dw1
		if cmp_int_128x128_pow10(coeff0, coeff1, 34) == 0 {
			coeff0 = pow10_128_dw0(33)
			coeff1 = pow10_128_dw1(33)
			qExp += 1
			if qExp > 6111 {
				return finalizeOverflow_ctxnull(sign, rounding, ctxnull)
			}
		}
	}
	z := decimalFNZ_64x64(sign, qExp, coeff0, coeff1)
	if ctxnull != nil {
		return ctxSignalInexact_ctxnull(ctxnull, z)
	}
	return z
}

func finalize_128_quiet(sign bool, qExpIn int, coeffIn u128, residueIn residue) d128 {
	if residueIn == RESIDUE_EXACT {
		if isCanonical34_128(qExpIn, coeffIn) {
			return d128Pack(sign, qExpIn, coeffIn.dw0, coeffIn.dw1)
		}
		if isZero_bool_128(coeffIn.dw0, coeffIn.dw1) {
			return d128_zero(sign, qExpIn)
		}
	}
	if isZero_bool_128(coeffIn.dw0, coeffIn.dw1) {
		return finalizeZero_quiet(sign, qExpIn, residueIn)
	}
	return finalize_128_nonTrivial_ctxnull(sign, qExpIn, coeffIn, residueIn, ROUND_TIES_TO_EVEN, nil)
}

func finalize_128_nonTrivial_ctxnull(sign bool, qExpIn int, coeffIn u128, residueIn residue, rounding round, ctxnull *context) d128 {
	rangeTruncationNeeded := -6176 - qExpIn
	precisionTruncationNeeded := max(digitLen_int_128(coeffIn.dw0, coeffIn.dw1)-34, 0)
	if rangeTruncationNeeded > precisionTruncationNeeded {
		return finalizeUnderflowRegion_128_ctxnull(sign, qExpIn, coeffIn, residueIn, rounding, ctxnull)
	}

	totalResidue := residueIn
	coeff0 := coeffIn.dw0
	coeff1 := coeffIn.dw1
	qExp := qExpIn
	if precisionTruncationNeeded > 0 {
		qr := div_128_pow10_q128res(coeffIn, precisionTruncationNeeded)
		coeff0 = qr.quotDw0
		coeff1 = qr.quotDw1
		totalResidue = residueMerge(qr.residue, totalResidue)
		qExp += precisionTruncationNeeded
		verify("", func() bool { return digitLen_int_128(coeff0, coeff1) == 34 })
	}
	verify("", func() bool { return digitLen_int_128(coeff0, coeff1) <= 34 })
	return roundAndPack_128_ctxnull(sign, qExp, coeff0, coeff1, totalResidue, rounding, ctxnull)
}

func finalizeZero_ctxnull(sign bool, qExp int, res residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return qExp <= 6111 })
	var z d128
	isOdd := false
	if res != RESIDUE_EXACT && needsRoundUp(rounding, sign, isOdd, res) {
		if qExp < -6176 {
			return finalizeUnderflowRegion_128_ctxnull(sign, qExp, u128{1, 0}, res, rounding, ctxnull)
		}
		z = decimalFNZ_64x64(sign, qExp, 1, 0)
	} else {
		z = d128_zero(sign, qExp)
	}
	if res != RESIDUE_EXACT && ctxnull != nil {
		return ctxSignalInexact_ctxnull(ctxnull, z)
	}
	return z
}

func finalizeOverflow_ctxnull(sign bool, rounding round, ctxnull *context) d128 {
	var z d128
	if roundingOverflowsToInfinity(rounding, sign) {
		z = d128_infinity(sign)
	} else {
		z = maxFiniteMagnitude(sign)
	}
	if ctxnull != nil {
		return ctxSignalInexactOverflow_ctxnull(ctxnull, z)
	}
	return z
}

func finalizeUnderflowRegion_128_ctxnull(sign bool, qExp int, coeff u128, res residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return qExp < -6176 })
	truncationNeeded := -6176 - qExp
	digitLen := digitLen_int_128(coeff.dw0, coeff.dw1)
	if truncationNeeded > digitLen {
		return finalizeUnderflowFullySwamped(sign, rounding, ctxnull)
	}
	if truncationNeeded == digitLen {
		scaleResidue := residueFromDecadeWithDigitLen_128(coeff.dw0, coeff.dw1, digitLen)
		totalResidue := residueMerge(scaleResidue, res)
		return finalizeUnderflowExactBoundary_ctxnull(sign, totalResidue, rounding, ctxnull)
	}
	verify("", func() bool { return truncationNeeded > 0 && truncationNeeded < digitLen })
	qr := div_128_pow10_q128res(coeff, truncationNeeded)
	totalResidue := residueMerge(qr.residue, res)
	return finalizeSubnormal_ctxnull(sign, qr.quotDw0, qr.quotDw1, totalResidue, rounding, ctxnull)
}

func finalizeUnderflowRegion_256_ctxnull(sign bool, qExp int, u256In u256, res residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return qExp < -6176 })
	truncationNeeded := -6176 - qExp
	digitLen := digitLen_int_256(u256In.dw0, u256In.dw1, u256In.dw2, u256In.dw3)
	if truncationNeeded > digitLen {
		return finalizeUnderflowFullySwamped(sign, rounding, ctxnull)
	}
	if truncationNeeded == digitLen {
		scaleResidue := residueFromDecadeWithDigitLen_256(u256In, digitLen)
		totalResidue := residueMerge(scaleResidue, res)
		return finalizeUnderflowExactBoundary_ctxnull(sign, totalResidue, rounding, ctxnull)
	}
	verify("", func() bool { return truncationNeeded > 0 && truncationNeeded < digitLen })
	qr := div_256_pow10_q128res(u256In, truncationNeeded)
	totalResidue := residueMerge(qr.residue, res)
	return finalizeSubnormal_ctxnull(sign, qr.quotDw0, qr.quotDw1, totalResidue, rounding, ctxnull)
}

func finalizeUnderflowFullySwamped(sign bool, rounding round, ctxnull *context) d128 {
	var z d128
	if roundingUnderflowsToZero(rounding, sign) {
		z = d128_zero(sign, -6176)
	} else {
		z = minFiniteMagnitude(sign)
	}
	if ctxnull != nil {
		return ctxSignalInexactUnderflow_ctxnull(ctxnull, z)
	}
	return z
}

func finalizeUnderflowExactBoundary_ctxnull(sign bool, res residue, rounding round, ctxnull *context) d128 {
	var z d128
	isOdd := false
	if needsRoundUp(rounding, sign, isOdd, res) {
		z = minFiniteMagnitude(sign)
	} else {
		z = d128_zero(sign, -6176)
	}
	if ctxnull != nil {
		return ctxSignalInexactUnderflow_ctxnull(ctxnull, z)
	}
	return z
}

func finalizeSubnormal_ctxnull(sign bool, coeffDw0, coeffDw1 int64, res residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return digitLen_int_128(coeffDw0, coeffDw1) <= 34 })
	coeffT0 := coeffDw0
	coeffT1 := coeffDw1
	qExpT := -6176
	isOdd := (coeffT0 & 1) != 0
	if res != RESIDUE_EXACT && needsRoundUp(rounding, sign, isOdd, res) {
		rounded := add_128_128x128(coeffT0, coeffT1, 1, 0)
		coeffT0 = rounded.dw0
		coeffT1 = rounded.dw1
		verify("", func() bool { return cmp_int_128x128_pow10(coeffT0, coeffT1, 34) <= 0 })
		if cmp_int_128x128_pow10(coeffT0, coeffT1, 34) == 0 {
			coeffT0 = pow10_128_dw0(33)
			coeffT1 = pow10_128_dw1(33)
			qExpT += 1
		}
	}
	z := decimalFNZ_64x64(sign, qExpT, coeffT0, coeffT1)
	if res != RESIDUE_EXACT && ctxnull != nil {
		return ctxSignalInexactUnderflow_ctxnull(ctxnull, z)
	}
	return z
}

func roundAndPack_128_ctxnull(sign bool, qExpIn int, coeffDw0, coeffDw1 int64, totalResidue residue, rounding round, ctxnull *context) d128 {
	qExp := qExpIn
	coeff0 := coeffDw0
	coeff1 := coeffDw1
	verify("", func() bool { return digitLen_int_128(coeff0, coeff1) <= 34 })
	applyRounding := totalResidue != RESIDUE_EXACT
	isOdd := (coeff0 & 1) != 0
	if applyRounding && needsRoundUp(rounding, sign, isOdd, totalResidue) {
		incremented := add_128_128x128(coeff0, coeff1, 1, 0)
		coeff0 = incremented.dw0
		coeff1 = incremented.dw1
		if cmp_int_128x128_pow10(coeff0, coeff1, 34) == 0 {
			coeff0 = pow10_128_dw0(33)
			coeff1 = pow10_128_dw1(33)
			qExp += 1
		}
	}
	verify("", func() bool { return qExp >= -6176 })
	verify("", func() bool { return digitLen_int_128(coeff0, coeff1) <= 34 })
	if qExp > 6111 {
		qExcess := qExp - 6111
		if digitLen_int_128(coeff0, coeff1)+qExcess <= 34 {
			return finalizeClamping_128(sign, coeff0, coeff1, qExp)
		}
		return finalizeOverflow_ctxnull(sign, rounding, ctxnull)
	}
	z := decimalFNZ_64x64(sign, qExp, coeff0, coeff1)
	if applyRounding && ctxnull != nil {
		return ctxSignalInexact_ctxnull(ctxnull, z)
	}
	return z
}

func isCanonical34_128(qExp int, coeff u128) bool {
	return isLt_pow10_34_bool_128(coeff.dw0, coeff.dw1) && qExp >= -6176 && qExp <= 6111
}

func finalizeClamping_128(sign bool, coeffDw0, coeffDw1 int64, qExp int) d128 {
	qExcess := qExp - 6111
	verify("", func() bool { return qExcess > 0 && qExcess <= 34-digitLen_int_128(coeffDw0, coeffDw1) })
	coeffScaled := mulPow10_128_128(coeffDw0, coeffDw1, qExcess)
	verify("", func() bool { return digitLen_int_128(coeffScaled.dw0, coeffScaled.dw1) <= 34 })
	return decimalFNZ_64x64(sign, 6111, coeffScaled.dw0, coeffScaled.dw1)
}

func finalize_256_ctxnull(sign bool, qExpIn int, u256In u256, residueIn residue, rounding round, ctxnull *context) d128 {
	if isZero_bool_128(u256In.dw2, u256In.dw3) {
		return finalize_128_ctxnull(sign, qExpIn, u256In.lo(), residueIn, rounding, ctxnull)
	}
	return finalizeFullWidth_256_ctxnull(sign, qExpIn, u256In, residueIn, rounding, ctxnull)
}

func finalizeFullWidth_256_ctxnull(sign bool, qExpIn int, u256In u256, residueIn residue, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return !isZero_bool_128(u256In.dw2, u256In.dw3) })
	rangeTruncationNeeded := -6176 - qExpIn
	precisionTruncationNeeded := max(digitLen_int_256(u256In.dw0, u256In.dw1, u256In.dw2, u256In.dw3)-34, 0)
	if rangeTruncationNeeded > precisionTruncationNeeded {
		return finalizeUnderflowRegion_256_ctxnull(sign, qExpIn, u256In, residueIn, rounding, ctxnull)
	}
	verify("", func() bool { return precisionTruncationNeeded >= 5 })
	qr := div_256_pow10_q128res(u256In, precisionTruncationNeeded)
	totalResidue := residueMerge(qr.residue, residueIn)
	qExp := qExpIn + precisionTruncationNeeded
	verify("", func() bool { return digitLen_int_128(qr.quotDw0, qr.quotDw1) == 34 })
	return roundAndPack_128_ctxnull(sign, qExp, qr.quotDw0, qr.quotDw1, totalResidue, rounding, ctxnull)
}

func finalizeZero_quiet(sign bool, qExp int, res residue) d128 {
	verify("", func() bool { return qExp <= 6111 })
	if res != RESIDUE_GT_HALF {
		return d128_zero(sign, qExp)
	}
	if qExp >= -6176 {
		return d128Pack(sign, qExp, 1, 0)
	}
	return finalizeUnderflowRegion_128_ctxnull(sign, qExp, u128{1, 0}, res, ROUND_TIES_TO_EVEN, nil)
}

// --- magnitude limits ---

func maxFiniteMagnitude(sign bool) d128 {
	c := sub_128_128x128(pow10_128_dw0(34), pow10_128_dw1(34), 1, 0)
	return decimalFNZ_128(sign, 6111, c.dw0, c.dw1)
}

func minFiniteMagnitude(sign bool) d128 {
	return decimalFNZ_64x64(sign, -6176, 1, 0)
}
