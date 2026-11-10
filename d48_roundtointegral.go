// D48 round-to-integral (ties-to-away and toward-negative) and the exact
// D48 → int64 conversion used by the integer-exponent math kernels (Swift
// D48RoundToIntegral.swift).

package decimal128

import "math"

// d48_roundToIntegralTiesToAway rounds x to an integral value, ties away from
// zero (the exp range reduction).
func d48_roundToIntegralTiesToAway(x d48) d48 {
	q := d48_qExp(x)
	if q >= 0 {
		return x // already integral
	}
	sign := d48_signFlag(x)
	if d48_isZero(x) {
		return d48_fromSignQExp_160(sign, 0, u192{})
	}
	qr := divPow10_checked_q320res(d48_u320(x), -q)
	quot := qr.quot()
	if (qr.residue == RESIDUE_HALF) || (qr.residue == RESIDUE_GT_HALF) {
		quot = add_320_320x64(quot, 1) // ties away from zero
	}
	return d48_fromSignQExp_160(sign, 0, quot.u192())
}

// d48_roundToIntegralTowardNegative is floor(x): positive truncates toward
// zero; negative with any remainder steps one more negative (magnitude + 1).
func d48_roundToIntegralTowardNegative(x d48) d48 {
	q := d48_qExp(x)
	if q >= 0 {
		return x // already integral
	}
	sign := d48_signFlag(x)
	if d48_isZero(x) {
		return d48_fromSignQExp_160(sign, 0, u192{})
	}
	qr := divPow10_checked_q320res(d48_u320(x), -q)
	quot := qr.quot()
	if sign && (qr.residue != RESIDUE_EXACT) {
		quot = add_320_320x64(quot, 1)
	}
	return d48_fromSignQExp_160(sign, 0, quot.u192())
}

// d48_toLongOrMin returns x's exact integer value when x is an exact integer in
// int64 range; otherwise the sentinel math.MinInt64.
func d48_toLongOrMin(x d48) int64 {
	if d48_isZero(x) {
		return 0
	}
	q := d48_qExp(x)
	sign := d48_signFlag(x)
	digitLen := d48_digitLen(x)
	c := d48_u320(x)

	var mag u320
	if q == 0 {
		mag = c
	} else if q > 0 {
		if q > 18 {
			return math.MinInt64
		}
		mag = mulPow10_320(c, q) // <= 66 digits, fits u320
	} else {
		dropDigits := -q
		if dropDigits >= digitLen {
			return math.MinInt64 // |x| < 1
		}
		qr := divPow10_q320res(c, dropDigits)
		if qr.residue != RESIDUE_EXACT {
			return math.MinInt64
		}
		mag = qr.quot()
	}

	if (mag.dw1 | mag.dw2 | mag.dw3 | mag.dw4) != 0 {
		return math.MinInt64 // >= 2^64
	}
	m := uint64(mag.dw0)
	if sign {
		if m > uint64(math.MaxInt64)+1 {
			return math.MinInt64
		}
		return int64(-m) // −m
	}
	if m > uint64(math.MaxInt64) {
		return math.MinInt64
	}
	return int64(m)
}
