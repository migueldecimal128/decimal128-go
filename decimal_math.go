// Computational operations beyond the four basic operators, at all three rung
// levels: the bare forms (ties-to-even, quiet) are methods on Decimal128; the
// rounding-direction forms (round in a chosen direction, no flags) are methods
// on Rounding; and the flag-tracking forms are receiver methods on Context
// (round by the context, accumulate exceptions).

package decimal128

// --- bare methods on Decimal128 (ties-to-even, quiet) ---

// Sqrt returns the square root, rounded ties-to-even.
func (d Decimal128) Sqrt() Decimal128 { return Decimal128{d128_sqrt_tte(d.core)} }

// Square returns this value squared (`d * d`), rounded ties-to-even.
func (d Decimal128) Square() Decimal128 { return Decimal128{d128_square_tte(d.core)} }

// FMA returns `d + (lhs * rhs)` computed with a single rounding (ties-to-even).
func (d Decimal128) FMA(a, b Decimal128) Decimal128 {
	return Decimal128{d128_fma_tte(a.core, b.core, d.core)}
}

// Exp returns e raised to this value (`e^x`), rounded ties-to-even. Correctly
// rounded, except that for arguments very close to 1 (for `Exp` and `Exp10`,
// very close to 0) the result can be 1 ulp off, almost always in a directed
// rounding mode (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (d Decimal128) Exp() Decimal128 { return Decimal128{d128_exp_tte(d.core)} }

// Exp10 returns 10 raised to this value (`10^x`), rounded ties-to-even.
// Correctly rounded, except that for arguments very close to 1 (for `Exp` and
// `Exp10`, very close to 0) the result can be 1 ulp off, almost always in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy").
func (d Decimal128) Exp10() Decimal128 { return Decimal128{d128_exp10_tte(d.core)} }

// Ln returns the natural logarithm (`ln x`), rounded ties-to-even. Correctly
// rounded, except that for arguments very close to 1 (for `Exp` and `Exp10`,
// very close to 0) the result can be 1 ulp off, almost always in a directed
// rounding mode (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (d Decimal128) Ln() Decimal128 { return Decimal128{d128_ln_tte(d.core)} }

// Log10 returns the base-10 logarithm (`log10 x`), rounded ties-to-even.
// Correctly rounded, except that for arguments very close to 1 (for `Exp` and
// `Exp10`, very close to 0) the result can be 1 ulp off, almost always in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy").
func (d Decimal128) Log10() Decimal128 { return Decimal128{d128_log10_tte(d.core)} }

// LnP1 returns the natural logarithm of one plus this value (`ln(1 + x)`),
// accurate for x near 0 where computing `Ln` of `1 + x` would lose digits,
// rounded ties-to-even (IEEE 754 `logp1`). Correctly rounded, except that a
// result very close to a rounding boundary can be 1 ulp off in a directed
// rounding mode (CrossPlatformArchitecture.md, "Transcendental accuracy";
// LogP1ExpM1WorkOrder.md).
func (d Decimal128) LnP1() Decimal128 { return Decimal128{d128_lnp1_tte(d.core)} }

// Log2P1 returns the base-2 logarithm of one plus this value (`log2(1 + x)`),
// rounded ties-to-even (IEEE 754 `log2p1`). Correctly rounded, except that a
// result very close to a rounding boundary can be 1 ulp off in a directed
// rounding mode (CrossPlatformArchitecture.md, "Transcendental accuracy";
// LogP1ExpM1WorkOrder.md).
func (d Decimal128) Log2P1() Decimal128 { return Decimal128{d128_log2p1_tte(d.core)} }

// Log10P1 returns the base-10 logarithm of one plus this value (`log10(1 +
// x)`), rounded ties-to-even (IEEE 754 `log10p1`). Correctly rounded, except
// that a result very close to a rounding boundary can be 1 ulp off in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy"; LogP1ExpM1WorkOrder.md).
func (d Decimal128) Log10P1() Decimal128 { return Decimal128{d128_log10p1_tte(d.core)} }

// ExpM1 returns `e` raised to this value, minus one (`e^x - 1`), accurate for x
// near 0 where computing `exp(x) - 1` would cancel, rounded ties-to-even (IEEE
// 754 `expm1`). Correctly rounded, except that a result very close to a
// rounding boundary can be 1 ulp off in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy";
// LogP1ExpM1WorkOrder.md).
func (d Decimal128) ExpM1() Decimal128 { return Decimal128{d128_expm1_tte(d.core)} }

// Exp10M1 returns 10 raised to this value, minus one (`10^x - 1`), rounded
// ties-to-even (IEEE 754 `exp10m1`). Correctly rounded, except that a result
// very close to a rounding boundary can be 1 ulp off in a directed rounding
// mode (CrossPlatformArchitecture.md, "Transcendental accuracy";
// LogP1ExpM1WorkOrder.md).
func (d Decimal128) Exp10M1() Decimal128 { return Decimal128{d128_exp10m1_tte(d.core)} }

