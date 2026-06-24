// IEEE 754-2019 §5.3.1 minimum / maximum operations.
//
// Eight variants from three orthogonal axes:
//   - min vs max
//   - numeric vs not (numeric: NaN treated as missing operand)
//   - magnitude vs not (compare |x| vs |y|; numeric value used to break ties)
//
// On magnitude ties, the numeric-value comparison determines the result
// (e.g. minimumMagnitude(-3, +3) returns -3). On numeric-value ties between
// distinct cohorts or signed zeros, totalOrder breaks the tie.
//
// These quiet free functions are value-only (flag-free even on signaling NaN);
// flag observability is provided by the _ctx forms in d128_ctx_minmax.go.

package decimal128

func d128_minimum(x, y d128) d128 { return minMaxImpl(x, y, minOp) }
func d128_maximum(x, y d128) d128 { return minMaxImpl(x, y, maxOp) }

func d128_minimumNumber(x, y d128) d128 { return minMaxImpl(x, y, minNumOp) }
func d128_maximumNumber(x, y d128) d128 { return minMaxImpl(x, y, maxNumOp) }

func d128_minimumMagnitude(x, y d128) d128 { return minMaxImpl(x, y, minMagOp) }
func d128_maximumMagnitude(x, y d128) d128 { return minMaxImpl(x, y, maxMagOp) }

func d128_minimumMagnitudeNumber(x, y d128) d128 { return minMaxImpl(x, y, minMagNumOp) }
func d128_maximumMagnitudeNumber(x, y d128) d128 { return minMaxImpl(x, y, maxMagNumOp) }

// Op encoding (matches the sibling ports).
const (
	maxMask = 1
	magMask = 2
	numMask = 4

	minOp       = 0
	maxOp       = maxMask
	minMagOp    = magMask
	maxMagOp    = maxMask | magMask
	minNumOp    = numMask
	maxNumOp    = maxMask | numMask
	minMagNumOp = magMask | numMask
	maxMagNumOp = maxMask | magMask | numMask
)

func minMaxImpl(x, y d128, op int) d128 {
	if !d128_isNAN(x) && !d128_isNAN(y) {
		return minMaxNonNaN(x, y, op)
	}
	return minMaxNaN(x, y, op)
}

func minMaxNonNaN(x, y d128, op int) d128 {
	var cmp int
	if (op & magMask) != 0 {
		cmp = compareNumericMagnitude(x, y)
	} else {
		cmp = compareNumericSigned(x, y)
	}
	if cmp == 0 {
		cmp = compareTotalOrder(x, y)
	}
	// cmp >= 0 means x >= y. For max we want x; for min we want y.
	pickX := (cmp >= 0) == ((op & maxMask) != 0)
	if pickX {
		return x
	}
	return y
}

func minMaxNaN(x, y d128, op int) d128 {
	verify("", d128_isNAN(x) || d128_isNAN(y))
	if (op & numMask) != 0 {
		// numeric variant: NaN treated as a missing operand.
		if !d128_isNAN(x) {
			return x
		}
		if !d128_isNAN(y) {
			return y
		}
		// both NaN: fall through to propagation
	}
	return propagateNAN(x, y)
}

func propagateNAN(x, y d128) d128 {
	verify("", d128_isNAN(x) || d128_isNAN(y))
	if d128_isSNAN(x) {
		return d128_quieted(x)
	}
	if d128_isSNAN(y) {
		return d128_quieted(y)
	}
	if d128_isNAN(x) {
		return d128_quieted(x)
	}
	return d128_quieted(y)
}

// compareNumericSigned is a signed numeric compare for min/max returning -1/0/1.
// Assumes no NaN. Treats +0 == -0; cohort differences yield 0.
func compareNumericSigned(x, y d128) int {
	verify("", !d128_isNAN(x) && !d128_isNAN(y))
	if bothFinite(x, y) {
		if d128_isZER(x) && d128_isZER(y) {
			return 0
		}
		if d128_isZER(x) {
			if d128_signFlag(y) {
				return 1
			}
			return -1
		}
		if d128_isZER(y) {
			if d128_signFlag(x) {
				return -1
			}
			return 1
		}
		// both FNZ
		xSignFlag := d128_signFlag(x)
		if xSignFlag != d128_signFlag(y) {
			if xSignFlag {
				return -1
			}
			return 1
		}
		cmpMag := cmpMagFnzFnz(x, y)
		if xSignFlag {
			return -cmpMag
		}
		return cmpMag
	}
	// at least one INF
	if d128_isINF(x) && d128_isINF(y) {
		if d128_signFlag(x) == d128_signFlag(y) {
			return 0
		}
		if d128_signFlag(x) {
			return -1
		}
		return 1
	}
	if d128_isINF(x) {
		if d128_signFlag(x) {
			return -1
		}
		return 1
	}
	if d128_signFlag(y) {
		return 1
	}
	return -1
}

// compareNumericMagnitude is an unsigned magnitude compare returning -1/0/1.
// Assumes no NaN.
func compareNumericMagnitude(x, y d128) int {
	verify("", !d128_isNAN(x) && !d128_isNAN(y))
	if d128_isFNZ(x) && d128_isFNZ(y) {
		return cmpMagFnzFnz(x, y)
	}
	// INF/ZER combinations only. INF beats anything non-INF; ZER loses to
	// anything non-ZER; equal-class ties to 0.
	xInf, yInf, xZer, yZer := 0, 0, 0, 0
	if d128_isINF(x) {
		xInf = 1
	}
	if d128_isINF(y) {
		yInf = 1
	}
	if d128_isZER(x) {
		xZer = 1
	}
	if d128_isZER(y) {
		yZer = 1
	}
	return (xInf - yInf) | (yZer - xZer)
}
