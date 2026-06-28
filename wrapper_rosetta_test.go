package decimal128

import (
	"fmt"
	"strconv"
)

// WrapperRosetta: the same conformance corpora, driven entirely through the
// public Decimal128 / Context API, confirming the wrapper forwards to the engine
// correctly. Result equality is checked bitwise via the canonical DPD encoding
// (a public method), and raised exceptions via Context.RaisedFlags — so the
// operation under test only ever touches the public surface.
//
// Operations without a clean public-API form (compare, min/max with flags,
// totalOrder, copy/class/canonical, scaleB's decimal-argument adapter) are
// reported unsupported here; the white-box engine harness covers them.

// decodeOperandWrapper decodes a corpus operand into a Decimal128 through the
// public constructors (DPD/BID interchange for hex, ParseOrNaN for decimals).
func decodeOperandWrapper(s string) Decimal128 {
	if s == "#" {
		return SignalingNaN()
	}
	if len(s) > 0 {
		if s[0] == '#' {
			if u, ok := dpd128ParseDecTestHex(s); ok {
				return DecimalFromDPD(uint64(u.dw1), uint64(u.dw0))
			}
		} else if s[0] == '[' {
			if u, ok := bid128ParseIntelHex(s); ok {
				return DecimalFromBID(uint64(u.dw1), uint64(u.dw0))
			}
		}
	}
	return ParseOrNaN(s)
}

// dpdEqual reports bitwise equality via the canonical DPD interchange pattern.
func dpdEqual(a, b Decimal128) bool {
	ah, al := a.DPD()
	bh, bl := b.DPD()
	return ah == bh && al == bl
}

// wrapperFlagLetters renders RaisedFlags in the corpus xuozi order.
func wrapperFlagLetters(f Flags) string {
	var b []byte
	if f.Has(FlagInexact) {
		b = append(b, 'x')
	}
	if f.Has(FlagUnderflow) {
		b = append(b, 'u')
	}
	if f.Has(FlagOverflow) {
		b = append(b, 'o')
	}
	if f.Has(FlagDivisionByZero) {
		b = append(b, 'z')
	}
	if f.Has(FlagInvalidOperation) {
		b = append(b, 'i')
	}
	return string(b)
}

// tokenToRounding maps a corpus rounding token to a public Rounding; ok is false
// for directions decimal128 cannot represent (up/half_up/...).
func tokenToRounding(tok string) (Rounding, bool) {
	switch tok {
	case "TIES_TO_EVEN":
		return ToNearestEven, true
	case "TIES_TO_AWAY":
		return ToNearestAway, true
	case "TOWARD_ZERO":
		return TowardZero, true
	case "TOWARD_POSITIVE":
		return TowardPositive, true
	case "TOWARD_NEGATIVE":
		return TowardNegative, true
	default:
		return 0, false
	}
}

func wrapCheck(c *rosettaCase, ctx *Context, got Decimal128) (outcome, string) {
	want := decodeOperandWrapper(c.result)
	flags := wrapperFlagLetters(ctx.RaisedFlags())
	if dpdEqual(got, want) && flags == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		got.Format(StyleCoefficientExponent), flags,
		want.Format(StyleCoefficientExponent), c.expectedFlags)
}

func wrapBinaryCtx(c *rosettaCase, op func(*Context, Decimal128, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	return wrapCheck(c, ctx, op(ctx, decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1])))
}

func wrapUnaryCtx(c *rosettaCase, op func(*Context, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	return wrapCheck(c, ctx, op(ctx, decodeOperandWrapper(c.operands[0])))
}

func wrapTernaryCtx(c *rosettaCase, op func(*Context, Decimal128, Decimal128, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 3 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	return wrapCheck(c, ctx, op(ctx, decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1]), decodeOperandWrapper(c.operands[2])))
}

func wrapIntCtx(c *rosettaCase, op func(*Context, Decimal128, int) Decimal128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	n, err := strconv.Atoi(c.operands[1])
	if err != nil {
		return outUnsupported, "non-integer exponent operand"
	}
	ctx := NewContextWithRounding(rnd)
	return wrapCheck(c, ctx, op(ctx, decodeOperandWrapper(c.operands[0]), n))
}

func wrapFormat(c *rosettaCase, style Style) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	got := decodeOperandWrapper(c.operands[0]).Format(style)
	if got == c.result && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, "got " + got + " want " + c.result
}

// wrapBinaryBool runs a two-operand boolean predicate (no flag tracking — a
// case that expects flags is reported unsupported, since the quiet public form
// cannot raise them).
func wrapBinaryBool(c *rosettaCase, pred func(Decimal128, Decimal128) bool) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.expectedFlags != "" {
		return outUnsupported, "quiet predicate cannot track flags"
	}
	got := "0"
	if pred(decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1])) {
		got = "1"
	}
	if got == c.result {
		return outPass, ""
	}
	return outFail, "got " + got + " want " + c.result
}

// wrapDirectedRound runs the quiet directed round-to-integral (no flags).
func wrapDirectedRound(c *rosettaCase, r Rounding) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	if c.expectedFlags != "" {
		return outUnsupported, "quiet round cannot track flags"
	}
	got := r.RoundToIntegral(decodeOperandWrapper(c.operands[0]))
	want := decodeOperandWrapper(c.result)
	if dpdEqual(got, want) {
		return outPass, ""
	}
	return outFail, "got " + got.Format(StyleCoefficientExponent) + " want " + want.Format(StyleCoefficientExponent)
}

