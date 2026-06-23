// Magnitude compare of two FNZ (finite nonzero) values, used by add/subtract and
// the compare family. A binary-exponent bracket pre-check (compareBExpMagnitude)
// resolves most cases without aligning; ties fall through to an aligned 128-bit
// compare. The full IEEE compare surface lands with the compare operators.

package decimal128

func eExpFnz(qExp, digitLen int) int {
	verify("", digitLen > 0)
	return qExp + digitLen - 1
}

func compareBExpMagnitudeFnzFnz(xQExp, xBitLen, xDigitLen, yQExp, yBitLen, yDigitLen int) int {
	xE := eExpFnz(xQExp, xDigitLen)
	yE := eExpFnz(yQExp, yDigitLen)
	if xE != yE {
		return ((xE - yE) >> 63) | 1 // -1 or 1
	}
	if calcBExpMin(xBitLen, xQExp) > calcBExpMax(yBitLen, yQExp) {
		return 1
	}
	if calcBExpMax(xBitLen, xQExp) < calcBExpMin(yBitLen, yQExp) {
		return -1
	}
	return 0
}

func cmpMagFnzFnz(x, y d128) int {
	verify("", d128_isFNZ(x) && d128_isFNZ(y))
	xCoeff := d128_u128(x)
	yCoeff := d128_u128(y)
	xBitLen := bitLen_int_128(xCoeff.dw0, xCoeff.dw1)
	yBitLen := bitLen_int_128(yCoeff.dw0, yCoeff.dw1)
	xDigitLen := digitLen_int_128_withBitLen(xCoeff.dw0, xCoeff.dw1, xBitLen)
	yDigitLen := digitLen_int_128_withBitLen(yCoeff.dw0, yCoeff.dw1, yBitLen)
	cmpBExp := compareBExpMagnitudeFnzFnz(d128_qExp(x), xBitLen, xDigitLen, d128_qExp(y), yBitLen, yDigitLen)
	if cmpBExp != 0 {
		return cmpBExp
	}
	xAligned := xCoeff
	yAligned := yCoeff
	qDelta := d128_qExp(x) - d128_qExp(y)
	if qDelta > 0 {
		xAligned = mulPow10_128_128(xAligned.dw0, xAligned.dw1, qDelta)
	} else if qDelta < 0 {
		yAligned = mulPow10_128_128(yAligned.dw0, yAligned.dw1, -qDelta)
	}
	return cmp_int_128x128(xAligned.dw0, xAligned.dw1, yAligned.dw0, yAligned.dw1)
}
