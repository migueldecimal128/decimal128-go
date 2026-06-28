// IEEE 754-2019 §5.6.1 comparison. Two tri-state cores (quiet / signaling) over
// Comparison754, the 22 required Boolean predicates, and the Int CMP754_* shims.
// NaN operands yield UNORDERED; the quiet cores signal invalid only on sNaN, the
// signaling cores on any NaN. The magnitude compare (cmpMagFnzFnz) lives in
// compare_mag.go.

package decimal128

// Comparison754 tri-state codes (the wrapper/core boundary value). Comparison754
// is an int alias so .rawValue is identity.
type comparison754 = int

const (
	CMP754_LT        comparison754 = 0
	CMP754_EQ        comparison754 = 1
	CMP754_GT        comparison754 = 2
	CMP754_UNORDERED comparison754 = 3
)

func comparison754FromSignum(signum int) comparison754 {
	verify("", signum >= -1 && signum <= 1)
	return signum + 1
}

// --- tri-state cores ---

func compareQuiet754(x, y d128) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

func compareSignaling754(x, y d128) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return CMP754_UNORDERED // signal is a no-op on the quiet path
	}
	return compareNonFinite(x, y)
}

func compareFinFin(x, y d128) comparison754 {
	verify("", bothFinite(x, y))
	if d128_isZER(x) {
		if d128_isZER(y) {
			return CMP754_EQ
		}
		if d128_signFlag(y) {
			return CMP754_GT
		}
		return CMP754_LT
	}
	if d128_isZER(y) {
		if d128_signFlag(x) {
			return CMP754_LT
		}
		return CMP754_GT
	}
	xSignFlag := d128_signFlag(x)
	if xSignFlag != d128_signFlag(y) {
		if xSignFlag {
			return CMP754_LT
		}
		return CMP754_GT
	}
	cmpMag := cmpMagFnzFnz(x, y)
	if cmpMag == 0 {
		return CMP754_EQ
	}
	signed := cmpMag
	if xSignFlag {
		signed = -cmpMag
	}
	return comparison754FromSignum(signed)
}

func compareNonFinite(x, y d128) comparison754 {
	verify("", !d128_isNAN(x) && !d128_isNAN(y))
	verify("", d128_isINF(x) || d128_isINF(y))
	if d128_isINF(x) && d128_isINF(y) {
		if d128_signFlag(x) == d128_signFlag(y) {
			return CMP754_EQ
		}
		if d128_signFlag(x) {
			return CMP754_LT
		}
		return CMP754_GT
	}
	if d128_isINF(x) {
		if d128_signFlag(x) {
			return CMP754_LT
		}
		return CMP754_GT
	}
	if d128_signFlag(y) {
		return CMP754_GT
	}
	return CMP754_LT
}

// --- Int boundary (quiet) ---

func d128_compare(x, y d128) int          { return compareQuiet754(x, y) }
func d128_compareSignaling(x, y d128) int { return compareSignaling754(x, y) }

// --- quiet Boolean predicates (11) ---

func d128_isEqual(x, y d128) bool      { return compareQuiet754(x, y) == CMP754_EQ }
func d128_isNotEqual(x, y d128) bool   { return compareQuiet754(x, y) != CMP754_EQ }
func d128_isLess(x, y d128) bool       { return compareQuiet754(x, y) == CMP754_LT }
func d128_isGreater(x, y d128) bool    { return compareQuiet754(x, y) == CMP754_GT }
func d128_isUnordered(x, y d128) bool  { return compareQuiet754(x, y) == CMP754_UNORDERED }
func d128_isOrdered(x, y d128) bool    { return compareQuiet754(x, y) != CMP754_UNORDERED }
func d128_isNotGreater(x, y d128) bool { return compareQuiet754(x, y) != CMP754_GT }
func d128_isNotLess(x, y d128) bool    { return compareQuiet754(x, y) != CMP754_LT }
func d128_isLessEqual(x, y d128) bool {
	r := compareQuiet754(x, y)
	return r == CMP754_LT || r == CMP754_EQ
}
func d128_isGreaterEqual(x, y d128) bool {
	r := compareQuiet754(x, y)
	return r == CMP754_GT || r == CMP754_EQ
}
func d128_isGreaterUnordered(x, y d128) bool {
	r := compareQuiet754(x, y)
	return r == CMP754_GT || r == CMP754_UNORDERED
}
func d128_isLessUnordered(x, y d128) bool {
	r := compareQuiet754(x, y)
	return r == CMP754_LT || r == CMP754_UNORDERED
}

// --- signaling Boolean predicates (11) — quiet-path no-op, same values ---

