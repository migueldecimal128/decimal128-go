// Rounding-to-integral, adjacent values, scaling, and quantum operations. Bare
// (quiet, ties-to-even) forms and the quantum predicates are methods on
// Decimal128; the directed round-to-integral lives on Rounding; the flag-tracking
// forms live on Context.

package decimal128

// --- bare methods on Decimal128 ---

// RoundToIntegral returns d rounded to an integral (no fractional digits),
// ties-to-even, quiet (raises no flags).
//
// Note: unlike float64 rounding (ties-away-from-zero), the decimal default is
// ties-to-even, matching IEEE 754 decimal. For another direction use
// Rounding.RoundToIntegral; for flag tracking use Context.RoundToIntegral.
func (d Decimal128) RoundToIntegral() Decimal128 {
	return Decimal128{d128_roundedToIntegral(d.core)}
}

// NextUp returns the least representable value greater than d (IEEE nextUp).
func (d Decimal128) NextUp() Decimal128 { return Decimal128{d128_nextUp_ctxnull(d.core, nil)} }

// NextDown returns the greatest representable value less than d (IEEE nextDown).
func (d Decimal128) NextDown() Decimal128 { return Decimal128{d128_nextDown_ctxnull(d.core, nil)} }

// ScaleB returns d multiplied by 10ⁿ by adjusting the exponent (IEEE scaleB),
// without changing the coefficient when it stays in range.
func (d Decimal128) ScaleB(n int) Decimal128 { return Decimal128{d128_scaleB_ctxnull(d.core, n, nil)} }

// Quantize returns d re-expressed to have the same quantum (exponent) as other,
// rounding the coefficient as needed (IEEE quantize).
func (d Decimal128) Quantize(other Decimal128) Decimal128 {
	return Decimal128{d128_quantize_ctxnull(d.core, other.core, nil)}
}

// SameQuantum reports whether d and other already share the same quantum.
func (d Decimal128) SameQuantum(other Decimal128) bool {
	return d128_isSameQuantum(d.core, other.core)
}

// QuantumExponent returns d's quantum exponent q (its quantum is 1 × 10^q), with
// no flag tracking. A NaN or infinite operand returns an unspecified sentinel;
// use Context.QuantumExponent to capture InvalidOperation.
func (d Decimal128) QuantumExponent() int { return d128_quantumInt_ctxnull(d.core, nil) }

// StripTrailingZeros returns d with all trailing decimal zeros removed from the
// coefficient (exponent raised to match), preserving the numeric value (IEEE 754
// stripTrailingZeros). Zeros collapse to canonical 0E+0.
func (d Decimal128) StripTrailingZeros() Decimal128 {
	return Decimal128{d128_stripTrailingZeros(d.core, 99)}
}

// --- round to integral in a chosen direction ---

// RoundToIntegral returns x rounded to an integral (no fractional digits) in
// this direction. Quiet — raises no flags.
func (r Rounding) RoundToIntegral(x Decimal128) Decimal128 {
	return Decimal128{d128_roundedToIntegralRnd(x.core, r.core())}
}

// --- context-receiver rounding / quantum operations ---

// RoundToIntegral returns x rounded to an integral by this context's direction,
// recording Inexact when rounding discards a fraction.
func (c *Context) RoundToIntegral(x Decimal128) Decimal128 {
	return Decimal128{d128_roundToIntegralSignalInexact(x.core, c.core)}
}

// ScaleB returns x multiplied by 10ⁿ (IEEE scaleB), flag-tracked through this
// context.
func (c *Context) ScaleB(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_scaleB_ctxnull(x.core, n, c.core)}
}

// Quantize returns x re-expressed to share other's quantum (IEEE quantize),
// flag-tracked through this context.
func (c *Context) Quantize(x, other Decimal128) Decimal128 {
	return Decimal128{d128_quantize_ctxnull(x.core, other.core, c.core)}
}

// QuantumExponent returns x's quantum exponent q. A NaN or infinite operand
// raises InvalidOperation into this context and returns an unspecified sentinel.
func (c *Context) QuantumExponent(x Decimal128) int {
	return d128_quantumInt_ctxnull(x.core, c.core)
}

// StripTrailingZeros returns x with trailing zeros stripped (IEEE
// stripTrailingZeros), flag-tracked through this context.
func (c *Context) StripTrailingZeros(x Decimal128) Decimal128 {
	return Decimal128{d128_stripTrailingZeros_ctx(x.core, 99, c.core)}
}
