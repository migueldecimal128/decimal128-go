// IEEE 754-2019 §5.6.1 comparison. Two tri-state cores (quiet / signaling) over
// Comparison754, the 22 required Boolean predicates, and the Int CMP754_* shims.
// NaN operands yield UNORDERED; the quiet cores signal invalid only on sNaN, the
// signaling cores on any NaN. The magnitude compare (cmpMagFnzFnz) lives in
// compare_mag.go.

package decimal128

// --- tri-state cores ---

// IEEE 754 quiet compare core. NaN operands yield `.unordered`.
// Signaling NaNs do not signal under `Quiet` (flags discarded).
func comparePartialQuiet754(x, y d128) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return const_CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

// IEEE 754 signaling compare core. Same as `comparePartialQuiet754` but signals
// invalid on any NaN (quiet or signaling). Under `Quiet`, signaling is a no-op.
func comparePartialSignaling754(x, y d128) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		signalInvalidCompare_quiet()
		return const_CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

// --- Int boundary (quiet) ---

// IEEE 754 quiet compare. Returns a `CMP754_*` code; NaN operands yield
// `CMP754_UNORDERED`.
func d128_comparePartial(x, y d128) int { return comparePartialQuiet754(x, y) }

// IEEE 754 signaling compare. As `d128_comparePartial`, but signals invalid on any
// NaN. Returns a `CMP754_*` code.
func d128_comparePartialSignaling(x, y d128) int { return comparePartialSignaling754(x, y) }

// Magnitude compare of two finite-nonzero values: -1 / 0 / +1. Shared with the
// add and subtract cores.
func cmpMagFnzFnz(x, y d128) int {
	verify("", func() bool { return d128_isFNZ(x) && d128_isFNZ(y) })
	qDelta := d128_qExp(x) - d128_qExp(y)
	if qDelta >= 34 {
		return 1
	}
	if qDelta <= -34 {
		return -1
	}
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
	// |qDelta| is bounded by digitLen_max (34 for decimal128) because sciExps
	// match, so qDelta = digitLen_y - digitLen_x. The binary-bracket pre-check
	// makes typical qDelta small but doesn't tighten this worst-case bound.
	if qDelta > 0 {
		xAligned = mulPow10_128_128(xAligned.dw0, xAligned.dw1, qDelta)
	} else if qDelta < 0 {
		yAligned = mulPow10_128_128(yAligned.dw0, yAligned.dw1, -qDelta)
	}
	return cmp_int_128x128(xAligned.dw0, xAligned.dw1, yAligned.dw0, yAligned.dw1)
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

func eExpFnz(qExp, digitLen int) int {
	verify("", func() bool { return digitLen > 0 })
	return qExp + digitLen - 1
}

// --- quiet Boolean predicates (11) ---

func d128_isEqual(x, y d128) bool    { return comparePartialQuiet754(x, y) == const_CMP754_EQ }
func d128_isNotEqual(x, y d128) bool { return comparePartialQuiet754(x, y) != const_CMP754_EQ }
func d128_isLess(x, y d128) bool     { return comparePartialQuiet754(x, y) == const_CMP754_LT }

func d128_isLessEqual(x, y d128) bool {
	r := comparePartialQuiet754(x, y)
	return r == const_CMP754_LT || r == const_CMP754_EQ
}
func d128_isGreater(x, y d128) bool { return comparePartialQuiet754(x, y) == const_CMP754_GT }

func d128_isGreaterEqual(x, y d128) bool {
	r := comparePartialQuiet754(x, y)
	return r == const_CMP754_GT || r == const_CMP754_EQ
}
func d128_isUnordered(x, y d128) bool { return comparePartialQuiet754(x, y) == const_CMP754_UNORDERED }
func d128_isOrdered(x, y d128) bool   { return comparePartialQuiet754(x, y) != const_CMP754_UNORDERED }

// `not(x > y)` -- true when `x <= y` or unordered. Distinct from
// `isLessEqual` because NaN cases differ: this returns true on
// unordered, the other false.
func d128_isNotGreater(x, y d128) bool { return comparePartialQuiet754(x, y) != const_CMP754_GT }

// `not(x < y)` -- true when `x >= y` or unordered.
func d128_isNotLess(x, y d128) bool { return comparePartialQuiet754(x, y) != const_CMP754_LT }

// True when `x > y` or unordered.
func d128_isGreaterUnordered(x, y d128) bool {
	r := comparePartialQuiet754(x, y)
	return r == const_CMP754_GT || r == const_CMP754_UNORDERED
}

// True when `x < y` or unordered.
func d128_isLessUnordered(x, y d128) bool {
	r := comparePartialQuiet754(x, y)
	return r == const_CMP754_LT || r == const_CMP754_UNORDERED
}

// --- signaling Boolean predicates (11) — quiet-path no-op, same values ---

func d128_isEqualSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) == const_CMP754_EQ
}
func d128_isNotEqualSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) != const_CMP754_EQ
}
func d128_isLessSignaling(x, y d128) bool { return comparePartialSignaling754(x, y) == const_CMP754_LT }