func d128_isEqualSignaling(x, y d128) bool      { return compareSignaling754(x, y) == CMP754_EQ }
func d128_isNotEqualSignaling(x, y d128) bool   { return compareSignaling754(x, y) != CMP754_EQ }
func d128_isLessSignaling(x, y d128) bool       { return compareSignaling754(x, y) == CMP754_LT }
func d128_isGreaterSignaling(x, y d128) bool    { return compareSignaling754(x, y) == CMP754_GT }
func d128_isUnorderedSignaling(x, y d128) bool  { return compareSignaling754(x, y) == CMP754_UNORDERED }
func d128_isOrderedSignaling(x, y d128) bool    { return compareSignaling754(x, y) != CMP754_UNORDERED }
func d128_isNotGreaterSignaling(x, y d128) bool { return compareSignaling754(x, y) != CMP754_GT }
func d128_isNotLessSignaling(x, y d128) bool    { return compareSignaling754(x, y) != CMP754_LT }
func d128_isLessEqualSignaling(x, y d128) bool {
	r := compareSignaling754(x, y)
	return r == CMP754_LT || r == CMP754_EQ
}
func d128_isGreaterEqualSignaling(x, y d128) bool {
	r := compareSignaling754(x, y)
	return r == CMP754_GT || r == CMP754_EQ
}
func d128_isGreaterUnorderedSignaling(x, y d128) bool {
	r := compareSignaling754(x, y)
	return r == CMP754_GT || r == CMP754_UNORDERED
}
func d128_isLessUnorderedSignaling(x, y d128) bool {
	r := compareSignaling754(x, y)
	return r == CMP754_LT || r == CMP754_UNORDERED
}

// --- context-aware cores ---

func compareQuiet754_ctx(x, y d128, ctx *context) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		if d128_isSNAN(x) || d128_isSNAN(y) {
			ctxSignalInvalidOperation_ctxnull(ctx, causeSnanOperand)
		}
		return CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

func compareSignaling754_ctx(x, y d128, ctx *context) comparison754 {
	if bothFinite(x, y) {
		return compareFinFin(x, y)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		cause := causeNanOperand
		if d128_isSNAN(x) || d128_isSNAN(y) {
			cause = causeSnanOperand
		}
		ctxSignalInvalidOperation_ctxnull(ctx, cause)
		return CMP754_UNORDERED
	}
	return compareNonFinite(x, y)
}

func d128_compare_ctx(x, y d128, ctx *context) int { return compareQuiet754_ctx(x, y, ctx) }
func d128_compareSignaling_ctx(x, y d128, ctx *context) int {
	return compareSignaling754_ctx(x, y, ctx)
}

// --- quiet Boolean predicates (ctx) ---

func d128_isEqual_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) == CMP754_EQ
}
func d128_isNotEqual_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) != CMP754_EQ
}
func d128_isLess_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) == CMP754_LT
}
func d128_isGreater_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) == CMP754_GT
}
func d128_isUnordered_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) == CMP754_UNORDERED
}
func d128_isOrdered_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) != CMP754_UNORDERED
}
func d128_isNotGreater_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) != CMP754_GT
}
func d128_isNotLess_ctx(x, y d128, ctx *context) bool {
	return compareQuiet754_ctx(x, y, ctx) != CMP754_LT
}
func d128_isLessEqual_ctx(x, y d128, ctx *context) bool {
	r := compareQuiet754_ctx(x, y, ctx)
	return r == CMP754_LT || r == CMP754_EQ
}
func d128_isGreaterEqual_ctx(x, y d128, ctx *context) bool {
	r := compareQuiet754_ctx(x, y, ctx)
	return r == CMP754_GT || r == CMP754_EQ
}
func d128_isGreaterUnordered_ctx(x, y d128, ctx *context) bool {
	r := compareQuiet754_ctx(x, y, ctx)
	return r == CMP754_GT || r == CMP754_UNORDERED
}
func d128_isLessUnordered_ctx(x, y d128, ctx *context) bool {
	r := compareQuiet754_ctx(x, y, ctx)
	return r == CMP754_LT || r == CMP754_UNORDERED
}

// --- signaling Boolean predicates (ctx) ---

func d128_isEqualSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) == CMP754_EQ
}
func d128_isNotEqualSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) != CMP754_EQ
}
func d128_isLessSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) == CMP754_LT
}
func d128_isGreaterSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) == CMP754_GT
}
func d128_isUnorderedSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) == CMP754_UNORDERED
}
func d128_isOrderedSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) != CMP754_UNORDERED
}
func d128_isNotGreaterSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) != CMP754_GT
}
func d128_isNotLessSignaling_ctx(x, y d128, ctx *context) bool {
	return compareSignaling754_ctx(x, y, ctx) != CMP754_LT
}
func d128_isLessEqualSignaling_ctx(x, y d128, ctx *context) bool {
	r := compareSignaling754_ctx(x, y, ctx)
	return r == CMP754_LT || r == CMP754_EQ
}
func d128_isGreaterEqualSignaling_ctx(x, y d128, ctx *context) bool {
	r := compareSignaling754_ctx(x, y, ctx)
	return r == CMP754_GT || r == CMP754_EQ
}
func d128_isGreaterUnorderedSignaling_ctx(x, y d128, ctx *context) bool {
	r := compareSignaling754_ctx(x, y, ctx)
	return r == CMP754_GT || r == CMP754_UNORDERED
}
func d128_isLessUnorderedSignaling_ctx(x, y d128, ctx *context) bool {
	r := compareSignaling754_ctx(x, y, ctx)
	return r == CMP754_LT || r == CMP754_UNORDERED
}
