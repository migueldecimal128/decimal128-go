// Conversion to a 64-bit integer (IEEE 754 convertToInteger family).
//
// Three separable concerns, kept distinct — and, unlike the old model, with no
// *silent* conversion (WrapperLayer.md §4.8):
//
//   (a) Range guard — FitsInt64 / FitsUInt64: true iff the value is finite and
//       within the closed integer range under *any* rounding direction
//       (rounding-independent, so a later conversion cannot overflow). A range
//       test only; it does not check integrality (3.7 fits). NaN and infinity
//       are false.
//   (b) Fallible-exact value conversion — Int64 / Uint64: yields the integer
//       only when the value is an exact integer in range, else a non-nil error.
//       There is no saturating or clamping form; a fraction, an out-of-range
//       magnitude, NaN, or infinity is a hard failure, not a clamp.
//   (c) Directed / flag-tracking conversion — the Rounding and Context receiver
//       forms below. A Rounding discards the fraction in its direction with no
//       flags; a Context rounds by its rounding and raises InvalidOperation (and
//       optionally Inexact) into the context. These retain the IEEE model — an
//       unspecified sentinel on the invalid path — observable because a context
//       sink is present. The no-flag Rounding form has no sink, so guard it with
//       FitsInt64 first. See WrapperLayer.md §11.

package decimal128

// IntConversionError reports a Decimal128 value that is not an exact integer in
// the target type's range (a fractional part, an out-of-range magnitude, NaN, or
// infinity). Returned by Int64 and Uint64.
type IntConversionError struct {
	// Value is the decimal rendering of the operand that failed to convert.
	Value string
	// Target is the destination integer type ("int64" or "uint64").
	Target string
}

func (e *IntConversionError) Error() string {
	return "decimal128: " + e.Value + " is not an exact " + e.Target
}

// --- (a) range guards ---

// FitsInt64 reports whether d is finite and lies in [math.MinInt64,
// math.MaxInt64] — the rounding-independent guarantee that an int64 conversion
// under any direction cannot overflow. This is a range test only; it does not
// check integrality (3.7 fits). NaN and infinity are false.
func (d Decimal128) FitsInt64() bool { return d128_fitsInt64(d.core) }

// FitsUInt64 reports whether d is finite, non-negative in value, and lies in
// [0, math.MaxUint64]. −0 fits (it converts to 0); a negative nonzero value does
// not. Like FitsInt64, a range test only — it does not check integrality. NaN and
// infinity are false.
func (d Decimal128) FitsUInt64() bool { return d128_fitsUInt64(d.core) }

// --- (b) fallible-exact conversions ---

// Int64 converts d to int64 only when it is an exact integer in range, returning
// a non-nil *IntConversionError otherwise. A fractional part, out-of-range
// magnitude, NaN, or infinity all fail — there is no truncation or clamping. To
// round a fraction, use a directed Rounding/Context form; to test range only, use
// FitsInt64.
func (d Decimal128) Int64() (int64, error) {
	// Integrality: d equals its own truncation. False for any fraction and for
	// NaN; infinity passes here but is rejected by the range guard.
	if !d128_isEqual(d.core, d128_roundToIntegralTowardZero(d.core)) {
		return 0, &IntConversionError{d.String(), "int64"}
	}
	if !d128_fitsInt64(d.core) {
		return 0, &IntConversionError{d.String(), "int64"}
	}
	// Integral and in range → toward-zero extraction is exact.
	return d128_toInt64TowardZero_ctxnull(d.core, nil), nil
}

// Uint64 converts d to uint64 only when it is an exact integer in range,
// returning a non-nil *IntConversionError otherwise. A fractional part, negative
// nonzero value, out-of-range magnitude, NaN, or infinity all fail — no
// truncation or clamping. −0 converts to 0. To test range only, use FitsUInt64.
func (d Decimal128) Uint64() (uint64, error) {
	if !d128_isEqual(d.core, d128_roundToIntegralTowardZero(d.core)) {
		return 0, &IntConversionError{d.String(), "uint64"}
	}
	if !d128_fitsUInt64(d.core) {
		return 0, &IntConversionError{d.String(), "uint64"}
	}
	return d128_toUInt64TowardZero_ctxnull(d.core, nil), nil
}

// --- (c) directed / flag-tracking conversions (unchanged) ---

// ToInt64 converts x to int64, discarding any fraction in this direction, with
// no flag tracking.
func (r Rounding) ToInt64(x Decimal128) int64 { return toInt64Mode_ctxnull(x, r, nil, false) }

// ToInt64Exact is like ToInt64 but is the signaling form (IEEE
// convertToIntegerExact). Without a context there is no sink, so the Inexact
// signal is dropped; use Context.ToInt64Exact to capture it.
func (r Rounding) ToInt64Exact(x Decimal128) int64 { return toInt64Mode_ctxnull(x, r, nil, true) }

