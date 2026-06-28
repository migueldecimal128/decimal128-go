// decimal128 → string formatting (GDAS to-scientific/engineering/plain, plus the
// coefficient×10^qExp debug form). d128_format is the boundary entry.
// Renders directly into a heap-allocated String.

package decimal128

import "unsafe"

// Format-style selector codes for d128_format.
const (
	FORMAT_AUTO             = 0 // GDAS auto: plain within [-6,0], scientific otherwise
	FORMAT_EXPONENTIAL      = 1 // always exponential
	FORMAT_ENGINEERING      = 2 // scientific, exponent a multiple of 3
	FORMAT_COEFFICIENT_QEXP = 3 // raw coefficient × 10^qExp
)

func withStringBuffer(length int, f func(p unsafe.Pointer)) string {
	buf := make([]byte, length)
	if length > 0 {
		f(unsafe.Pointer(&buf[0]))
	}
	return unsafe.String(unsafe.SliceData(buf), length)
}

func d128_format(x d128, formatStyle int) string {
	return d128ToString(x, formatStyle)
}

func d128ToString(x d128, formatStyle int) string {
	if d128_isFIN(x) {
		return toFiniteString(d128_signFlag(x), d128_qExp(x), d128_u128(x), formatStyle)
	}
	return toNonFiniteString(x)
}

// [Inf+, Inf-, NaN+, NaN-, sNaN+, sNaN-]
var specialValueStrings = []string{"Infinity", "-Infinity", "NaN", "-NaN", "sNaN", "-sNaN"}

func toNonFiniteString(x d128) string {
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

	nanPrefLen := 3
	if snanBit == 1 {
		nanPrefLen = 4
	}
	prefLen := int(signBit) + nanPrefLen
	digitLen := digitLen_int_128(nanPayload.dw0, nanPayload.dw1)
	totalLen := prefLen + digitLen

	return withStringBuffer(totalLen, func(ascii unsafe.Pointer) {
		if signBit == 1 {
			writeByte(ascii, 0, ascii_minus)
		}
		i := int(signBit)
		if snanBit == 1 {
			writeByte(ascii, i, ascii_s)
			i++
		}
		writeByte(ascii, i, ascii_N)
		i++
		writeByte(ascii, i, ascii_a)
		i++
		writeByte(ascii, i, ascii_N)
		i++
		u128ToASCII(digitLen, nanPayload, ascii, i)
	})
}

