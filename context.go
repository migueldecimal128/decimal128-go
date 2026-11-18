// The arithmetic context: a rounding direction plus a shared, mutable exception-
// flag sink. A pointer type (Swift's reference Context); a nil *context is the
// "quiet" / no-flag path (Swift's Context?). The NaN/invalid-operation signal
// helpers (which need invalidCause) land with the arithmetic operations in
// Phase 3; the inexact/overflow/underflow/divByZero signals finalize needs are
// here.

package decimal128

type context struct {
	round round
	flags *flags
}

func newContext() *context             { return &context{round: const_ROUND_TIES_TO_EVEN, flags: newFlags()} }
func newContextRound(r round) *context { return &context{round: r, flags: newFlags()} }

// IEEE 754 default context. Equivalent to `Context()`.
func ieee754() *context { return &context{round: const_ROUND_TIES_TO_EVEN, flags: &flags{}} }

// withRounding returns a new context with the given rounding direction.
// The returned context shares the same `flags` reference as `self`.
func (ctx *context) withRounding(newRounding round) *context {
	return &context{round: newRounding, flags: ctx.flags}
}

// withRoundingAndNewFlags returns a new context with the given rounding direction and a
// fresh, empty `Flags` sink.
func (ctx *context) withRoundingAndNewFlags(newRounding round) *context {
	return &context{round: newRounding, flags: newFlags()}
}

// Whether the given exception flag is currently set.
func (ctx *context) isSet(exception exception754) bool { return ctx.flags.isSet(exception) }

// Convenience predicate for the `overflow` flag.
func (ctx *context) isOverflow() bool { return ctx.isSet(const_OVERFLOW) }

// Convenience predicate for the `underflow` flag.
func (ctx *context) isUnderflow() bool { return ctx.isSet(const_UNDERFLOW) }

// Canonical letter rendering of the signaled exceptions (see `Flags`).
func (ctx *context) exceptionLetters() string { return ctx.flags.exceptionLetters() }

// The raised exception flags as the raw `Int` bit word, bit positions per
// `Exception754` (`INVALID_OPERATION=0 ... INEXACT=4`). This is the Int-typed
// boundary the wrapper reads to build its public `DecimalFlags` OptionSet
// (CrossPlatformArchitecture.md Section 6.4); only the low 5 bits are ever set.
func (ctx *context) exceptionFlagsInt() int { return int(ctx.flags.raw()) }

// Whether this context's rounding direction is `towardNegative`.
func isRoundTowardNegative(ctx *context) bool { return ctx.round == const_ROUND_TOWARD_NEGATIVE }

// The context-free counterpart of isRoundTowardNegative, used where no context
// is threaded: always false (the default rounding is ties-to-even).
func isRoundTowardNegative_quiet() bool { return false }

// ctxSignalInexact_ctxnull signals inexact and returns d.
func ctxSignalInexact_ctxnull(ctx *context, d d128) d128 {
	ctx.flags.set(const_INEXACT)
	return d
}

// ctxSignalInexactOverflow_ctxnull signals overflow and inexact, and returns d.
func ctxSignalInexactOverflow_ctxnull(ctx *context, d d128) d128 {
	ctx.flags.set(const_OVERFLOW)
	ctx.flags.set(const_INEXACT)
	return d
}

func ctxSignalInexactUnderflow_ctxnull(ctx *context, d d128) d128 {
	ctx.flags.set(const_UNDERFLOW)
	ctx.flags.set(const_INEXACT)
	return d
}

// ctxSignalDivByZero_ctxnull signals divisionByZero (when ctxnull is non-nil) and returns the infinity of
// the given sign.
func ctxSignalDivByZero_ctxnull(ctxnull *context, sign bool) d128 {
	if ctxnull != nil {
		ctxnull.flags.set(const_DIVISION_BY_ZERO)
	}
	if sign {
		return const_D128_NEG_INFINITY
	}
	return const_D128_POS_INFINITY
}

// ctxSignalInvalidOperation_ctxnull signals invalidOperation (when ctxnull is non-nil) and returns result.
func ctxSignalInvalidOperation_ctxnull(ctxnull *context, cause invalidCause, result d128) d128 {
	_ = cause
	if ctxnull != nil {
		ctxnull.flags.set(const_INVALID_OPERATION)
	}
	return result
}

// ctxSignalSNanOperandFound_ctxnull returns nan quieted and signals invalidOperation (when ctxnull is non-nil).
// The operand must be a NaN; callers pass one only when an operand was
// signaling.
func ctxSignalSNanOperandFound_ctxnull(ctxnull *context, nan d128) d128 {
	verify("", func() bool { return d128_isNAN(nan) })
	quietedNaN := nan
	if d128_isSNAN(nan) {
		quietedNaN = d128_quieted(nan)
	}
	if ctxnull != nil {
		ctxnull.flags.set(const_INVALID_OPERATION)
	}
	return quietedNaN
}

// ctxNanOperand2_ctxnull selects and propagates the NaN of a two-operand operation: x's, unless only
// y is a NaN or only y is signaling. If either operand is signaling, the
// result is quieted and invalidOperation is signaled.
func ctxNanOperand2_ctxnull(ctxnull *context, x, y d128) d128 {
	verify("", func() bool { return d128_isNAN(x) || d128_isNAN(y) })
	takeY := !d128_isNAN(x) || (d128_isSNAN(y) && !d128_isSNAN(x))
	theNaN := x
	if takeY {
		theNaN = y
	}
	verify("", func() bool { return d128_isNAN(theNaN) })
	isSignaling := d128_isSNAN(x) || d128_isSNAN(y)
	if !isSignaling {
		return theNaN
	}
	return ctxSignalSNanOperandFound_ctxnull(ctxnull, theNaN)
}

// ctxNanOperand1_ctxnull propagates a single NaN operand: a quiet NaN unchanged; a signaling NaN
// quieted, with invalidOperation signaled.
func ctxNanOperand1_ctxnull(ctxnull *context, x d128) d128 {
	verify("", func() bool { return d128_isNAN(x) })
	if d128_isQNAN(x) {
		return x
	}
	return ctxSignalSNanOperandFound_ctxnull(ctxnull, x)
}

func signalDivByZero_quiet(resultSign bool) d128 {
	if resultSign {
		return const_D128_NEG_INFINITY
	}
	return const_D128_POS_INFINITY
}

func signalInexactOverflow_quiet(dec d128) d128 { return dec }

func signalInexactUnderflow_quiet(dec d128) d128 { return dec }

func signalInvalidCompare_quiet() {}

func signalSNanOperandFound_quiet(theNaN d128) d128 {
	verify("", func() bool { return d128_isNAN(theNaN) })
	quieted := d128_quieted(theNaN)
	return quieted
}
