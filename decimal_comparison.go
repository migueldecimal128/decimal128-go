// Three-way comparison: the numeric IEEE 754 comparison (which can be
// unordered) and the total-order comparison (which never is), plus the named
// quiet predicates and the flag-tracking context forms.

package decimal128

// Comparison is the outcome of comparing two Decimal128 values. CmpUnordered
// arises only from the numeric comparisons when an operand is NaN; the
// total-order comparisons never produce it.
type Comparison int

const (
	CmpLess      Comparison = iota // the receiver is less than the operand
	CmpEqual                       // the two compare equal
	CmpGreater                     // the receiver is greater than the operand
	CmpUnordered                   // an operand is NaN — not numerically ordered
)

// String returns a short name for the comparison outcome.
func (c Comparison) String() string {
	switch c {
	case CmpLess:
		return "Less"
	case CmpEqual:
		return "Equal"
	case CmpGreater:
		return "Greater"
	case CmpUnordered:
		return "Unordered"
	default:
		return "Comparison(?)"
	}
}

// comparisonFromCode maps a core CMP754_* code to the public Comparison.
func comparisonFromCode(code comparison754) Comparison {
	switch code {
	case CMP754_LT:
		return CmpLess
	case CMP754_EQ:
		return CmpEqual
	case CMP754_GT:
		return CmpGreater
	default:
		return CmpUnordered
	}
}

// --- numeric three-way comparison (IEEE 754 compare) ---

// Compare returns the quiet IEEE 754 comparison of d to other. A NaN operand
// yields CmpUnordered; a signaling-NaN operand additionally raises
// InvalidOperation — use Context.Compare to capture that flag.
func (d Decimal128) Compare(other Decimal128) Comparison {
	return comparisonFromCode(d128_compare(d.core, other.core))
}

// --- total-order comparison (never unordered) ---

// TotalOrder returns the IEEE 754 total-ordering comparison: a reflexive
// ordering over all values — NaNs, signed zeros, and distinct cohorts included —
// so it never returns CmpUnordered.
func (d Decimal128) TotalOrder(other Decimal128) Comparison {
	return totalThreeWay(d.IsTotallyOrdered(other), other.IsTotallyOrdered(d))
}

// TotalOrderByMagnitude returns the total ordering of magnitudes (signs
// ignored). Never CmpUnordered.
func (d Decimal128) TotalOrderByMagnitude(other Decimal128) Comparison {
	return totalThreeWay(d.IsTotallyOrderedByMagnitude(other), other.IsTotallyOrderedByMagnitude(d))
}

// IsTotallyOrdered reports whether d totalOrders at or before other (the
// reflexive total-order ≤).
func (d Decimal128) IsTotallyOrdered(other Decimal128) bool {
	return d128_isTotallyOrdered(d.core, other.core)
}

// IsTotallyOrderedByMagnitude is the magnitude form of IsTotallyOrdered (signs
// ignored).
func (d Decimal128) IsTotallyOrderedByMagnitude(other Decimal128) bool {
	return d128_isTotallyOrderedMag(d.core, other.core)
}

// totalThreeWay collapses the two directions of a total-order ≤ test to a
// three-way result.
func totalThreeWay(leftBelowEqual, rightBelowEqual bool) Comparison {
	if leftBelowEqual && rightBelowEqual {
		return CmpEqual
	}
	if leftBelowEqual {
		return CmpLess
	}
	return CmpGreater
}

// --- named quiet predicates ---
//
// Equal and Less are already provided on Decimal128 (the IEEE == and < forms);
// these add the remaining named predicates.

// LessOrEqual reports whether d ≤ other under IEEE rules (false if either is NaN).
func (d Decimal128) LessOrEqual(other Decimal128) bool {
	return d128_isLessEqual(d.core, other.core)
}

// IsUnordered reports whether either operand is NaN (so the two are not ordered).
func (d Decimal128) IsUnordered(other Decimal128) bool {
	return d128_isUnordered(d.core, other.core)
}

// --- context-receiver comparison (flag-tracking) ---

// Compare returns the quiet IEEE 754 comparison of a to b, recording any raised
// flag (InvalidOperation on a signaling-NaN operand) into this context. NaN
// operands yield CmpUnordered.
func (c *Context) Compare(a, b Decimal128) Comparison {
	return comparisonFromCode(d128_compare_ctx(a.core, b.core, c.core))
}

// CompareSignaling returns the signaling IEEE 754 comparison of a to b: any NaN
// operand (quiet or signaling) raises InvalidOperation into this context.
// Returns CmpUnordered for NaN.
func (c *Context) CompareSignaling(a, b Decimal128) Comparison {
	return comparisonFromCode(d128_compareSignaling_ctx(a.core, b.core, c.core))
}
