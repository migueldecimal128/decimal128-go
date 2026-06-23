// decimal128 → string formatting (GDAS to-scientific/engineering/plain, plus the
// coefficient×10^qExp debug form). d128_format is the boundary entry.

package decimal128

// Format-style selector codes for d128_format.
const (
	FORMAT_AUTO             = 0 // GDAS auto: plain within [-6,0], scientific otherwise
	FORMAT_EXPONENTIAL      = 1 // always exponential
	FORMAT_ENGINEERING      = 2 // scientific, exponent a multiple of 3
	FORMAT_COEFFICIENT_QEXP = 3 // raw coefficient × 10^qExp
)

func d128_format(x d128, formatStyle int) string {
	return d128ToString(x, formatStyle)
}

func d128ToString(x d128, formatStyle int) string {
	var ascii asciiBuffer
	ascii[0] = ascii_minus // overwritten for non-negative values
	if d128_isFIN(x) {
		return toFiniteString(d128_signFlag(x), d128_qExp(x), d128_u128(x), formatStyle, &ascii)
	}
	return toNonFiniteString(x, &ascii)
}

// [Inf+, Inf-, NaN+, NaN-, sNaN+, sNaN-]
var specialValueStrings = []string{"Infinity", "-Infinity", "NaN", "-NaN", "sNaN", "-sNaN"}

func toNonFiniteString(x d128, ascii *asciiBuffer) string {
	signBit := d128_signBit(x)
	if d128_isINF(x) {
		return specialValueStrings[signBit]
	}
	snanBit := 0
	if !d128_isNotSNAN(x) {
		snanBit = 1
	}
	nanIndex := 2 + (snanBit << 1)
	nanStr := specialValueStrings[nanIndex+signBit]
	nanPayload := d128_nanPayload(x)
	if isZero_bool_128(nanPayload.dw0, nanPayload.dw1) {
		return nanStr
	}
	// mixed-case: 's' lower, 'N' upper, 'a' lower, 'N' upper
	i := signBit
	ascii[i&asciiBCE] = ascii_s
	i += snanBit
	ascii[i&asciiBCE] = ascii_N
	i++
	ascii[i&asciiBCE] = ascii_a
	i++
	ascii[i&asciiBCE] = ascii_N
	i++
	digitLen := digitLen_int_128(nanPayload.dw0, nanPayload.dw1)
	totalLen := i + digitLen
	u128ToASCII(digitLen, nanPayload, ascii, i)
	return ascii.toString(totalLen)
}

func toFiniteString(sign bool, qExp int, coeff u128, formatStyle int, ascii *asciiBuffer) string {
	digitLen := digitLen_int_128(coeff.dw0, coeff.dw1)
	eExp := qExp + digitLen + (-digitLen >> 63)
	if formatStyle == FORMAT_COEFFICIENT_QEXP {
		return toCoefficientQExpString(sign, qExp, coeff, digitLen, ascii)
	}
	if formatStyle != FORMAT_EXPONENTIAL {
		if qExp == 0 {
			return toIntegerString(sign, coeff, ascii)
		}
		if qExp < 0 && eExp >= minPlainExponent {
			return toDecimalPointString(sign, qExp, coeff, digitLen, ascii)
		}
		if formatStyle == FORMAT_ENGINEERING {
			return toEngineeringString(sign, qExp, eExp, coeff, digitLen, ascii)
		}
	}
	return toExponentialString(sign, qExp, eExp, coeff, digitLen, ascii)
}

const minPlainExponent = -6

var smallIntegerStrings = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15",
	"-0", "-1", "-2", "-3", "-4", "-5", "-6", "-7", "-8", "-9", "-10", "-11", "-12", "-13", "-14", "-15",
}

const (
	smallIntegerMaxxPow2       = 4
	smallIntegerMaxx     int64 = 1 << smallIntegerMaxxPow2
	smallIntegerBCE            = (1 << (smallIntegerMaxxPow2 + 1)) - 1
)

func toIntegerString(sign bool, coeff u128, ascii *asciiBuffer) string {
	signBit := boolToInt64(sign)
	if cmp_int_128x128(coeff.dw0, coeff.dw1, smallIntegerMaxx, 0) < 0 {
		raw := (signBit << smallIntegerMaxxPow2) + coeff.dw0
		i := int(raw & int64(smallIntegerBCE))
		return smallIntegerStrings[i]
	}
	verify("", ascii[0] == ascii_minus)
	digitLen := digitLen_int_128(coeff.dw0, coeff.dw1)
	signLen := int(signBit)
	u128ToASCII(digitLen, coeff, ascii, signLen)
	totalLen := signLen + digitLen
	return ascii.toString(totalLen)
}

func toDecimalPointString(sign bool, qExp int, coeff u128, digitLen int, ascii *asciiBuffer) string {
	digitsRightOfDecimal := -qExp
	leadingZeroCount := max(1+digitsRightOfDecimal-digitLen, 0)
	signLen := 0
	if sign {
		signLen = 1
	}
	totalLen := signLen + leadingZeroCount + 1 + digitLen
	verify("", ascii[0] == ascii_minus)
	i := signLen
	for i <= leadingZeroCount { // one extra here
		ascii[i&asciiBCE] = ascii_0
		i++
	}
	u128ToASCII(digitLen, coeff, ascii, signLen+leadingZeroCount)
	iDot := totalLen - digitsRightOfDecimal - 1
	i = totalLen - 1
	for i > iDot {
		ascii[i&asciiBCE] = ascii[(i-1)&asciiBCE]
		i--
	}
	ascii[iDot&asciiBCE] = ascii_dot
	return ascii.toString(totalLen)
}

