// Computational operations beyond the four basic operators, at all three rung
// levels: the bare forms (ties-to-even, quiet) are methods on Decimal128; the
// rounding-direction forms (round in a chosen direction, no flags) are methods
// on Rounding; and the flag-tracking forms are receiver methods on Context
// (round by the context, accumulate exceptions).

package decimal128

// --- bare methods on Decimal128 (ties-to-even, quiet) ---

// Sqrt returns the square root of d, rounded ties-to-even.
func (d Decimal128) Sqrt() Decimal128 { return Decimal128{d128_sqrt_tte(d.core)} }

// Square returns d² (d × d), rounded ties-to-even.
func (d Decimal128) Square() Decimal128 { return Decimal128{d128_square_tte(d.core)} }

// FMA returns d + (a × b) computed with a single rounding (ties-to-even)
// — the fused multiply-add with d as the addend.
func (d Decimal128) FMA(a, b Decimal128) Decimal128 {
	return Decimal128{d128_fma_tte(a.core, b.core, d.core)}
}

// Exp returns e raised to d (eˣ), rounded ties-to-even.
func (d Decimal128) Exp() Decimal128 { return Decimal128{d128_exp_tte(d.core)} }

// Exp10 returns 10 raised to d (10ˣ), rounded ties-to-even.
func (d Decimal128) Exp10() Decimal128 { return Decimal128{d128_exp10_tte(d.core)} }

// Ln returns the natural logarithm of d, rounded ties-to-even.
func (d Decimal128) Ln() Decimal128 { return Decimal128{d128_ln_tte(d.core)} }

// Log10 returns the base-10 logarithm of d, rounded ties-to-even.
func (d Decimal128) Log10() Decimal128 { return Decimal128{d128_log10_tte(d.core)} }

// PowInt returns d raised to the integer power n (IEEE 754 pown), rounded
// ties-to-even.
func (d Decimal128) PowInt(n int) Decimal128 { return Decimal128{d128_pown_tte(d.core, n)} }

// Pow returns d raised to the decimal power exponent (IEEE 754 pow), rounded
// ties-to-even. A negative base with a non-integer exponent yields NaN.
func (d Decimal128) Pow(exponent Decimal128) Decimal128 {
	return Decimal128{d128_pow_tte(d.core, exponent.core)}
}

// Root returns the nth root of d (d^(1/n), IEEE 754 rootn), rounded
// ties-to-even. An even-degree root of a negative value yields NaN; n must be
// non-zero.
func (d Decimal128) Root(n int) Decimal128 { return Decimal128{d128_rootn_tte(d.core, n)} }

// Compound returns (1 + d) raised to the integer power n (IEEE 754 compound),
// rounded ties-to-even — compound growth of a rate d over n periods. d must be
// ≥ -1 (else NaN).
func (d Decimal128) Compound(n int) Decimal128 { return Decimal128{d128_compound_tte(d.core, n)} }

// RemainderTrunc returns the remainder of d / other with the quotient
// truncated toward zero.
func (d Decimal128) RemainderTrunc(other Decimal128) Decimal128 {
	return Decimal128{d128_remainderTrunc(d.core, other.core)}
}

// RemainderNear returns the IEEE 754 remainder of d / other (quotient rounded to
// nearest, ties to even).
func (d Decimal128) RemainderNear(other Decimal128) Decimal128 {
	return Decimal128{d128_remainderNear(d.core, other.core)}
}

// IntegerDivide returns the integer part of d / other, truncated toward zero
// (IEEE 754 divideInteger), rounded ties-to-even.
func (d Decimal128) IntegerDivide(other Decimal128) Decimal128 {
	return Decimal128{d128_divideInteger(d.core, other.core)}
}

// Logb returns the adjusted (scientific) exponent of d as a Decimal128 (IEEE 754
// logB): ⌊log₁₀|d|⌋ for finite nonzero values, -∞ for zero, +∞ for infinity.
func (d Decimal128) Logb() Decimal128 { return Decimal128{d128_logB_ctxnull(d.core, nil)} }

// --- context-receiver forms (rounded by the context, flag-tracked) ---

// Sqrt returns the square root of x, flag-tracked through this context.
func (c *Context) Sqrt(x Decimal128) Decimal128 {
	return Decimal128{d128_sqrt_ctx(x.core, c.core)}
}

// Square returns x² (x × x), flag-tracked through this context.
func (c *Context) Square(x Decimal128) Decimal128 {
	return Decimal128{d128_square_ctx(x.core, c.core)}
}

// FMA returns a × b + c (fused multiply-add) with a single rounding,
// flag-tracked through this context.
func (c *Context) FMA(a, b, addend Decimal128) Decimal128 {
	return Decimal128{d128_fma_ctx(a.core, b.core, addend.core, c.core)}
}

// Exp returns eˣ, flag-tracked through this context.
func (c *Context) Exp(x Decimal128) Decimal128 {
	return Decimal128{d128_exp_ctx(x.core, c.core)}
}

