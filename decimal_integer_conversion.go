// Conversion to a 64-bit integer (IEEE 754 convertToInteger family).
//
// The rounding direction selects how a fractional part is discarded. The bare
// form on Decimal128 truncates toward zero with no flags (the int64(float64)
// analog); the Rounding forms pick a direction; the Context forms additionally
// track flags.
//
// A NaN, infinite, or out-of-range operand is an invalid conversion: it raises
// InvalidOperation (when a Context is the receiver) and returns an unspecified
// sentinel. Guard such inputs with IsFinite and a range check, or inspect the
// context's flags.

package decimal128

// ToInt64 converts d to int64, truncating any fraction toward zero, with no flag
// tracking. For another direction use Rounding.ToInt64; for flag tracking use
// Context.ToInt64.
func (d Decimal128) ToInt64() int64 { return d128_toInt64TowardZero_ctxnull(d.core, nil) }

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
