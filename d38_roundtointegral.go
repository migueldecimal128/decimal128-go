// D38 round-to-integral (ties-to-away and floor) and exact integer extraction,
// used by the exp / exp10 / pow range reduction: n = roundTiesToAway(x / ln10),
// the pow preferred-exponent floor ⌊y·Q(x)⌋, and the integer fast-path test.

package decimal128

import "math"

// d38_roundToIntegralTiesToAway rounds x to an integral value, ties away from
// zero. (Round-half-even is the D38 default elsewhere, but the exp range
// reduction specifically wants ties-to-away.)
func d38_roundToIntegralTiesToAway(x d38) d38 {
	q := d38_qExp(x)
	if q >= 0 {
		return x // already integral
	}
	if d38_isZero(x) {
		return d38Pack(d38_signFlag(x), 0, 0, 0)
	}
	sign := d38_signFlag(x)
	dropDigits := -q
	// _checked_ handles dropDigits ≥ digitLen (where the unchecked variant would
	// trip its pow10 < digitLen precondition), e.g. 0.5 → quot 0 + HALF.
	qr := div_128_pow10_checked_q128res(d38_u128(x), dropDigits)
	quot := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	if qr.residue == RESIDUE_HALF || qr.residue == RESIDUE_GT_HALF {
		quot = add_128_128x128(quot.dw0, quot.dw1, 1, 0) // ties away from zero
	}
	if isZero_bool_128(quot.dw0, quot.dw1) {
		return d38Pack(sign, 0, 0, 0)
	}
	return d38Pack(sign, 0, quot.dw0, quot.dw1)
}

// d38_roundToIntegralTowardNegative rounds x toward −∞ (floor). Used by pow to
// compute the preferred result exponent ⌊y·Q(x)⌋.
func d38_roundToIntegralTowardNegative(x d38) d38 {
	q := d38_qExp(x)
	if q >= 0 {
		return x // already integral
	}
	if d38_isZero(x) {
		return d38Pack(d38_signFlag(x), 0, 0, 0)
	}
	sign := d38_signFlag(x)
	dropDigits := -q
	qr := div_128_pow10_checked_q128res(d38_u128(x), dropDigits)
	quot := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	// floor: positive truncates toward zero; negative with any remainder steps
	// one more negative (magnitude + 1).
	if sign && qr.residue != RESIDUE_EXACT {
		quot = add_128_128x128(quot.dw0, quot.dw1, 1, 0)
	}
	if isZero_bool_128(quot.dw0, quot.dw1) {
		return d38Pack(sign, 0, 0, 0)
	}
	return d38Pack(sign, 0, quot.dw0, quot.dw1)
}

// d38_toLongOrMin returns x's exact integer value when x is an exact integer in
// int64 range; otherwise the sentinel math.MinInt64.
func d38_toLongOrMin(x d38) int64 {
	if d38_isZero(x) {
		return 0
	}
	q := d38_qExp(x)
	sign := d38_signFlag(x)
	digitLen := d38_digitLen(x)
	u := d38_u128(x)

	var mag u128
	if q == 0 {
		mag = u
	} else if q > 0 {
		if q > 18 {
			return math.MinInt64
		}
		mag = mulPow10_128_128(u.dw0, u.dw1, q) // ≤19 digits, fits U128
	} else {
		// q < 0: integral only if the low (−q) digits are all zero.
		dropDigits := -q
		if dropDigits >= digitLen {
			return math.MinInt64 // |x| < 1
		}
		qr := div_128_pow10_q128res(u, dropDigits)
		if qr.residue != RESIDUE_EXACT {
			return math.MinInt64
		}
		mag = fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	}

	if mag.dw1 != 0 {
		return math.MinInt64 // ≥ 2^64
	}
	m := uint64(mag.dw0)
	if sign {
		if m > uint64(math.MaxInt64)+1 {
			return math.MinInt64
		}
		return int64(^m + 1) // −m
	}
	if m > uint64(math.MaxInt64) {
		return math.MinInt64
	}
	return int64(m)
}
