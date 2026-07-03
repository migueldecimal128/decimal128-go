// The length family (PrimitiveLayer.md D6, D22): bitLen/digitLen over 128- and
// 256-bit magnitudes. Limb forms (ascending dwords; trailing precomputed bitLen
// last). The digit-length estimate uses the (bitLen * 1233) >> 12 ≈ bitLen *
// log10(2) lower bound, then one table compare to settle the exact count.

package decimal128

import "math/bits"

func calcBitLenInt(x int) int {
	verify("", func() bool { return x >= 0 })
	return bits.UintSize - bits.LeadingZeros(uint(x))
}

func digitLen_int_32(x int) int {
	verify("", func() bool { return x >= 0 })
	bitLen := calcBitLenInt(x)
	x64 := uint64(int64(x))
	return calcDigitLen64(bitLen, x64)
}

func calcDigitLen64(bitLen int, x uint64) int {
	loDigitCount := (bitLen * 1233) >> 12
	hiDigitCount := loDigitCount + 1
	if x < uint64(pow10_128_lo[loDigitCount]) {
		return loDigitCount
	}
	return hiDigitCount
}

func bitLen_int_128(xDw0, xDw1 int64) int {
	if xDw1 != 0 {
		return 128 - bits.LeadingZeros64(uint64(xDw1))
	}
	return 64 - bits.LeadingZeros64(uint64(xDw0))
}

func digitLen_int_128(xDw0, xDw1 int64) int {
	return digitLen_int_128_withBitLen(xDw0, xDw1, bitLen_int_128(xDw0, xDw1))
}

func digitLen_int_128_withBitLen(xDw0, xDw1 int64, bitLen int) int {
	loDigitCount := (bitLen * 1233) >> 12
	hiDigitCount := loDigitCount + 1
	if cmp_int_128x128(xDw0, xDw1, pow10_128_lo[loDigitCount], pow10_128_hi[loDigitCount]) < 0 {
		return loDigitCount
	}
	return hiDigitCount
}

func bitLen_int_256(xDw0, xDw1, xDw2, xDw3 int64) int {
	if (xDw2 | xDw3) != 0 {
		if xDw3 != 0 {
			return 256 - bits.LeadingZeros64(uint64(xDw3))
		}
		return 192 - bits.LeadingZeros64(uint64(xDw2))
	}
	if xDw1 != 0 {
		return 128 - bits.LeadingZeros64(uint64(xDw1))
	}
	return 64 - bits.LeadingZeros64(uint64(xDw0))
}

func digitLen_int_256(xDw0, xDw1, xDw2, xDw3 int64) int {
	if (xDw2 | xDw3) != 0 {
		return digitLen_int_256_withBitLen(xDw0, xDw1, xDw2, xDw3, bitLen_int_256(xDw0, xDw1, xDw2, xDw3))
	}
	return digitLen_int_128_withBitLen(xDw0, xDw1, bitLen_int_128(xDw0, xDw1))
}

func digitLen_int_256_withBitLen(xDw0, xDw1, xDw2, xDw3 int64, bitLen int) int {
	if bitLen <= 128 {
		return digitLen_int_128_withBitLen(xDw0, xDw1, bitLen)
	}
	// bitLen > 128: needs the POW10_256 lookup. 10^39 has bit length 130.
	loDigitCount := max((bitLen*1233)>>12, 39)
	b := loDigitCount - 39
	pow := u256{pow10_256_d0[b], pow10_256_d1[b], pow10_256_d2[b], pow10_256_d3[b]}
	x := u256{xDw0, xDw1, xDw2, xDw3}
	if x.less(pow) {
		return loDigitCount
	}
	return loDigitCount + 1
}

// --- u256 sugar (§3.4) ---

// less is the unsigned U256 ordering (Swift's U256.<): compare high half, then
// low half.
func (x u256) less(y u256) bool {
	ch := cmp_int_128x128(x.dw2, x.dw3, y.dw2, y.dw3)
	if ch != 0 {
		return ch < 0
	}
	return cmp_int_128x128(x.dw0, x.dw1, y.dw0, y.dw1) < 0
}

func (x u256) digitLen() int {
	return digitLen_int_256(x.dw0, x.dw1, x.dw2, x.dw3)
}