// PowInt returns this value raised to the integer power `n` (`d^n`), rounded
// ties-to-even (IEEE 754 `pown`). Correctly rounded, except that for arguments
// very close to 1 (for `Exp` and `Exp10`, very close to 0) the result can be 1
// ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (d Decimal128) PowInt(n int) Decimal128 { return Decimal128{d128_pown_tte(d.core, n)} }

// Pow returns this value raised to the decimal power `exponent` (`d^exponent`),
// rounded ties-to-even (IEEE 754 `pow`). A negative base with a non-integer
// exponent yields NaN (`FlagInvalidOperation`). Correctly rounded, except that
// for arguments very close to 1 (for `Exp` and `Exp10`, very close to 0) the
// result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (d Decimal128) Pow(exponent Decimal128) Decimal128 {
	return Decimal128{d128_pow_tte(d.core, exponent.core)}
}

// Root returns the `n`th root of this value (`d^(1/n)`), rounded ties-to-even
// (IEEE 754 `rootn`). An even-degree root of a negative value yields NaN
// (`FlagInvalidOperation`); `n` must be non-zero. Correctly rounded, except
// that for arguments very close to 1 (for `Exp` and `Exp10`, very close to 0)
// the result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (d Decimal128) Root(n int) Decimal128 { return Decimal128{d128_rootn_tte(d.core, n)} }

// Compound returns `(1 + d)` raised to the integer power `n` (IEEE 754
// `compound`), rounded ties-to-even -- the compound growth of a rate `d` over
// `n` periods. `d` must be >= -1 (else NaN, `FlagInvalidOperation`).
func (d Decimal128) Compound(n int) Decimal128 { return Decimal128{d128_compound_tte(d.core, n)} }

// RemainderTrunc returns the truncating remainder of `d / other` -- quotient
// truncated toward zero (fmod / C `%` semantics).
func (d Decimal128) RemainderTrunc(other Decimal128) Decimal128 {
	return Decimal128{d128_remainderTrunc(d.core, other.core)}
}

// RemainderNear returns the IEEE 754 near remainder of `d / other` -- quotient
// rounded to nearest, ties to even.
func (d Decimal128) RemainderNear(other Decimal128) Decimal128 {
	return Decimal128{d128_remainderNear(d.core, other.core)}
}

// IntegerDivide returns the integer part of `d / other`, truncated toward zero
// (IEEE 754 `divideInteger` / GDAS divide-integer), rounded ties-to-even.
func (d Decimal128) IntegerDivide(other Decimal128) Decimal128 {
	return Decimal128{d128_divideInteger(d.core, other.core)}
}

// Logb returns the adjusted (scientific) exponent of this value as a
// `Decimal128` (IEEE 754 `logB`): `floor(log10|d|)` for finite nonzero values,
// `-inf` for zero, `+inf` for infinity, NaN for NaN.
func (d Decimal128) Logb() Decimal128 { return Decimal128{d128_logB_ctxnull(d.core, nil)} }

// --- context-receiver forms (rounded by the context, flag-tracked) ---

// Sqrt returns the square root of `x`, rounded and flag-tracked through this
// context.
func (c *Context) Sqrt(x Decimal128) Decimal128 {
	return Decimal128{d128_sqrt_ctx(x.core, c.core)}
}

// Square returns `x^2` (`x * x`), rounded and flag-tracked through this
// context.
func (c *Context) Square(x Decimal128) Decimal128 {
	return Decimal128{d128_square_ctx(x.core, c.core)}
}

// FMA returns `a * b + addend` (fused multiply-add) with a single rounding,
// flag-tracked through this context. Matches Python `decimal.Context.fma` / GDAS
// `fusedMultiplyAdd`.
func (c *Context) FMA(a, b, addend Decimal128) Decimal128 {
	return Decimal128{d128_fma_ctx(a.core, b.core, addend.core, c.core)}
}

// Exp returns `e^x`, flag-tracked through this context. Correctly rounded,
// except that for arguments very close to 1 (for `Exp` and `Exp10`, very close
// to 0) the result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (c *Context) Exp(x Decimal128) Decimal128 {
	return Decimal128{d128_exp_ctx(x.core, c.core)}
}

// Exp10 returns `10^x`, flag-tracked through this context. Correctly rounded,
// except that for arguments very close to 1 (for `Exp` and `Exp10`, very close
// to 0) the result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (c *Context) Exp10(x Decimal128) Decimal128 {
	return Decimal128{d128_exp10_ctx(x.core, c.core)}
}