func toExponentialString(sign bool, qExp, eExp int, coeff u128, digitLen int, ascii *asciiBuffer) string {
	eExpMask := eExp >> 31
	eExpAbs := (eExp ^ eExpMask) - eExpMask
	signLen := 0
	if sign {
		signLen = 1
	}
	decimalPointLen := 0
	if digitLen > 1 {
		decimalPointLen = 1
	}
	printedDigitLen := digitLen + 1 + (-digitLen >> 63)
	expSignByte := ascii_plus
	if eExp < 0 {
		expSignByte = ascii_minus
	}
	expDigitLen := max(digitLen_int_32(eExpAbs), 1)
	totalLen := signLen + decimalPointLen + printedDigitLen + 1 + 1 + expDigitLen
	verify("", ascii[0] == ascii_minus)
	u128ToASCII(printedDigitLen, coeff, ascii, signLen+decimalPointLen)
	if decimalPointLen > 0 {
		ascii[signLen] = ascii[signLen+1]
		ascii[signLen+1] = ascii_dot
	}
	iE := signLen + decimalPointLen + printedDigitLen
	ascii[iE] = ascii_E
	ascii[iE+1] = expSignByte
	renderTailDigitsBeforeIndex(expDigitLen, int64(eExpAbs), ascii, totalLen)
	return ascii.toString(totalLen)
}

func toEngineeringString(sign bool, qExp, eExp int, coeff u128, digitLen int, ascii *asciiBuffer) string {
	signLen := 0
	if sign {
		signLen = 1
	}
	isZero := digitLen == 0
	zerExpAdjustment := 0
	fnzExpAdjustment := 0
	if isZero {
		zerExpAdjustment = (3 - ((eExp%3)+3)%3) % 3
	} else {
		if eExp >= 0 {
			fnzExpAdjustment = eExp % 3
		} else {
			fnzExpAdjustment = ((eExp % 3) + 3) % 3
		}
	}
	adjustedExp := eExp + zerExpAdjustment - fnzExpAdjustment

	adjustedExpMask := adjustedExp >> 63
	adjustedExpAbs := (adjustedExp ^ adjustedExpMask) - adjustedExpMask
	expSign := ascii_plus
	if adjustedExp < 0 {
		expSign = ascii_minus
	}
	expDigitLen := max(digitLen_int_32(adjustedExpAbs), 1)
	var i int
	if isZero {
		ascii[(signLen+0)&asciiBCE] = ascii_0
		ascii[(signLen+1)&asciiBCE] = ascii_dot
		ascii[(signLen+2)&asciiBCE] = ascii_0
		ascii[(signLen+3)&asciiBCE] = ascii_0
		if zerExpAdjustment == 0 {
			i = signLen + 1
		} else {
			i = signLen + 1 + 1 + zerExpAdjustment
		}
	} else {
		expAlignZeroCount := min(max(0, 1+fnzExpAdjustment-digitLen), 2)
		leftOfRadixPointCount := 1 + fnzExpAdjustment
		decimalPointLen := 0
		if digitLen > leftOfRadixPointCount {
			decimalPointLen = 1
		}
		u128ToASCII(digitLen, coeff, ascii, signLen+decimalPointLen)
		i = signLen + decimalPointLen + digitLen
		if expAlignZeroCount >= 1 {
			if expAlignZeroCount > 1 {
				ascii[i&asciiBCE] = ascii_0
				i++
			}
			ascii[i&asciiBCE] = ascii_0
			i++
		} else if digitLen > leftOfRadixPointCount {
			j := 0
			for j < leftOfRadixPointCount {
				ascii[(signLen+j)&asciiBCE] = ascii[signLen+j+1]
				j++
			}
			ascii[(signLen+leftOfRadixPointCount)&asciiBCE] = ascii_dot
		}
	}
	if adjustedExp == 0 {
		return ascii.toString(i)
	}
	ascii[i&asciiBCE] = ascii_E
	i++
	ascii[i&asciiBCE] = expSign
	i++
	uIntToASCII(expDigitLen, adjustedExpAbs, ascii, i)
	totalLen := i + expDigitLen
	return ascii.toString(totalLen)
}

func toCoefficientQExpString(sign bool, qExp int, coeff u128, digitLen int, ascii *asciiBuffer) string {
	signLen := 0
	if sign {
		signLen = 1
	}
	qExpAbs := qExp
	if qExpAbs < 0 {
		qExpAbs = -qExp
	}
	printedDigitLen := max(digitLen, 1)
	expSignByte := ascii_plus
	if qExp < 0 {
		expSignByte = ascii_minus
	}
	expDigitLen := max(digitLen_int_32(qExpAbs), 1)
	totalLen := signLen + printedDigitLen + 1 + 1 + expDigitLen
	verify("", ascii[0] == ascii_minus)
	i := signLen
	u128ToASCII(printedDigitLen, coeff, ascii, signLen)
	i += printedDigitLen
	ascii[i&asciiBCE] = ascii_E
	i++
	ascii[i&asciiBCE] = expSignByte
	i++
	u64ToASCII(expDigitLen, int64(qExpAbs), ascii, i)
	i += expDigitLen
	verify("", i == totalLen)
	return ascii.toString(totalLen)
}