// ToInt64 converts x to int64, discarding any fraction by this context's
// rounding. Raises InvalidOperation into the context for a NaN/infinite/
// out-of-range operand; does not raise Inexact for a discarded fraction — use
// ToInt64Exact for that.
func (c *Context) ToInt64(x Decimal128) int64 {
	return toInt64Mode_ctxnull(x, c.rounding, c.core, false)
}

// ToInt64Exact is like ToInt64 but additionally raises Inexact into the context
// when a fractional part is discarded (IEEE convertToIntegerExact).
func (c *Context) ToInt64Exact(x Decimal128) int64 {
	return toInt64Mode_ctxnull(x, c.rounding, c.core, true)
}

// ToUInt64 converts x to uint64, discarding any fraction in this direction, with
// no flag tracking.
func (r Rounding) ToUInt64(x Decimal128) uint64 { return toUInt64Mode_ctxnull(x, r, nil, false) }

// ToUInt64Exact is like ToUInt64 but is the signaling form (IEEE
// convertToIntegerExact). Without a context there is no sink, so the Inexact
// signal is dropped; use Context.ToUInt64Exact to capture it.
func (r Rounding) ToUInt64Exact(x Decimal128) uint64 { return toUInt64Mode_ctxnull(x, r, nil, true) }

// ToUInt64 converts x to uint64, discarding any fraction by this context's
// rounding. Raises InvalidOperation into the context for a NaN/infinite/negative/
// out-of-range operand; does not raise Inexact for a discarded fraction — use
// ToUInt64Exact for that.
func (c *Context) ToUInt64(x Decimal128) uint64 {
	return toUInt64Mode_ctxnull(x, c.rounding, c.core, false)
}

// ToUInt64Exact is like ToUInt64 but additionally raises Inexact into the context
// when a fractional part is discarded (IEEE convertToIntegerExact).
func (c *Context) ToUInt64Exact(x Decimal128) uint64 {
	return toUInt64Mode_ctxnull(x, c.rounding, c.core, true)
}

// toInt64Mode_ctxnull dispatches to the per-direction core conversion, optionally the
// inexact-signaling variant.
func toInt64Mode_ctxnull(x Decimal128, r Rounding, ctx *context, exact bool) int64 {
	if exact {
		switch r {
		case TiesToEven:
			return d128_toInt64TiesToEvenSignalInexact_ctxnull(x.core, ctx)
		case TiesToAway:
			return d128_toInt64TiesToAwaySignalInexact_ctxnull(x.core, ctx)
		case TowardZero:
			return d128_toInt64TowardZeroSignalInexact_ctxnull(x.core, ctx)
		case TowardPositive:
			return d128_toInt64TowardPositiveSignalInexact_ctxnull(x.core, ctx)
		default:
			return d128_toInt64TowardNegativeSignalInexact_ctxnull(x.core, ctx)
		}
	}
	switch r {
	case TiesToEven:
		return d128_toInt64TiesToEven_ctxnull(x.core, ctx)
	case TiesToAway:
		return d128_toInt64TiesToAway_ctxnull(x.core, ctx)
	case TowardZero:
		return d128_toInt64TowardZero_ctxnull(x.core, ctx)
	case TowardPositive:
		return d128_toInt64TowardPositive_ctxnull(x.core, ctx)
	default:
		return d128_toInt64TowardNegative_ctxnull(x.core, ctx)
	}
}

// toUInt64Mode_ctxnull dispatches to the per-direction core conversion, optionally
// the inexact-signaling variant.
func toUInt64Mode_ctxnull(x Decimal128, r Rounding, ctx *context, exact bool) uint64 {
	if exact {
		switch r {
		case TiesToEven:
			return d128_toUInt64TiesToEvenSignalInexact_ctxnull(x.core, ctx)
		case TiesToAway:
			return d128_toUInt64TiesToAwaySignalInexact_ctxnull(x.core, ctx)
		case TowardZero:
			return d128_toUInt64TowardZeroSignalInexact_ctxnull(x.core, ctx)
		case TowardPositive:
			return d128_toUInt64TowardPositiveSignalInexact_ctxnull(x.core, ctx)
		default:
			return d128_toUInt64TowardNegativeSignalInexact_ctxnull(x.core, ctx)
		}
	}
	switch r {
	case TiesToEven:
		return d128_toUInt64TiesToEven_ctxnull(x.core, ctx)
	case TiesToAway:
		return d128_toUInt64TiesToAway_ctxnull(x.core, ctx)
	case TowardZero:
		return d128_toUInt64TowardZero_ctxnull(x.core, ctx)
	case TowardPositive:
		return d128_toUInt64TowardPositive_ctxnull(x.core, ctx)
	default:
		return d128_toUInt64TowardNegative_ctxnull(x.core, ctx)
	}
}
