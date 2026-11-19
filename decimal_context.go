// The public IEEE 754 arithmetic context: a rounding direction plus an
// accumulating sink of raised exception flags.
//
// Pass a Context to the flag-tracking operation family to (a) round by the
// context's rounding and (b) record any IEEE exceptions into its flag set.
// Flags accumulate across operations until ClearFlags. A Context is a reference
// type (pointer) and is NOT safe for concurrent use — use one per goroutine;
// Decimal128 values themselves are immutable and freely shareable.

package decimal128

// Context is an IEEE 754 arithmetic context: a rounding direction plus an
// accumulating sink of raised exception flags.
//
// An operation called on a context (a) rounds by the context's `Rounding` and
// (b) records any IEEE exceptions into `RaisedFlags`. Flags accumulate across
// operations until `ClearFlags`.
//
// A `Context` is a mutable reference type and is not thread-safe: it is for use
// within a single task, matching the core's contract. `Decimal128` values
// themselves are immutable and safe to share; only the flag-tracking context is
// not. Use one context per task and read `RaisedFlags` (a value snapshot) when
// done.
type Context struct {
	rounding Rounding
	core     *context // shares its flag sink across rounding changes
}

// NewContext returns a context rounding ties-to-even with an empty flag set.
func NewContext() *Context { return NewContextWithRounding(TiesToEven) }

// NewContextWithRounding creates a context with the given rounding and an empty
// flag sink.
func NewContextWithRounding(r Rounding) *Context {
	return &Context{rounding: r, core: newContextRound(r.core())}
}

// Rounding returns the rounding direction applied to inexact results.
//
// Setting it preserves any flags already accumulated (the underlying sink is
// shared across the rounding change).
func (c *Context) Rounding() Rounding { return c.rounding }

// SetRounding changes the rounding direction, preserving any flags already
// accumulated (the underlying flag sink is shared across the change).
func (c *Context) SetRounding(r Rounding) {
	c.rounding = r
	c.core = c.core.withRounding(r.core())
}

// RaisedFlags returns a snapshot of the exception flags raised so far through
// this context.
func (c *Context) RaisedFlags() Flags { return Flags(c.core.exceptionFlagsInt()) }

// ClearFlags resets the accumulated flags to empty, keeping the rounding
// direction.
func (c *Context) ClearFlags() {
	c.core = c.core.withRoundingAndNewFlags(c.rounding.core())
}

// ParseOrNaN parses a decimal string the IEEE 754 / GDAS way: an over-precision
// literal is rounded to 34 digits by this context's `Rounding` (raising
// `FlagInexact`), and a malformed string yields a quiet NaN with
// `FlagInvalidOperation` raised -- never a trap or an error. Flags accumulate
// into this context.
//
// Contrast `Parse`, which is context-free and reports unparseable input as a
// failure rather than a NaN. Use `ParseOrNaN` when you want IEEE conversion
// semantics and flag visibility.
func (c *Context) ParseOrNaN(text string) Decimal128 {
	if d, ok := d128_parse_ctx(text, c.core); ok {
		return Decimal128{d}
	}
	// d128_parse_ctx already raised InvalidOperation on malformed input; the
	// IEEE result of a failed conversion is a quiet NaN.
	return NaN()
}

// --- arithmetic on the context (modifier-as-receiver) ---

// Add returns a + b, rounded and flag-tracked through this context.
func (c *Context) Add(a, b Decimal128) Decimal128 {
	return Decimal128{d128_add_ctx(a.core, b.core, c.core)}
}

// Sub returns `a - b`, rounded and flag-tracked through this context.
func (c *Context) Sub(a, b Decimal128) Decimal128 {
	return Decimal128{d128_subtract_ctx(a.core, b.core, c.core)}
}

// Mul returns `a * b`, rounded and flag-tracked through this context.
func (c *Context) Mul(a, b Decimal128) Decimal128 {
	return Decimal128{d128_multiply_ctx(a.core, b.core, c.core)}
}

// Quo returns `a / b`, rounded and flag-tracked through this context.
func (c *Context) Quo(a, b Decimal128) Decimal128 {
	return Decimal128{d128_divide_ctx(a.core, b.core, c.core)}
}
