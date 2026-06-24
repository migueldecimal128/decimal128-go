// IEEE minimum/maximum, finer classification, and sign composition. (The
// total-order predicate lives in decimal_comparison.go alongside the other
// comparisons.)

package decimal128

// --- finer-grained classification ---

// IsNormal reports whether d is finite, nonzero, and not subnormal.
func (d Decimal128) IsNormal() bool { return d128_isNormal(d.core) }

// IsSubnormal reports whether d is finite, nonzero, and below the normal range
// (reduced precision).
func (d Decimal128) IsSubnormal() bool { return d128_isSubnormal(d.core) }

// IsCanonical reports whether d is in canonical encoding (always true for the
// library's internal representation).
func (d Decimal128) IsCanonical() bool { return d128_isCanonical(d.core) }

// --- IEEE minimum / maximum (NaN-propagating) ---
//
// These are the IEEE 754-2019 minimum/maximum operations: if either operand is
// NaN the result is NaN. They have no flag-tracking context form (no _ctx core
// entry), so they are package functions rather than Context methods.

// Minimum returns the lesser of x and y; NaN if either is NaN.
func Minimum(x, y Decimal128) Decimal128 { return Decimal128{d128_minimum(x.core, y.core)} }

// Maximum returns the greater of x and y; NaN if either is NaN.
func Maximum(x, y Decimal128) Decimal128 { return Decimal128{d128_maximum(x.core, y.core)} }

// MinimumMagnitude returns the operand with the lesser magnitude (NaN if either
// is NaN).
func MinimumMagnitude(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimumMagnitude(x.core, y.core)}
}

// MaximumMagnitude returns the operand with the greater magnitude (NaN if either
// is NaN).
func MaximumMagnitude(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximumMagnitude(x.core, y.core)}
}

// --- IEEE minimumNumber / maximumNumber (NaN-ignoring) ---
//
// The GDAS / IEEE 754-2019 "Number" forms: if exactly one operand is NaN the
// other (the number) is returned; a signaling NaN still raises InvalidOperation
// — use the Context forms to capture it.

// MinimumNumber returns the lesser of x and y, ignoring a single NaN operand.
func MinimumNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimumNumber(x.core, y.core)}
}

// MaximumNumber returns the greater of x and y, ignoring a single NaN operand.
func MaximumNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximumNumber(x.core, y.core)}
}

// MinimumMagnitudeNumber returns the lesser-magnitude operand, ignoring a single
// NaN operand.
func MinimumMagnitudeNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimumMagnitudeNumber(x.core, y.core)}
}

// MaximumMagnitudeNumber returns the greater-magnitude operand, ignoring a
// single NaN operand.
func MaximumMagnitudeNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximumMagnitudeNumber(x.core, y.core)}
}

// --- sign composition ---

// CopySign returns a value with the magnitude of d and the sign of sign
// (IEEE copySign).
func (d Decimal128) CopySign(sign Decimal128) Decimal128 {
	return Decimal128{d128_copySign(d.core, sign.core)}
}

// --- context-receiver minimumNumber / maximumNumber (flag-tracking) ---
//
// The flag-tracking forms of the NaN-ignoring "Number" selections: a signaling
// NaN operand raises InvalidOperation into the context. The NaN-propagating
// Minimum/Maximum have no context form.

// MinimumNumber returns the lesser of x and y, ignoring a single NaN operand;
// flag-tracked through this context.
func (c *Context) MinimumNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimumNumber_ctx(x.core, y.core, c.core)}
}

// MaximumNumber returns the greater of x and y, ignoring a single NaN operand;
// flag-tracked through this context.
func (c *Context) MaximumNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximumNumber_ctx(x.core, y.core, c.core)}
}

// MinimumMagnitudeNumber returns the lesser-magnitude operand, ignoring a single
// NaN operand; flag-tracked through this context.
func (c *Context) MinimumMagnitudeNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimumMagnitudeNumber_ctx(x.core, y.core, c.core)}
}

// MaximumMagnitudeNumber returns the greater-magnitude operand, ignoring a
// single NaN operand; flag-tracked through this context.
func (c *Context) MaximumMagnitudeNumber(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximumMagnitudeNumber_ctx(x.core, y.core, c.core)}
}
