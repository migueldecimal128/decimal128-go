// Magnitude comparison in the 38-digit D38 domain. A binary-exponent bracket
// pre-check (shared with the D128 path) resolves any case whose scientific
// exponents differ without aligning — which also bounds the fall-through align
// shift to ≤37 digits, keeping the U256 product in range. Ties align both
// coefficients to a common scale in U256 and compare. Returns -1/0/1. Operands
// must be finite non-zero.

package decimal128

func cmpMagD38(x, y d38) int {
	// The bExp pre-check below requires non-zero operands (digitLen > 0); the
	// ln/log10 √-reduction compares a possibly-zero z = c'−1, so resolve zeros
	// first. |0| < |nonzero|.
	xZero := d38_isZero(x)
	yZero := d38_isZero(y)
	if xZero || yZero {
		switch {
		case xZero && yZero:
			return 0
		case xZero:
			return -1
		default:
			return 1
		}
	}

	xBitLen := bitLen_int_128(x.coeffLo64, x.coeffHi64)
	yBitLen := bitLen_int_128(y.coeffLo64, y.coeffHi64)
	xDigitLen := digitLen_int_128_withBitLen(x.coeffLo64, x.coeffHi64, xBitLen)
	yDigitLen := digitLen_int_128_withBitLen(y.coeffLo64, y.coeffHi64, yBitLen)
	cmpBExp := compareBExpMagnitudeFnzFnz(d38_qExp(x), xBitLen, xDigitLen, d38_qExp(y), yBitLen, yDigitLen)
	if cmpBExp != 0 {
		return cmpBExp
	}

	qDelta := d38_qExp(x) - d38_qExp(y)
	if qDelta == 0 {
		return cmp_int_128x128(x.coeffLo64, x.coeffHi64, y.coeffLo64, y.coeffHi64)
	}
	var xAligned, yAligned u256
	if qDelta > 0 {
		xAligned = mulPow10_256_128(x.coeffLo64, x.coeffHi64, qDelta)
		yAligned = u256FromU128(d38_u128(y))
	} else {
		xAligned = u256FromU128(d38_u128(x))
		yAligned = mulPow10_256_128(y.coeffLo64, y.coeffHi64, -qDelta)
	}
	if xAligned.less(yAligned) {
		return -1
	}
	if xAligned == yAligned {
		return 0
	}
	return 1
}
