// Comparisons against 10^p for the D48 domain (D48GoWorkOrder.md; Swift
// U320Pow10.swift): the digit count, and the residue of a remainder against a
// power of ten or a general divisor. Every power of ten is read at its own
// width from regions A/B (10^0..10^77, pow10) or region C (10^78..10^96,
// pow10_320); none is built as a five-limb value.

package decimal128

// cmp_int_320_pow10 is -1, 0 or 1 as x <, ==, > 10^p, 0 <= p <= 96. Compares
// only the limbs 10^p occupies; any nonzero limb of x above them makes x larger.
func cmp_int_320_pow10(x u320, p int) int {
	verify("", func() bool { return (p >= 0) && (p <= 96) })
	if p <= 38 {
		if (x.dw2 | x.dw3 | x.dw4) != 0 {
			return 1
		}
		return cmp_int_128x128(x.dw0, x.dw1, pow10_128_dw0(p), pow10_128_dw1(p))
	}
	if p <= 77 {
		if x.dw4 != 0 {
			return 1
		}
		i := p - 39
		return cmp_int_320x320(x, u320{pow10_256_dw0(i), pow10_256_dw1(i), pow10_256_dw2(i), pow10_256_dw3(i), 0})
	}
	i := p - 78
	return cmp_int_320x320(x, u320{pow10_320_dw0(i), pow10_320_dw1(i), pow10_320_dw2(i), pow10_320_dw3(i), pow10_320_dw4(i)})
}

// digitLen_320 is the decimal digit count; 0 for x == 0 (as
// digitLen_int_128). The bit-length estimate picks 10^lo, and the comparison
// runs at that power's width.
func digitLen_320(x u320) int {
	nbits := bitLen_int_320(x)
	if nbits == 0 {
		return 0
	}
	lo := (nbits * 1233) >> 12 // <= 96 for nbits <= 320
	if cmp_int_320_pow10(x, lo) < 0 {
		return lo
	}
	return lo + 1
}

// residueFromRemainderPow10_320 is the residue of rem against 10^p, 0 <= rem <
// 10^p, p <= 48: for the add/sub paths where the smaller operand drops
// entirely. rem < 10^48 < 2^160, so 2·rem and 10^p both fit three limbs.
func residueFromRemainderPow10_320(rem u320, p int) residue {
	verify("", func() bool { return (p >= 0) && (p <= 48) })
	verify("rem >= 10^p", func() bool { return cmp_int_320_pow10(rem, p) < 0 })
	if rem.isZero() {
		return RESIDUE_EXACT
	}
	c := cmp_int_320_pow10(add_320_320x320(rem, rem), p)
	if c < 0 {
		return RESIDUE_LT_HALF
	}
	if c == 0 {
		return RESIDUE_HALF
	}
	return RESIDUE_GT_HALF
}

// residueFromRemainderDivisor_320 is the residue of rem against a general
// divisor, 0 <= rem < divisor <= 10^96 (so 2·rem < 2^320): the D48 divide's
// remainder.
func residueFromRemainderDivisor_320(rem, divisor u320) residue {
	verify("rem >= divisor", func() bool { return lt_320(rem, divisor) })
	if rem.isZero() {
		return RESIDUE_EXACT
	}
	c := cmp_int_320x320(add_320_320x320(rem, rem), divisor)
	if c < 0 {
		return RESIDUE_LT_HALF
	}
	if c == 0 {
		return RESIDUE_HALF
	}
	return RESIDUE_GT_HALF
}