// Exp10 returns 10ˣ, flag-tracked through this context.
func (c *Context) Exp10(x Decimal128) Decimal128 {
	return Decimal128{d128_exp10_ctx(x.core, c.core)}
}

// Ln returns ln x, flag-tracked through this context.
func (c *Context) Ln(x Decimal128) Decimal128 {
	return Decimal128{d128_ln_ctx(x.core, c.core)}
}

// Log10 returns log₁₀ x, flag-tracked through this context.
func (c *Context) Log10(x Decimal128) Decimal128 {
	return Decimal128{d128_log10_ctx(x.core, c.core)}
}

// PowInt returns xⁿ for an integer power n (IEEE 754 pown), flag-tracked.
func (c *Context) PowInt(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_pown_ctx(x.core, n, c.core)}
}

// Pow returns x raised to the decimal power exponent (IEEE 754 pow), flag-tracked.
func (c *Context) Pow(x, exponent Decimal128) Decimal128 {
	return Decimal128{d128_pow_ctx(x.core, exponent.core, c.core)}
}

// Root returns the nth root of x (IEEE 754 rootn), flag-tracked.
func (c *Context) Root(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_rootn_ctx(x.core, n, c.core)}
}

// Compound returns (1 + x)ⁿ (IEEE 754 compound), flag-tracked.
func (c *Context) Compound(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_compound_ctx(x.core, n, c.core)}
}

// RemainderTrunc returns the remainder of a / b with the quotient
// truncated toward zero, flag-tracked.
func (c *Context) RemainderTrunc(a, b Decimal128) Decimal128 {
	return Decimal128{d128_remainderTrunc_ctx(a.core, b.core, c.core)}
}

// RemainderNear returns the IEEE 754 remainder of a / b (quotient to nearest, ties
// to even), flag-tracked.
func (c *Context) RemainderNear(a, b Decimal128) Decimal128 {
	return Decimal128{d128_remainderNear_ctx(a.core, b.core, c.core)}
}

// IntegerDivide returns the integer part of a / b, truncated toward zero
// (IEEE 754 divideInteger), flag-tracked.
func (c *Context) IntegerDivide(a, b Decimal128) Decimal128 {
	return Decimal128{d128_divideInteger_ctx(a.core, b.core, c.core)}
}

// Logb returns the adjusted (scientific) exponent of x as a Decimal128 (IEEE 754
// logB), flag-tracked.
func (c *Context) Logb(x Decimal128) Decimal128 {
	return Decimal128{d128_logB_ctxnull(x.core, c.core)}
}

// --- rounding-direction forms (rounded in this direction, no flags) ---
//
// The flag-free counterparts of the Context math above: the rounding direction
// is the receiver, the operand(s) are parameters, and no flags are raised. The
// _rnd core rungs exist for the whole family.

// Sqrt returns the square root of x, rounded in this direction.
func (r Rounding) Sqrt(x Decimal128) Decimal128 {
	return Decimal128{d128_sqrt_rnd(x.core, r.core())}
}

// FMA returns a × b + addend (fused multiply-add) with a single rounding in this
// direction.
func (r Rounding) FMA(a, b, addend Decimal128) Decimal128 {
	return Decimal128{d128_fma_rnd(a.core, b.core, addend.core, r.core())}
}

// Square returns x² (x × x), rounded in this direction.
func (r Rounding) Square(x Decimal128) Decimal128 {
	return Decimal128{d128_square_rnd(x.core, r.core())}
}

// Exp returns eˣ, rounded in this direction.
func (r Rounding) Exp(x Decimal128) Decimal128 {
	return Decimal128{d128_exp_rnd(x.core, r.core())}
}

// Exp10 returns 10ˣ, rounded in this direction.
func (r Rounding) Exp10(x Decimal128) Decimal128 {
	return Decimal128{d128_exp10_rnd(x.core, r.core())}
}

// Ln returns ln x, rounded in this direction.
func (r Rounding) Ln(x Decimal128) Decimal128 {
	return Decimal128{d128_ln_rnd(x.core, r.core())}
}

// Log10 returns log₁₀ x, rounded in this direction.
func (r Rounding) Log10(x Decimal128) Decimal128 {
	return Decimal128{d128_log10_rnd(x.core, r.core())}
}

// PowInt returns xⁿ for an integer power n (IEEE 754 pown), rounded in this
// direction.
func (r Rounding) PowInt(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_pown_rnd(x.core, n, r.core())}
}

// Pow returns x raised to the decimal power exponent (IEEE 754 pow), rounded in
// this direction.
func (r Rounding) Pow(x, exponent Decimal128) Decimal128 {
	return Decimal128{d128_pow_rnd(x.core, exponent.core, r.core())}
}

// Root returns the nth root of x (IEEE 754 rootn), rounded in this direction.
func (r Rounding) Root(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_rootn_rnd(x.core, n, r.core())}
}

// Compound returns (1 + x)ⁿ (IEEE 754 compound), rounded in this direction.
func (r Rounding) Compound(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_compound_rnd(x.core, n, r.core())}
}
