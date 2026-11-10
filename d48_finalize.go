// Round a (sign, qExp, coefficient, residue) into a 48-digit D48 (round-half-
// even; Swift D48Finalize.swift): one entry over u320, where D38 has
// d38_finalize_128 and d38_finalize_256.

package decimal128

// 10^47 and 10^48 as literals (Swift D-4): the carry check and the carry
// result; every other power of ten is read at its width.
var (
	d48_POW10_47 = u320{6450984253743169536, -5431240233227854204, 293873587, 0, 0}
	d48_POW10_48 = u320{9169610316303040512, 1027829888850112811, 2938735877, 0, 0}
)

func d48RoundsHalfEven(res residue, coeff u320) bool {
	return (res == RESIDUE_GT_HALF) || ((res == RESIDUE_HALF) && ((coeff.dw0 & 1) != 0))
}

// d48_scaleDownTail rounds a < 10^48 coefficient by res; a carry to 10^48
// rescales to 10^47 at qExp + 1.
func d48_scaleDownTail(sign bool, qExp int, coeff u320, res residue) d48 {
	verify("", func() bool { return lt_320(coeff, d48_POW10_48) })
	if !d48RoundsHalfEven(res, coeff) {
		// No round-up: coeff < 10^48 stays in range (D48GoWorkOrder.md Phase 5,
		// option A: the carry check runs only after a round-up).
		if coeff.isZero() {
			return d48_ZERO
		}
		return d48_fromSignQExp_160(sign, qExp, u192{coeff.dw0, coeff.dw1, coeff.dw2})
	}
	candidate := add_320_320x64(coeff, 1)
	if lt_320(candidate, d48_POW10_48) {
		return d48_fromSignQExp_160(sign, qExp, u192{candidate.dw0, candidate.dw1, candidate.dw2})
	}
	// Round pushed us to == 10^48: scale down by 10 (exact).
	verify("", func() bool { return candidate == d48_POW10_48 })
	return d48_fromSignQExp_160(sign, qExp+1, d48_POW10_47.u192())
}

func d48_finalize_320(sign bool, qExp int, coeff u320, res residue) d48 {
	digitLen := digitLen_320(coeff)
	if digitLen <= 48 {
		return d48_scaleDownTail(sign, qExp, coeff, res)
	}
	qDelta := digitLen - 48
	// Fast paths for the common drops (Swift Phase 5): products of two 48-digit
	// values have 95-96 digits; divide quotients 49.
	var qr quot320Residue
	switch qDelta {
	case 48:
		qr = divPow10_q320res_320_recipmul_48(coeff)
	case 47:
		qr = divPow10_q320res_320_recipmul_47(coeff)
	case 1:
		qr = divPow10_q320res_320_recipmul_1(coeff)
	default:
		qr = divPow10_q320res(coeff, qDelta)
	}
	totalResidue := residueMerge(qr.residue, res)
	return d48_scaleDownTail(sign, qExp+qDelta, qr.quot(), totalResidue)
}
