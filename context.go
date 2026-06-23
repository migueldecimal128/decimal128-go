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

func ctxSignalInexact(ctx *context, d d128) d128 {
	ctx.flags.set(INEXACT)
	return d
}

func ctxSignalInexactOverflow(ctx *context, d d128) d128 {
	ctx.flags.set(OVERFLOW)
	ctx.flags.set(INEXACT)
	return d
}

func ctxSignalInexactUnderflow(ctx *context, d d128) d128 {
	ctx.flags.set(UNDERFLOW)
	ctx.flags.set(INEXACT)
	return d
}

func ctxSignalDivByZero(ctxnull *context, sign bool) d128 {
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

// ctxSignalInvalidOperation sets the invalid-operation flag and returns a quiet
// NaN. (The NaN-operand propagation helpers land in Phase 3.) The cause is
// diagnostic and currently unused.
func ctxSignalInvalidOperation(ctxnull *context, cause invalidCause) d128 {
	_ = cause
	if ctxnull != nil {
		ctxnull.flags.set(INVALID_OPERATION)
	}
	return D128_QNAN0
}