func wrapPredicate(c *rosettaCase, pred func(Decimal128) bool) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	got := "0"
	if pred(decodeOperandWrapper(c.operands[0])) {
		got = "1"
	}
	if got == c.result {
		return outPass, ""
	}
	return outFail, "got " + got + " want " + c.result
}

// dispatchWrapper runs a case through the public API.
func dispatchWrapper(c *rosettaCase) (outcome, string) {
	switch c.operation {
	case "d128_add_ctx", "add":
		return wrapBinaryCtx(c, (*Context).Add)
	case "d128_subtract_ctx", "subtract":
		return wrapBinaryCtx(c, (*Context).Sub)
	case "d128_multiply_ctx", "multiply":
		return wrapBinaryCtx(c, (*Context).Mul)
	case "d128_divide_ctx", "divide":
		return wrapBinaryCtx(c, (*Context).Quo)
	case "d128_fma_ctx", "fma":
		return wrapTernaryCtx(c, (*Context).FMA)
	case "d128_sqrt_ctx", "squareroot":
		return wrapUnaryCtx(c, (*Context).Sqrt)
	case "d128_exp_ctx":
		return wrapUnaryCtx(c, (*Context).Exp)
	case "d128_exp10_ctx":
		return wrapUnaryCtx(c, (*Context).Exp10)
	case "d128_ln_ctx":
		return wrapUnaryCtx(c, (*Context).Ln)
	case "d128_log10_ctx":
		return wrapUnaryCtx(c, (*Context).Log10)
	case "d128_pow_ctx":
		return wrapBinaryCtx(c, (*Context).Pow)
	case "d128_pown_ctx":
		return wrapIntCtx(c, (*Context).PowInt)
	case "d128_rootn_ctx":
		return wrapIntCtx(c, (*Context).Root)
	case "d128_compound_ctx":
		return wrapIntCtx(c, (*Context).Compound)
	case "d128_quantize_ctxnull", "quantize":
		return wrapBinaryCtx(c, (*Context).Quantize)
	case "d128_remainder_ctx", "remainder":
		return wrapBinaryCtx(c, (*Context).TruncatingRemainder)
	case "d128_remainderNear_ctx", "remaindernear":
		return wrapBinaryCtx(c, (*Context).Remainder)
	case "d128_divideInteger_ctx", "divideint":
		return wrapBinaryCtx(c, (*Context).IntegerDivide)
	case "d128_roundToIntegralSignalInexact", "tointegralx":
		return wrapUnaryCtx(c, (*Context).RoundToIntegral)
	case "d128_stripTrailingZeros", "reduce":
		return wrapUnaryCtx(c, (*Context).StripTrailingZeros)
	case "d128_toString":
		return wrapFormat(c, StyleAutomatic)
	case "d128_toExpString":
		return wrapFormat(c, StyleExponential)
	case "d128_toEngString":
		return wrapFormat(c, StyleEngineering)
	case "d128_toRawString":
		return wrapFormat(c, StyleCoefficientExponent)
	case "d128_isSigned":
		return wrapPredicate(c, Decimal128.Signbit)
	case "d128_isInf":
		return wrapPredicate(c, Decimal128.IsInf)
	case "d128_isFinite":
		return wrapPredicate(c, Decimal128.IsFinite)
	case "d128_isNaN":
		return wrapPredicate(c, Decimal128.IsNaN)
	case "d128_isSignaling":
		return wrapPredicate(c, Decimal128.IsSignalingNaN)
	case "d128_isQNaN":
		return wrapPredicate(c, func(d Decimal128) bool { return d.IsNaN() && !d.IsSignalingNaN() })
	case "d128_isEqual_ctx":
		return wrapBinaryBool(c, Decimal128.Equal)
	case "d128_isLess_ctx":
		return wrapBinaryBool(c, Decimal128.Less)
	case "d128_totalOrder":
		return wrapBinaryBool(c, Decimal128.IsTotallyOrdered)
	case "d128_round_integral_nearest_even":
		return wrapDirectedRound(c, ToNearestEven)
	case "d128_round_integral_nearest_away":
		return wrapDirectedRound(c, ToNearestAway)
	case "d128_round_integral_zero":
		return wrapDirectedRound(c, TowardZero)
	case "d128_round_integral_positive":
		return wrapDirectedRound(c, TowardPositive)
	case "d128_round_integral_negative":
		return wrapDirectedRound(c, TowardNegative)
	default:
		return outUnsupported, "no public-API arm for " + c.operation
	}
}

func runWrapperRosetta(cases []rosettaCase) rosettaResult {
	var r rosettaResult
	r.total = len(cases)
	for i := range cases {
		out, diag := dispatchWrapper(&cases[i])
		switch out {
		case outPass:
			r.passed++
		case outFail:
			r.failed++
			if len(r.failures) < maxRecordedFailures {
				r.failures = append(r.failures, cases[i].text+" :: "+diag)
			}
		case outUnsupported:
			r.unsupported++
		}
	}
	return r
}