func toFiniteString(sign bool, qExp int, coeff u128, formatStyle int) string {
	digitLen := digitLen_int_128(coeff.dw0, coeff.dw1)
	eExp := qExp + digitLen + (-digitLen >> 63)
	if formatStyle == FORMAT_COEFFICIENT_QEXP {
		return toCoefficientQExpString(sign, qExp, coeff, digitLen)
	}
	if formatStyle != FORMAT_EXPONENTIAL {
		if qExp == 0 {
			return toIntegerString(sign, coeff)
		}
		if qExp < 0 && eExp >= minPlainExponent {
			return toDecimalPointString(sign, qExp, coeff, digitLen)
		}
		if formatStyle == FORMAT_ENGINEERING {
			return toEngineeringString(sign, qExp, eExp, coeff, digitLen)
		}
	}
	return toExponentialString(sign, qExp, eExp, coeff, digitLen)
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

func toIntegerString(sign bool, coeff u128) string {
	signBit := boolToInt64(sign)
	if cmp_int_128x128(coeff.dw0, coeff.dw1, smallIntegerMaxx, 0) < 0 {
		raw := (signBit << smallIntegerMaxxPow2) + coeff.dw0
		i := int(raw & int64(smallIntegerBCE))
		return smallIntegerStrings[i]
	}
	digitLen := digitLen_int_128(coeff.dw0, coeff.dw1)
	signLen := int(signBit)
	totalLen := signLen + digitLen

	return withStringBuffer(totalLen, func(ascii unsafe.Pointer) {
		if sign {
			writeByte(ascii, 0, ascii_minus)
		}
		u128ToASCII(digitLen, coeff, ascii, signLen)
	})
}

func toDecimalPointString(sign bool, qExp int, coeff u128, digitLen int) string {
	digitsRightOfDecimal := -qExp
	leadingZeroCount := max(1+digitsRightOfDecimal-digitLen, 0)
	signLen := 0
	if sign {
		signLen = 1
	}
	totalLen := signLen + leadingZeroCount + 1 + digitLen

	return withStringBuffer(totalLen, func(ascii unsafe.Pointer) {
		if sign {
			writeByte(ascii, 0, ascii_minus)
		}
		if leadingZeroCount > 0 {
			writeByte(ascii, signLen, ascii_0)
			writeByte(ascii, signLen+1, ascii_dot)
			for idx := signLen + 2; idx < signLen+leadingZeroCount+1; idx++ {
				writeByte(ascii, idx, ascii_0)
			}
			u128ToASCII(digitLen, coeff, ascii, signLen+leadingZeroCount+1)
		} else {
			u128ToASCII(digitLen, coeff, ascii, signLen)
			iDot := totalLen - digitsRightOfDecimal - 1
			i := totalLen - 1
			for i > iDot {
				writeByte(ascii, i, readByte(ascii, i-1))
				i--
			}
			writeByte(ascii, iDot, ascii_dot)
		}
	})
}

func toExponentialString(sign bool, qExp, eExp int, coeff u128, digitLen int) string {
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

	return withStringBuffer(totalLen, func(ascii unsafe.Pointer) {
		if sign {
			writeByte(ascii, 0, ascii_minus)
		}
		u128ToASCII(printedDigitLen, coeff, ascii, signLen+decimalPointLen)
		if decimalPointLen > 0 {
			writeByte(ascii, signLen, readByte(ascii, signLen+1))
			writeByte(ascii, signLen+1, ascii_dot)
		}
		iE := signLen + decimalPointLen + printedDigitLen
		writeByte(ascii, iE, ascii_E)
		writeByte(ascii, iE+1, expSignByte)
		renderTailDigitsBeforeIndex(expDigitLen, int64(eExpAbs), ascii, totalLen)
	})
}

func toEngineeringString(sign bool, qExp, eExp int, coeff u128, digitLen int) string {
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
		i = signLen + 1 + (func() int {
			if zerExpAdjustment == 0 {
				return 0
			}
			return 1 + zerExpAdjustment
		})()
	} else {
		expAlignZeroCount := min(max(0, 1+fnzExpAdjustment-digitLen), 2)
		leftOfRadixPointCount := 1 + fnzExpAdjustment
		decimalPointLen := 0
		if digitLen > leftOfRadixPointCount {
			decimalPointLen = 1
		}
		i = signLen + decimalPointLen + digitLen + expAlignZeroCount
	}

	totalLen := i + (func() int {
		if adjustedExp == 0 {
			return 0
		}
		return 1 + 1 + expDigitLen
	})()

	return withStringBuffer(totalLen, func(ascii unsafe.Pointer) {
		if sign {
			writeByte(ascii, 0, ascii_minus)
		}
		if isZero {
			writeByte(ascii, signLen, ascii_0)
			if zerExpAdjustment > 0 {
				writeByte(ascii, signLen+1, ascii_dot)
				writeByte(ascii, signLen+2, ascii_0)
				if zerExpAdjustment > 1 {
					writeByte(ascii, signLen+3, ascii_0)
				}
			}
		} else {
			expAlignZeroCount := min(max(0, 1+fnzExpAdjustment-digitLen), 2)
			leftOfRadixPointCount := 1 + fnzExpAdjustment
			decimalPointLen := 0
			if digitLen > leftOfRadixPointCount {
				decimalPointLen = 1
			}
			u128ToASCII(digitLen, coeff, ascii, signLen+decimalPointLen)

			if expAlignZeroCount >= 1 {
				startZeroIdx := signLen + decimalPointLen + digitLen
				if expAlignZeroCount > 1 {
					writeByte(ascii, startZeroIdx, ascii_0)
					writeByte(ascii, startZeroIdx+1, ascii_0)
				} else {
					writeByte(ascii, startZeroIdx, ascii_0)
				}
			} else if digitLen > leftOfRadixPointCount {
				j := 0
				for j < leftOfRadixPointCount {
					writeByte(ascii, signLen+j, readByte(ascii, signLen+j+1))
					j++
				}
				writeByte(ascii, signLen+leftOfRadixPointCount, ascii_dot)
			}
		}

		if adjustedExp != 0 {
			writeByte(ascii, i, ascii_E)
			writeByte(ascii, i+1, expSign)
			uIntToASCII(expDigitLen, adjustedExpAbs, ascii, i+2)
		}
	})
}

func toCoefficientQExpString(sign bool, qExp int, coeff u128, digitLen int) string {
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

	return withStringBuffer(totalLen, func(ascii unsafe.Pointer) {
		if sign {
			writeByte(ascii, 0, ascii_minus)
		}
		u128ToASCII(printedDigitLen, coeff, ascii, signLen)
		i := signLen + printedDigitLen
		writeByte(ascii, i, ascii_E)
		writeByte(ascii, i+1, expSignByte)
		u64ToASCII(expDigitLen, int64(qExpAbs), ascii, i+2)
	})
}