// Ln returns ln x, flag-tracked through this context.
// Correctly rounded, except that for arguments very close to 1 (for `exp`
// and `exp10`, very close to 0) the result can be 1 ulp off, almost always in
// a directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy").
func (c *Context) Ln(x Decimal128) Decimal128 {
	return Decimal128{d128_ln_ctx(x.core, c.core)}
}

// Log10 returns `log10 x`, flag-tracked through this context. Correctly
// rounded, except that for arguments very close to 1 (for `Exp` and `Exp10`,
// very close to 0) the result can be 1 ulp off, almost always in a directed
// rounding mode (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (c *Context) Log10(x Decimal128) Decimal128 {
	return Decimal128{d128_log10_ctx(x.core, c.core)}
}

// LnP1 returns `ln(1 + x)`, flag-tracked through this context. Correctly
// rounded, except that a result very close to a rounding boundary can be 1 ulp
// off in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy"; LogP1ExpM1WorkOrder.md).
func (c *Context) LnP1(x Decimal128) Decimal128 {
	return Decimal128{d128_lnp1_ctx(x.core, c.core)}
}

// Log2P1 returns `log2(1 + x)`, flag-tracked through this context. Correctly
// rounded, except that a result very close to a rounding boundary can be 1 ulp
// off in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy"; LogP1ExpM1WorkOrder.md).
func (c *Context) Log2P1(x Decimal128) Decimal128 {
	return Decimal128{d128_log2p1_ctx(x.core, c.core)}
}

// Log10P1 returns `log10(1 + x)`, flag-tracked through this context. Correctly
// rounded, except that a result very close to a rounding boundary can be 1 ulp
// off in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy"; LogP1ExpM1WorkOrder.md).
func (c *Context) Log10P1(x Decimal128) Decimal128 {
	return Decimal128{d128_log10p1_ctx(x.core, c.core)}
}

// ExpM1 returns `e^x - 1`, flag-tracked through this context. Correctly
// rounded, except that a result very close to a rounding boundary can be 1 ulp
// off in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy"; LogP1ExpM1WorkOrder.md).
func (c *Context) ExpM1(x Decimal128) Decimal128 {
	return Decimal128{d128_expm1_ctx(x.core, c.core)}
}

// Exp10M1 returns `10^x - 1`, flag-tracked through this context. Correctly
// rounded, except that a result very close to a rounding boundary can be 1 ulp
// off in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy"; LogP1ExpM1WorkOrder.md).
func (c *Context) Exp10M1(x Decimal128) Decimal128 {
	return Decimal128{d128_exp10m1_ctx(x.core, c.core)}
}

// PowInt returns `x^n` for an integer power `n` (IEEE 754 `pown`),
// flag-tracked. Correctly rounded, except that for arguments very close to 1
// (for `Exp` and `Exp10`, very close to 0) the result can be 1 ulp off, almost
// always in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy").
func (c *Context) PowInt(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_pown_ctx(x.core, n, c.core)}
}

// Pow returns `x^exponent` for a decimal power `exponent` (IEEE 754 `pow`),
// flag-tracked. Correctly rounded, except that for arguments very close to 1
// (for `Exp` and `Exp10`, very close to 0) the result can be 1 ulp off, almost
// always in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy").
func (c *Context) Pow(x, exponent Decimal128) Decimal128 {
	return Decimal128{d128_pow_ctx(x.core, exponent.core, c.core)}
}

// Root returns the nth root of x (IEEE 754 rootn), flag-tracked.
// Correctly rounded, except that for arguments very close to 1 (for `exp`
// and `exp10`, very close to 0) the result can be 1 ulp off, almost always in
// a directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy").
func (c *Context) Root(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_rootn_ctx(x.core, n, c.core)}
}

// Compound returns `(1 + x)^n` (IEEE 754 `compound`), flag-tracked.
func (c *Context) Compound(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_compound_ctx(x.core, n, c.core)}
}

// RemainderTrunc returns the remainder of `a / b` with the quotient truncated
// toward zero, flag-tracked (e.g. `FlagInvalidOperation` for a zero divisor).
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

// Logb returns the adjusted (scientific) exponent of `x` as a `Decimal128`
// (IEEE 754 `logB`), flag-tracked (`FlagDivisionByZero` for a zero operand,
// `FlagInvalidOperation` for a signaling NaN).
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

// FMA returns `a * b + c` (fused multiply-add) with a single rounding in this
// direction.
func (r Rounding) FMA(a, b, c Decimal128) Decimal128 {
	return Decimal128{d128_fma_rnd(a.core, b.core, c.core, r.core())}
}

// Square returns `x^2` (`x * x`), rounded in this direction.
func (r Rounding) Square(x Decimal128) Decimal128 {
	return Decimal128{d128_square_rnd(x.core, r.core())}
}