func d128_isLessEqualSignaling(x, y d128) bool {
	r := comparePartialSignaling754(x, y)
	return r == const_CMP754_LT || r == const_CMP754_EQ
}
func d128_isGreaterSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) == const_CMP754_GT
}

func d128_isGreaterEqualSignaling(x, y d128) bool {
	r := comparePartialSignaling754(x, y)
	return r == const_CMP754_GT || r == const_CMP754_EQ
}
func d128_isUnorderedSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) == const_CMP754_UNORDERED
}
func d128_isOrderedSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) != const_CMP754_UNORDERED
}
func d128_isNotGreaterSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) != const_CMP754_GT
}
func d128_isNotLessSignaling(x, y d128) bool {
	return comparePartialSignaling754(x, y) != const_CMP754_LT
}
func d128_isGreaterUnorderedSignaling(x, y d128) bool {
	r := comparePartialSignaling754(x, y)
	return r == const_CMP754_GT || r == const_CMP754_UNORDERED
}
func d128_isLessUnorderedSignaling(x, y d128) bool {
	r := comparePartialSignaling754(x, y)
	return r == const_CMP754_LT || r == const_CMP754_UNORDERED
}

func compareFinFin(x, y d128) comparison754 {
	verify("", func() bool { return bothFinite(x, y) })
	if d128_isZER(x) {
		if d128_isZER(y) {
			return const_CMP754_EQ
		}
		if d128_signFlag(y) {
			return const_CMP754_GT
		}
		return const_CMP754_LT
	}
	if d128_isZER(y) {
		if d128_signFlag(x) {
			return const_CMP754_LT
		}
		return const_CMP754_GT
	}
	xSignFlag := d128_signFlag(x)
	if xSignFlag != d128_signFlag(y) {
		if xSignFlag {
			return const_CMP754_LT
		}
		return const_CMP754_GT
	}
	cmpMag := cmpMagFnzFnz(x, y)
	if cmpMag == 0 {
		return const_CMP754_EQ
	}
	signed := cmpMag
	if xSignFlag {
		signed = -cmpMag
	}
	return comparison754FromSignum(signed)
}

func compareNonFinite(x, y d128) comparison754 {
	verify("", func() bool { return !d128_isNAN(x) && !d128_isNAN(y) })
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if d128_isINF(x) && d128_isINF(y) {
		if d128_signFlag(x) == d128_signFlag(y) {
			return const_CMP754_EQ
		}
		if d128_signFlag(x) {
			return const_CMP754_LT
		}
		return const_CMP754_GT
	}
	if d128_isINF(x) {
		if d128_signFlag(x) {
			return const_CMP754_LT
		}
		return const_CMP754_GT
	}
	if d128_signFlag(y) {
		return const_CMP754_GT
	}
	return const_CMP754_LT
}

// --- context-aware cores ---

