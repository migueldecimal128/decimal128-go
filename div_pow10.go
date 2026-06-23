// Core composition over the primitive divPow10 kernels: the strategy dispatch
// (D16) that picks magic/recipmul/barrett by exponent and width, and the chunked
// Barrett loops for large exponents. (The rrmp10 path is not wired here — Swift
// leaves it a FIXME; the magic+recipmul+barrett dispatch covers the engine.)

package decimal128

func div_128_pow10_checked_q128res(x u128, pow10 int) quot128Residue {
	verify("", pow10 >= 0 && pow10 <= 13000)
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
	verify("", pow10 >= 1 && pow10 < digitLen_int_128(x.dw0, x.dw1))
	// pow10 == 1 keeps the divide-by-10 magic (single-digit residue lookup); a
	// single reciprocal-multiply kernel covers 2..38 uniformly.
	if pow10 == 1 {
		return div1e1_q128res_128_magic(x.dw0, x.dw1)
	}
	return divPow10_q128res_128_recipmul(x.dw0, x.dw1, pow10)
}

func div_256_pow10_checked_q256res(x u256, pow10 int) quot256Residue {
	verify("", pow10 >= 0 && pow10 <= 19000)
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
		verify("", isZero_bool_128(quotient.dw2, quotient.dw3))
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
	if pow10 <= barrettPow10Max {
		return divPow10_q128res_256_barrett(x.dw0, x.dw1, x.dw2, x.dw3, pow10)
	}
	return div_256_pow10_q128res_barrett_loop(x, pow10)
}

// div_256_pow10_q128res_barrett_loop chains barrettPow10Max-sized steps for pow10
// up to 77, merging per-step residues (strategy D16).
func div_256_pow10_q128res_barrett_loop(x u256, pow10 int) quot128Residue {
	verify("", pow10 >= 0 && pow10 <= 77)
	pow10Remaining := pow10
	res := RESIDUE_EXACT
	t256 := x
	for !isZero_bool_128(t256.dw2, t256.dw3) && pow10Remaining > 0 {
		pow10Step := min(pow10Remaining, barrettPow10Max)
		qr := divPow10_q256res_256_barrett(t256.dw0, t256.dw1, t256.dw2, t256.dw3, pow10Step)
		t256 = u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
		res = residueMerge(qr.residue, res)
		pow10Remaining -= barrettPow10Max
	}
	demand("", isZero_bool_128(t256.dw2, t256.dw3))
	t128 := t256.lo()
	for pow10Remaining > 0 {
		pow10Step := min(pow10Remaining, barrettPow10Max)
		qr := divPow10_q128res_128_barrett(t128.dw0, t128.dw1, pow10Step)
		t128 = fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		res = residueMerge(qr.residue, res)
		pow10Remaining -= barrettPow10Max
	}
	return quot128Residue{t128.dw0, t128.dw1, res}
}
