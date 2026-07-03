// Core-layer residue helpers (D17): merge/inverse and the decade/divisor
// classifications the finalize and arithmetic paths consume. Direction semantics
// (what a residue means under a rounding mode) live in round.go; the residue
// type and the primitive-layer classifications live in residue.go.

package decimal128

func residueMerge(left, right residue) residue {
	verify("", func() bool { return (left|right)&^3 == 0 })
	return (left | right>>1 | right&1) & 3
}

func residueSubtractionInverse(res residue) residue {
	verify("", func() bool { return res&^3 == 0 })
	// EXACT->EXACT, LT_HALF->GT_HALF, HALF->HALF, GT_HALF->LT_HALF, packed nybbles.
	const inverseMap = 0x1230
	return (inverseMap >> uint(res<<2)) & 0x3
}

func residueFromRemainderDivisor_256(remainder, divisor u256) residue {
	if (remainder.dw3 >> 63) == 0 {
		if isZero_bool_128(remainder.dw2|remainder.dw0, remainder.dw3|remainder.dw1) {
			return RESIDUE_EXACT
		}
		doubled := shl1_256_256(remainder.dw0, remainder.dw1, remainder.dw2, remainder.dw3)
		ge := 0
		if !doubled.less(divisor) {
			ge = 1
		}
		eq := 0
		if doubled == divisor {
			eq = 1
		}
		return (ge << 1) | (1 ^ eq)
	}
	return RESIDUE_GT_HALF
}

func residueFromDecadeWithDigitLen_128(dw0, dw1 int64, digitLen int) residue {
	// Not mergeable with residueFromRemainderDivisor_128: digitLen could be 39
	// but 10^39 does not fit in 128 bits.
	if digitLen >= 0 && digitLen <= 38 {
		halfPow10 := shr_128_128(pow10_128_dw0(digitLen), pow10_128_dw1(digitLen), 1)
		c := cmp_int_128x128(dw0, dw1, halfPow10.dw0, halfPow10.dw1)
		ge := 0
		if c >= 0 {
			ge = 1
		}
		eq := 0
		if c == 0 {
			eq = 1
		}
		return (ge << 1) | (1 ^ eq)
	}
	verify("", func() bool { return digitLen == 39 })
	return RESIDUE_LT_HALF
}

func residueFromDecadeWithDigitLen_256(v u256, digitLen int) residue {
	if isZero_bool_128(v.dw2, v.dw3) {
		verify("", func() bool { return digitLen >= 0 && digitLen <= 39 })
		return residueFromDecadeWithDigitLen_128(v.dw0, v.dw1, digitLen)
	}
	if digitLen >= 39 && digitLen <= 77 {
		i := digitLen - 39
		halfPow10 := shr1_256_256(pow10_256_dw0(i), pow10_256_dw1(i), pow10_256_dw2(i), pow10_256_dw3(i))
		ge := 0
		if !v.less(halfPow10) {
			ge = 1
		}
		eq := 0
		if v == halfPow10 {
			eq = 1
		}
		return (ge << 1) | (1 ^ eq)
	}
	impossible("invalid digitLen")
	return 0 // unreachable
}
