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

func newContext() *context             { return &context{round: TIES_TO_EVEN, flags: newFlags()} }
func newContextRound(r round) *context { return &context{round: r, flags: newFlags()} }
func ieee754Context() *context         { return newContext() }

func (ctx *context) withRounding(r round) *context {
	return &context{round: r, flags: ctx.flags}
}
func (ctx *context) withRoundingAndNewFlags(r round) *context {
	return &context{round: r, flags: newFlags()}
}

func (ctx *context) isSet(e exception754) bool { return ctx.flags.isSet(e) }
func (ctx *context) isOverflow() bool          { return ctx.isSet(OVERFLOW) }
func (ctx *context) isUnderflow() bool         { return ctx.isSet(UNDERFLOW) }
func (ctx *context) exceptionLetters() string  { return ctx.flags.exceptionLetters() }
func (ctx *context) exceptionFlagsInt() int    { return int(ctx.flags.raw()) }

func isRoundTowardNegative(ctx *context) bool { return ctx.round == TOWARD_NEGATIVE }
func isRoundTowardNegative_quiet() bool       { return false }

func ctxSignalInexact_ctxnull(ctx *context, d d128) d128 {
	ctx.flags.set(INEXACT)
	return d
}

func ctxSignalInexactOverflow_ctxnull(ctx *context, d d128) d128 {
	ctx.flags.set(OVERFLOW)
	ctx.flags.set(INEXACT)
	return d
}

func ctxSignalInexactUnderflow_ctxnull(ctx *context, d d128) d128 {
	ctx.flags.set(UNDERFLOW)
	ctx.flags.set(INEXACT)
	return d
}

func ctxSignalDivByZero_ctxnull(ctxnull *context, sign bool) d128 {
	if ctxnull != nil {
		ctxnull.flags.set(DIVISION_BY_ZERO)
	}
	if sign {
		return D128_NEG_INFINITY
	}
	return D128_POS_INFINITY
}

func signalDivByZero_quiet(resultSign bool) d128 {
	if resultSign {
		return D128_NEG_INFINITY
	}
	return D128_POS_INFINITY
}

func signalInexactOverflow_quiet(dec d128) d128  { return dec }
func signalInexactUnderflow_quiet(dec d128) d128 { return dec }

// ctxSignalInvalidOperation_ctxnull sets the invalid-operation flag and returns a quiet
// NaN (the default result). The cause is diagnostic and currently unused.
func ctxSignalInvalidOperation_ctxnull(ctxnull *context, cause invalidCause) d128 {
	_ = cause
	if ctxnull != nil {
		ctxnull.flags.set(INVALID_OPERATION)
	}
	return D128_QNAN0
}

// ctxSignalInvalidOperationResult is ctxSignalInvalidOperation_ctxnull with an explicit
// result (Swift's defaulted result parameter).
func ctxSignalInvalidOperationResult(ctxnull *context, cause invalidCause, result d128) d128 {
	_ = cause
	if ctxnull != nil {
		ctxnull.flags.set(INVALID_OPERATION)
	}
	return result
}

// ctxSignalSNanOperandFound_ctxnull quiets an sNaN operand and raises invalidOperation.
func ctxSignalSNanOperandFound_ctxnull(ctxnull *context, nan d128) d128 {
	verify("", d128_isNAN(nan))
	quietedNaN := nan
	if d128_isSNAN(nan) {
		quietedNaN = d128_quieted(nan)
	}
	if ctxnull != nil {
		ctxnull.flags.set(INVALID_OPERATION)
	}
	return quietedNaN
}

// ctxNanOperand2_ctxnull selects and propagates the NaN of a two-operand op (sNaN takes
// precedence; raises invalidOperation if any operand was signaling).
func ctxNanOperand2_ctxnull(ctxnull *context, x, y d128) d128 {
	verify("", d128_isNAN(x) || d128_isNAN(y))
	takeY := !d128_isNAN(x) || (d128_isSNAN(y) && !d128_isSNAN(x))
	theNaN := x
	if takeY {
		theNaN = y
	}
	verify("", d128_isNAN(theNaN))
	isSignaling := d128_isSNAN(x) || d128_isSNAN(y)
	if !isSignaling {
		return theNaN
	}
	return ctxSignalSNanOperandFound_ctxnull(ctxnull, theNaN)
}

// ctxNanOperand1_ctxnull propagates a single NaN operand (quieting an sNaN with a signal).
func ctxNanOperand1_ctxnull(ctxnull *context, x d128) d128 {
	verify("", d128_isNAN(x))
	if d128_isQNAN(x) {
		return x
	}
	return ctxSignalSNanOperandFound_ctxnull(ctxnull, x)
}
