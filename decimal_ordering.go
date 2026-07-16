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
// a quiet NaN the result is NaN. A signaling NaN operand still raises
// InvalidOperation; use the flag-tracking (c *Context) forms below to capture
// it. The propagating and "Number" families signal on sNaN identically and
// differ only in how they treat a quiet NaN. The package-level value forms here
// discard flags; the Context methods track them.

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

// Copysign returns a value with the magnitude of d and the sign of sign
// (IEEE copySign).
func (d Decimal128) Copysign(sign Decimal128) Decimal128 {
	return Decimal128{d128_copySign(d.core, sign.core)}
}

// --- context-receiver minimum / maximum (flag-tracking) ---
//
// Flag-tracking forms of both selection families: a signaling NaN operand raises
// InvalidOperation into the context. The NaN-propagating Minimum/Maximum and the
// NaN-ignoring …Number forms signal on sNaN identically, differing only in
// quiet-NaN handling (propagating returns NaN; …Number returns the number).

// Minimum returns the lesser of x and y; NaN if either is a quiet NaN.
// A signaling NaN operand raises InvalidOperation into this context.
func (c *Context) Minimum(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimum_ctx(x.core, y.core, c.core)}
}

// Maximum returns the greater of x and y; NaN if either is a quiet NaN.
// A signaling NaN operand raises InvalidOperation into this context.
func (c *Context) Maximum(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximum_ctx(x.core, y.core, c.core)}
}

// MinimumMagnitude returns the lesser-magnitude operand; NaN if either is a
// quiet NaN. A signaling NaN operand raises InvalidOperation into this context.
func (c *Context) MinimumMagnitude(x, y Decimal128) Decimal128 {
	return Decimal128{d128_minimumMagnitude_ctx(x.core, y.core, c.core)}
}

// MaximumMagnitude returns the greater-magnitude operand; NaN if either is a
// quiet NaN. A signaling NaN operand raises InvalidOperation into this context.
func (c *Context) MaximumMagnitude(x, y Decimal128) Decimal128 {
	return Decimal128{d128_maximumMagnitude_ctx(x.core, y.core, c.core)}
}

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
