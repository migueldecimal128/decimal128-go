// D38 finalization: round a 128- or 256-bit intermediate coefficient down to the
// 38-digit D38 form, merging residues. Used by the D38 extended-precision engine.

package decimal128

func finalize38_128(sign bool, qExp int, u128v u128, res residue) d38 {
	if cmp_int_128x128_pow10(u128v.dw0, u128v.dw1, 38) < 0 {
		var roundUp int64 = 0
		if res == RESIDUE_GT_HALF || (res == RESIDUE_HALF && (u128v.dw0&1) != 0) {
			roundUp = 1
		}
		rounded := add_128_128x128(u128v.dw0, u128v.dw1, roundUp, 0)
		if isZero_bool_128(rounded.dw0, rounded.dw1) {
			return D38_ZERO
		}
		if cmp_int_128x128_pow10(rounded.dw0, rounded.dw1, 38) < 0 {
			return d38Pack(sign, qExp, rounded.dw0, rounded.dw1)
		}
		verify("", rounded == POW10_38)
		return d38Pack(sign, qExp+1, pow10_128_dw0(37), pow10_128_dw1(37))
	}
	// u128 >= 10^38: scale down by 10, merging residues.
	qr := div1e1_q128res_128_magic(u128v.dw0, u128v.dw1)
	totalResidue := residueMerge(qr.residue, res)
	return scaleDownTailD38(sign, qExp+1, qr.quotDw0, qr.quotDw1, totalResidue)
}

func finalize38_256(sign bool, qExp int, u256v u256, res residue) d38 {
	if isZero_bool_128(u256v.dw2, u256v.dw3) {
		return finalize38_128(sign, qExp, u256v.lo(), res)
	}
	digitLen := digitLen_int_256(u256v.dw0, u256v.dw1, u256v.dw2, u256v.dw3)
	qDelta := digitLen - 38
	verify("", qDelta > 0)
	qr := div_256_pow10_q128res(u256v, qDelta)
	totalResidue := residueMerge(qr.residue, res)
	return scaleDownTailD38(sign, qExp+qDelta, qr.quotDw0, qr.quotDw1, totalResidue)
}

func scaleDownTailD38(sign bool, qExp int, u128Dw0, u128Dw1 int64, res residue) d38 {
	var roundUp int64 = 0
	if res == RESIDUE_GT_HALF || (res == RESIDUE_HALF && (u128Dw0&1) != 0) {
		roundUp = 1
	}
	candidate := add_128_128x128(u128Dw0, u128Dw1, roundUp, 0)
	if cmp_int_128x128_pow10(candidate.dw0, candidate.dw1, 38) < 0 {
		if isZero_bool_128(candidate.dw0, candidate.dw1) {
			return D38_ZERO
		}
		return d38Pack(sign, qExp, candidate.dw0, candidate.dw1)
	}
	verify("", candidate == POW10_38)
	return d38Pack(sign, qExp+1, pow10_128_dw0(37), pow10_128_dw1(37))
}
