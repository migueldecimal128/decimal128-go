// Core composition over the primitive divPow10 kernels: the strategy dispatch
// (D16) that picks magic/recipmul by exponent and width. The q256res FMA-
// alignment path (only AddWide) stays on the chunked Barrett loop; the q128res
// 128- and 256-bit-dividend paths are single reciprocal-multiply kernels (D45/D47).

package decimal128

func div_128_pow10_checked_q128res(x u128, pow10 int) quot128Residue {
	verify("", func() bool { return pow10 >= 0 && pow10 <= 13000 })
	if pow10 == 0 {
		return quot128Residue{x.dw0, x.dw1, RESIDUE_EXACT}
	}
	digitLen := digitLen_int_128(x.dw0, x.dw1)
	if pow10 < digitLen {
		return div_128_pow10_q128res(x, pow10)
	}
	var res residue
	if pow10 > digitLen {
		if digitLen > 0 {
			res = RESIDUE_LT_HALF
		} else {
			res = RESIDUE_EXACT
		}
	} else {
		res = residueFromDecadeWithDigitLen_128(x.dw0, x.dw1, digitLen)
	}
	return quot128Residue{0, 0, res}
}

func div_128_pow10_q128res(x u128, pow10 int) quot128Residue {
	verify("", func() bool { return pow10 >= 1 && pow10 < digitLen_int_128(x.dw0, x.dw1) })
	// pow10 == 1 keeps the divide-by-10 magic (single-digit residue lookup); a
	// single reciprocal-multiply kernel covers 2..38 uniformly.
	if pow10 == 1 {
		return div1e1_q128res_128_magic(x.dw0, x.dw1)
	}
	return divPow10_q128res_128_recipmul(x.dw0, x.dw1, pow10)
}

func div_256_pow10_checked_q256res(x u256, pow10 int) quot256Residue {
	verify("", func() bool { return pow10 >= 0 && pow10 <= 19000 })
	if pow10 == 0 {
		return quot256Residue{x.dw0, x.dw1, x.dw2, x.dw3, RESIDUE_EXACT}
	}
	digitLen := digitLen_int_256(x.dw0, x.dw1, x.dw2, x.dw3)
	if pow10 < digitLen {
		return div_256_pow10_q256res(x, pow10)
	}
	var res residue
	if pow10 > digitLen {
		if digitLen > 0 {
			res = RESIDUE_LT_HALF
		} else {
			res = RESIDUE_EXACT
		}
	} else {
		res = residueFromDecadeWithDigitLen_256(x, digitLen)
	}
	return quot256Residue{0, 0, 0, 0, res}
}

func div_256_pow10_q256res(x u256, pow10 int) quot256Residue {
	pow10Remaining := pow10
	totalResidue := RESIDUE_EXACT
	quotient := x
	for pow10Remaining > 0 && !isZero_bool_128(quotient.dw2, quotient.dw3) {
		chunk := min(pow10Remaining, barrettPow10Max)
		qr := divPow10_q256res_256_barrett(quotient.dw0, quotient.dw1, quotient.dw2, quotient.dw3, chunk)
		totalResidue = residueMerge(qr.residue, totalResidue)
		quotient = u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
		pow10Remaining -= chunk
	}
	for pow10Remaining > 0 {
		verify("", func() bool { return isZero_bool_128(quotient.dw2, quotient.dw3) })
		chunk := min(pow10Remaining, barrettPow10Max)
		qr := divPow10_q128res_128_barrett(quotient.dw0, quotient.dw1, chunk)
		totalResidue = residueMerge(qr.residue, totalResidue)
		quotient = u256{qr.quotDw0, qr.quotDw1, 0, 0}
		pow10Remaining -= chunk
	}
	return quot256Residue{quotient.dw0, quotient.dw1, quotient.dw2, quotient.dw3, totalResidue}
}

func div_256_pow10_q128res(x u256, pow10 int) quot128Residue {
	if pow10 == 0 {
		if !isZero_bool_128(x.dw2, x.dw3) {
			impossible("div_256_pow10_q128res: pow10 == 0 quotient overflows U128")
		}
		return quot128Residue{x.dw0, x.dw1, RESIDUE_EXACT}
	}
	if isZero_bool_128(x.dw2, x.dw3) {
		return div_128_pow10_q128res(x.lo(), pow10)
	}
	// A single reciprocal-multiply kernel (D47) covers the whole 256-bit-dividend
	// q128res range pow10 1..77, replacing the former barrett(<=13)/barrett-loop(>13)
	// split: it is flat in k where the barrett-loop grew with k, so it is a win
	// across the range (and retires the rrmp10-256 machinery). Mirrors the 128-bit
	// recipmul collapse (D45).
	return divPow10_q128res_256_recipmul(x, pow10)
}