func comparePartialQuiet754_ctx(x, y d128, ctx *context) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		if d128_isSNAN(x) || d128_isSNAN(y) {
			ctxSignalInvalidOperation_ctxnull(ctx, causeSnanOperand, const_D128_QNAN0)
		}
		return const_CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

func comparePartialSignaling754_ctx(x, y d128, ctx *context) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		cause := causeNanOperand
		if d128_isSNAN(x) || d128_isSNAN(y) {
			cause = causeSnanOperand
		}
		ctxSignalInvalidOperation_ctxnull(ctx, cause, const_D128_QNAN0)
		return const_CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

func d128_comparePartial_ctx(x, y d128, ctx *context) int {
	return comparePartialQuiet754_ctx(x, y, ctx)
}
func d128_comparePartialSignaling_ctx(x, y d128, ctx *context) int {
	return comparePartialSignaling754_ctx(x, y, ctx)
}

// --- quiet Boolean predicates (ctx) ---

func d128_isEqual_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) == const_CMP754_EQ
}
func d128_isNotEqual_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) != const_CMP754_EQ
}
func d128_isLess_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) == const_CMP754_LT
}

func d128_isLessEqual_ctx(x, y d128, ctx *context) bool {
	r := comparePartialQuiet754_ctx(x, y, ctx)
	return r == const_CMP754_LT || r == const_CMP754_EQ
}
func d128_isGreater_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) == const_CMP754_GT
}

func d128_isGreaterEqual_ctx(x, y d128, ctx *context) bool {
	r := comparePartialQuiet754_ctx(x, y, ctx)
	return r == const_CMP754_GT || r == const_CMP754_EQ
}
func d128_isUnordered_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) == const_CMP754_UNORDERED
}
func d128_isOrdered_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) != const_CMP754_UNORDERED
}
func d128_isNotGreater_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) != const_CMP754_GT
}
func d128_isNotLess_ctx(x, y d128, ctx *context) bool {
	return comparePartialQuiet754_ctx(x, y, ctx) != const_CMP754_LT
}
func d128_isGreaterUnordered_ctx(x, y d128, ctx *context) bool {
	r := comparePartialQuiet754_ctx(x, y, ctx)
	return r == const_CMP754_GT || r == const_CMP754_UNORDERED
}
func d128_isLessUnordered_ctx(x, y d128, ctx *context) bool {
	r := comparePartialQuiet754_ctx(x, y, ctx)
	return r == const_CMP754_LT || r == const_CMP754_UNORDERED
}

// --- signaling Boolean predicates (ctx) ---

func d128_isEqualSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) == const_CMP754_EQ
}
func d128_isNotEqualSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) != const_CMP754_EQ
}
func d128_isLessSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) == const_CMP754_LT
}

func d128_isLessEqualSignaling_ctx(x, y d128, ctx *context) bool {
	r := comparePartialSignaling754_ctx(x, y, ctx)
	return r == const_CMP754_LT || r == const_CMP754_EQ
}
func d128_isGreaterSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) == const_CMP754_GT
}

func d128_isGreaterEqualSignaling_ctx(x, y d128, ctx *context) bool {
	r := comparePartialSignaling754_ctx(x, y, ctx)
	return r == const_CMP754_GT || r == const_CMP754_EQ
}
func d128_isUnorderedSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) == const_CMP754_UNORDERED
}
func d128_isOrderedSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) != const_CMP754_UNORDERED
}
func d128_isNotGreaterSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) != const_CMP754_GT
}
func d128_isNotLessSignaling_ctx(x, y d128, ctx *context) bool {
	return comparePartialSignaling754_ctx(x, y, ctx) != const_CMP754_LT
}
func d128_isGreaterUnorderedSignaling_ctx(x, y d128, ctx *context) bool {
	r := comparePartialSignaling754_ctx(x, y, ctx)
	return r == const_CMP754_GT || r == const_CMP754_UNORDERED
}
func d128_isLessUnorderedSignaling_ctx(x, y d128, ctx *context) bool {
	r := comparePartialSignaling754_ctx(x, y, ctx)
	return r == const_CMP754_LT || r == const_CMP754_UNORDERED
}