// Exp returns `e^x`, rounded in this direction. Correctly rounded, except that
// for arguments very close to 1 (for `Exp` and `Exp10`, very close to 0) the
// result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (r Rounding) Exp(x Decimal128) Decimal128 {
	return Decimal128{d128_exp_rnd(x.core, r.core())}
}

// Exp10 returns `10^x`, rounded in this direction. Correctly rounded, except
// that for arguments very close to 1 (for `Exp` and `Exp10`, very close to 0)
// the result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (r Rounding) Exp10(x Decimal128) Decimal128 {
	return Decimal128{d128_exp10_rnd(x.core, r.core())}
}

// Ln returns ln x, rounded in this direction.
// Correctly rounded, except that for arguments very close to 1 (for `exp`
// and `exp10`, very close to 0) the result can be 1 ulp off, almost always in
// a directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy").
func (r Rounding) Ln(x Decimal128) Decimal128 {
	return Decimal128{d128_ln_rnd(x.core, r.core())}
}

// Log10 returns `log10 x`, rounded in this direction. Correctly rounded, except
// that for arguments very close to 1 (for `Exp` and `Exp10`, very close to 0)
// the result can be 1 ulp off, almost always in a directed rounding mode
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func (r Rounding) Log10(x Decimal128) Decimal128 {
	return Decimal128{d128_log10_rnd(x.core, r.core())}
}

// LnP1 returns `ln(1 + x)`, rounded in this direction. Correctly rounded,
// except that a result very close to a rounding boundary can be 1 ulp off in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy"; LogP1ExpM1WorkOrder.md).
func (r Rounding) LnP1(x Decimal128) Decimal128 {
	return Decimal128{d128_lnp1_rnd(x.core, r.core())}
}

// Log2P1 returns `log2(1 + x)`, rounded in this direction. Correctly rounded,
// except that a result very close to a rounding boundary can be 1 ulp off in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy"; LogP1ExpM1WorkOrder.md).
func (r Rounding) Log2P1(x Decimal128) Decimal128 {
	return Decimal128{d128_log2p1_rnd(x.core, r.core())}
}

// Log10P1 returns `log10(1 + x)`, rounded in this direction. Correctly rounded,
// except that a result very close to a rounding boundary can be 1 ulp off in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy"; LogP1ExpM1WorkOrder.md).
func (r Rounding) Log10P1(x Decimal128) Decimal128 {
	return Decimal128{d128_log10p1_rnd(x.core, r.core())}
}

// ExpM1 returns `e^x - 1`, rounded in this direction. Correctly rounded, except
// that a result very close to a rounding boundary can be 1 ulp off in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy"; LogP1ExpM1WorkOrder.md).
func (r Rounding) ExpM1(x Decimal128) Decimal128 {
	return Decimal128{d128_expm1_rnd(x.core, r.core())}
}

// Exp10M1 returns `10^x - 1`, rounded in this direction. Correctly rounded,
// except that a result very close to a rounding boundary can be 1 ulp off in a
// directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy"; LogP1ExpM1WorkOrder.md).
func (r Rounding) Exp10M1(x Decimal128) Decimal128 {
	return Decimal128{d128_exp10m1_rnd(x.core, r.core())}
}

// PowInt returns `x^n` for an integer power `n` (IEEE 754 `pown`), rounded in
// this direction. Correctly rounded, except that for arguments very close to 1
// (for `Exp` and `Exp10`, very close to 0) the result can be 1 ulp off, almost
// always in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy").
func (r Rounding) PowInt(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_pown_rnd(x.core, n, r.core())}
}

// Pow returns `x^exponent` for a decimal power `exponent` (IEEE 754 `pow`),
// rounded in this direction. Correctly rounded, except that for arguments very
// close to 1 (for `Exp` and `Exp10`, very close to 0) the result can be 1 ulp
// off, almost always in a directed rounding mode (CrossPlatformArchitecture.md,
// "Transcendental accuracy").
func (r Rounding) Pow(x, exponent Decimal128) Decimal128 {
	return Decimal128{d128_pow_rnd(x.core, exponent.core, r.core())}
}

// Root returns the nth root of x (IEEE 754 rootn), rounded in this direction.
// Correctly rounded, except that for arguments very close to 1 (for `exp`
// and `exp10`, very close to 0) the result can be 1 ulp off, almost always in
// a directed rounding mode (CrossPlatformArchitecture.md, "Transcendental
// accuracy").
func (r Rounding) Root(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_rootn_rnd(x.core, n, r.core())}
}

// Compound returns `(1 + x)^n` (IEEE 754 `compound`), rounded in this
// direction.
func (r Rounding) Compound(x Decimal128, n int) Decimal128 {
	return Decimal128{d128_compound_rnd(x.core, n, r.core())}
}
