// IEEE 754 §5.10 totalOrder.
//
// Ordering (least to greatest):
//   -qNaN < -sNaN < -Inf < -finite < -0 < +0 < +finite < +Inf < +sNaN < +qNaN
//
// Within a finite cohort: for positives, smaller exponent comes first
// (1.0 < 1.00); for negatives, larger exponent comes first (-1.00 < -1.0).
// NaNs are ordered by payload (sign-adjusted).

package decimal128

// d128_isTotallyOrdered reports whether x totalOrders before-or-equal-to other.
func d128_isTotallyOrdered(x, other d128) bool {
	return compareTotalOrder(x, other) <= 0
}

// d128_isTotallyOrderedMag reports whether |x| totalOrders before-or-equal-to |other|.
func d128_isTotallyOrderedMag(x, other d128) bool {
	return compareTotalOrderMag(x, other) <= 0
}

// compareTotalOrder returns negative / 0 / positive, like memcmp.
func compareTotalOrder(x, y d128) int {
	xSignMask := d128_signMask(x) // 0 or -1
	ySignMask := d128_signMask(y)
	if xSignMask != ySignMask {
		// x negative, y non-negative -> x first (-1)
		// x non-negative, y negative -> y first (+1)
		return (xSignMask << 1) + 1
	}
	cmpMag := compareTotalOrderMag(x, y)
	return negateForSign(cmpMag, xSignMask)
}

func compareTotalOrderMag(x, y d128) int {
	xNaN := d128_isNAN(x)
	yNaN := d128_isNAN(y)
	if !xNaN && !yNaN {
		return totalOrderCompareMagNonNaN(x, y)
	}
	return compareTotalOrderMagNaN(x, y, xNaN, yNaN)
}

func totalOrderCompareMagNonNaN(x, y d128) int {
	if d128_isFNZ(x) && d128_isFNZ(y) {
		return compareTotalOrderMagFnzFnz(x, y)
	}
	// INF beats everything else in magnitude.
	if d128_isINF(x) {
		if d128_isINF(y) {
			return 0
		}
		return 1
	}
	if d128_isINF(y) {
		return -1
	}
	// No INF, no NaN. Remaining: FNZ vs ZER, ZER vs FNZ, ZER vs ZER.
	// FNZ has greater magnitude than ZER.
	if d128_isFNZ(x) {
		return 1
	}
	if d128_isFNZ(y) {
		return -1
	}
	// both ZER (possibly distinct cohorts)
	verify("", d128_isZER(x) && d128_isZER(y))
	return cmpInt(d128_qExp(x), d128_qExp(y))
}

func compareTotalOrderMagFnzFnz(x, y d128) int {
	cmpMag := cmpMagFnzFnz(x, y)
	if cmpMag != 0 {
		return cmpMag
	}
	// Equal magnitudes (same cohort): tie-break on exponent.
	return cmpInt(d128_qExp(x), d128_qExp(y))
}

func compareTotalOrderMagNaN(x, y d128, xNaN, yNaN bool) int {
	verify("", xNaN || yNaN)
	if !xNaN {
		return -1 // non-NaN < NaN in magnitude
	}
	if !yNaN {
		return 1
	}
	// Both NaN. sNaN sorts after qNaN in the positive half (larger magnitude).
	xSNAN := d128_isSNAN(x)
	ySNAN := d128_isSNAN(y)
	if xSNAN != ySNAN {
		if xSNAN {
			return -1
		}
		return 1
	}
	// Same NaN kind: order by payload.
	xPayload := d128_nanPayload(x)
	yPayload := d128_nanPayload(y)
	return cmp_int_128x128(xPayload.dw0, xPayload.dw1, yPayload.dw0, yPayload.dw1)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func negateForSign(cmp, signMask int) int {
	return (cmp ^ signMask) - signMask
}
