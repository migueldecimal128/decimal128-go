package decimal128

import (
	"fmt"
	"strconv"
	"strings"
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
	if s == "#" || s == "S" {
		// dectest's bare "#" and fptest's "S" both denote a signaling NaN operand.
		return SignalingNaN()
	}
	if s == "Q" {
		// fptest's quiet-NaN operand spelling.
		return NaN()
	}
	if len(s) > 0 {
		if s[0] == '#' {
			if u, ok := dpd128ParseDecTestHex(s); ok {
				return FromDPD128(uint64(u.dw1), uint64(u.dw0))
			}
		} else if s[0] == '[' {
			if u, ok := bid128ParseIntelHex(s); ok {
				return FromBID128(uint64(u.dw1), uint64(u.dw0))
			}
		}
	}
	return ParseOrNaN(s)
}

// dpdEqual reports bitwise equality via the canonical DPD interchange pattern.
func dpdEqual(a, b Decimal128) bool {
	ah, al := a.DPD128()
	bh, bl := b.DPD128()
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
		return TiesToEven, true
	case "TIES_TO_AWAY":
		return TiesToAway, true
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
	flags := wrapperFlagLetters(ctx.RaisedFlags())
	if c.src == srcFptest {
		return wrapCheckFptest(c, flags, got)
	}
	want := decodeOperandWrapper(c.result)
	if c.src == srcIntel {
		want = intelNormalizeWant(c, want)
	}
	if dpdEqual(got, want) && flags == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		got.Format(StyleCoefficientExponent), flags,
		want.Format(StyleCoefficientExponent), c.expectedFlags)
}

// intelNormalizeWant applies the Intel result-normalization (NaN cohort choice
// for binary/ternary ops, min/max cohort selection) to the expected value before
// the bitwise compare — reusing the internal-d128 helpers from the core Intel
// machinery. Public Decimal128 operands are unwrapped to d128 via the (same
// package) core field. Mirrors the c.src == srcIntel branches in the core
// harness's checkBinaryCtx / checkTernaryCtx / checkMinMaxNumber.
func intelNormalizeWant(c *rosettaCase, want Decimal128) Decimal128 {
	switch len(c.operands) {
	case 2:
		o1 := decodeOperandWrapper(c.operands[0]).core
		o2 := decodeOperandWrapper(c.operands[1]).core
		w := intelNanBinary(o1, o2, want.core)
		if isMinMaxOp(c.operation) {
			w = intelNumber(o1, o2, w, strings.Contains(c.operation, "max"))
		}
		return Decimal128{w}
	case 3:
		o1 := decodeOperandWrapper(c.operands[0]).core
		o2 := decodeOperandWrapper(c.operands[1]).core
		o3 := decodeOperandWrapper(c.operands[2]).core
		return Decimal128{intelNanTernary(o1, o2, o3, want.core)}
	default:
		return want
	}
}

// isMinMaxOp reports whether op is one of the canonical min/max tokens (the
// NaN-as-missing "Number" family that carries the Intel cohort-selection rule).
func isMinMaxOp(op string) bool {
	switch op {
	case "min", "max", "minmag", "maxmag":
		return true
	default:
		return false
	}
}

// wrapCheckFptest compares against an fptest expected result: a raw cohort form
// (`+coeff e exp`) or a special (Q/S/+inf/-inf/#). Trap-enabled lines are
// skipped; a "#" result is a don't-care value (flags only). Flags are u-stripped
// on both sides (fptest's tininess-before-rounding underflow convention differs
// from ours). Mirrors the Swift wrapper compareFptest.
func wrapCheckFptest(c *rosettaCase, flags string, got Decimal128) (outcome, string) {
	if c.traps != "" {
		return outUnsupported, "fptest trap-enabled"
	}
	if c.result == "#" {
		if fptestFlagsMatch(flags, c.expectedFlags) {
			return outPass, ""
		}
		return outFail, fmt.Sprintf("flags got %q want %q (u-stripped)", flags, c.expectedFlags)
	}
	if fptestValueMatch(c.result, got) && fptestFlagsMatch(flags, c.expectedFlags) {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s→%s flags=%q",
		got.Format(StyleCoefficientExponent), flags,
		c.result, fptestToCoefExp(c.result), c.expectedFlags)
}

// fptestValueMatch reports whether got matches the fptest expected token.
func fptestValueMatch(exp string, got Decimal128) bool {
	switch {
	case strings.HasPrefix(exp, "Q"):
		return got.IsNaN() && !got.IsSignalingNaN()
	case strings.HasPrefix(exp, "S"):
		return got.IsSignalingNaN()
	case exp == "+inf":
		return got.IsInf() && !got.Signbit()
	case exp == "-inf":
		return got.IsInf() && got.Signbit()
	default:
		return fptestToCoefExp(exp) == got.Format(StyleCoefficientExponent)
	}
}

// fptestToCoefExp converts an fptest finite result (`+coeff e exp`) to our
// StyleCoefficientExponent spelling: drop the coefficient '+', uppercase 'e'→'E',
// and sign a non-negative exponent with '+' (which our format always emits;
// fptest omits it).
func fptestToCoefExp(s string) string {
	t := strings.ToUpper(s)
	t = strings.TrimPrefix(t, "+")
	if i := strings.IndexByte(t, 'E'); i >= 0 && i+1 < len(t) {
		if t[i+1] != '-' && t[i+1] != '+' {
			t = t[:i+1] + "+" + t[i+1:]
		}
	}
	return t
}

// fptestFlagsMatch compares raised-exception letters with fptest's expectation,
// stripping the underflow flag from both sides — but our library must not report
// underflow that fptest did not (fptest detects tininess before rounding). Mirrors
// the C# verifyFptestFlags.
func fptestFlagsMatch(obs, exp string) bool {
	if strings.Contains(obs, "u") && !strings.Contains(exp, "u") {
		return false
	}
	return removeFlagChar(obs, 'u') == removeFlagChar(exp, 'u')
}

func removeFlagChar(s string, ch byte) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] != ch {
			b = append(b, s[i])
		}
	}
	return string(b)
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

// The *CtxRnd helpers run a rounding-bearing math op through BOTH the Context
// (_ctx, flag-tracking) and the Rounding (_rnd, flag-free) public surfaces and
// require the two bit-identical (they forward to the same core body). The
// Context result then carries the usual value/flags check. This validates the
// flag-free math surface from the shared corpus with no new vectors.

func rndDivergence(ctxRes, rndRes Decimal128) (outcome, string, bool) {
	if !dpdEqual(ctxRes, rndRes) {
		return outFail, fmt.Sprintf("_rnd vs _ctx diverge: got %s want %s",
			rndRes.Format(StyleCoefficientExponent), ctxRes.Format(StyleCoefficientExponent)), true
	}
	return outPass, "", false
}

func wrapUnaryCtxRnd(c *rosettaCase, ctxOp func(*Context, Decimal128) Decimal128, rndOp func(Rounding, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	x := decodeOperandWrapper(c.operands[0])
	ctxRes := ctxOp(ctx, x)
	if o, diag, bad := rndDivergence(ctxRes, rndOp(rnd, x)); bad {
		return o, diag
	}
	return wrapCheck(c, ctx, ctxRes)
}

func wrapBinaryCtxRnd(c *rosettaCase, ctxOp func(*Context, Decimal128, Decimal128) Decimal128, rndOp func(Rounding, Decimal128, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	a, b := decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1])
	ctxRes := ctxOp(ctx, a, b)
	if o, diag, bad := rndDivergence(ctxRes, rndOp(rnd, a, b)); bad {
		return o, diag
	}
	return wrapCheck(c, ctx, ctxRes)
}

func wrapTernaryCtxRnd(c *rosettaCase, ctxOp func(*Context, Decimal128, Decimal128, Decimal128) Decimal128, rndOp func(Rounding, Decimal128, Decimal128, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 3 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	a, b, cc := decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1]), decodeOperandWrapper(c.operands[2])
	ctxRes := ctxOp(ctx, a, b, cc)
	if o, diag, bad := rndDivergence(ctxRes, rndOp(rnd, a, b, cc)); bad {
		return o, diag
	}
	return wrapCheck(c, ctx, ctxRes)
}

func wrapIntCtxRnd(c *rosettaCase, ctxOp func(*Context, Decimal128, int) Decimal128, rndOp func(Rounding, Decimal128, int) Decimal128) (outcome, string) {
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
	x := decodeOperandWrapper(c.operands[0])
	ctxRes := ctxOp(ctx, x, n)
	if o, diag, bad := rndDivergence(ctxRes, rndOp(rnd, x, n)); bad {
		return o, diag
	}
	return wrapCheck(c, ctx, ctxRes)
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

// wrapQuietUnary runs a non-computational unary op (abs, nextUp/nextDown) that
// only moves bits or quiets a NaN. Because the public form is flag-free, a case
// that expects flags is reported unsupported.
func wrapQuietUnary(c *rosettaCase, op func(Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	if c.expectedFlags != "" {
		return outUnsupported, "quiet op cannot track flags"
	}
	got := op(decodeOperandWrapper(c.operands[0]))
	want := decodeOperandWrapper(c.result)
	if dpdEqual(got, want) {
		return outPass, ""
	}
	return outFail, "got " + got.Format(StyleCoefficientExponent) + " want " + want.Format(StyleCoefficientExponent)
}

// wrapQuietBinary runs a non-computational binary op (copySign) — bitwise
// compare, no flags.
func wrapQuietBinary(c *rosettaCase, op func(Decimal128, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.expectedFlags != "" {
		return outUnsupported, "quiet op cannot track flags"
	}
	got := op(decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1]))
	want := decodeOperandWrapper(c.result)
	if dpdEqual(got, want) {
		return outPass, ""
	}
	return outFail, "got " + got.Format(StyleCoefficientExponent) + " want " + want.Format(StyleCoefficientExponent)
}

// wrapMinMaxNumber runs a min/max "Number" (NaN-as-missing) op. The sNaN-versus-
// number case diverges between 754-2019 minimumNumber and decNumber minNum, so it
// is reported unsupported; otherwise the Intel cohort normalization in wrapCheck
// reconciles the equal-value cohort selection.
func wrapMinMaxNumber(c *rosettaCase, op func(*Context, Decimal128, Decimal128) Decimal128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	x := decodeOperandWrapper(c.operands[0])
	y := decodeOperandWrapper(c.operands[1])
	if (x.IsSignalingNaN() && !y.IsNaN()) || (y.IsSignalingNaN() && !x.IsNaN()) {
		return outUnsupported, "sNaN handling diverges: 754-2019 minimumNumber vs decNumber minNum"
	}
	ctx := NewContextWithRounding(rnd)
	return wrapCheck(c, ctx, op(ctx, x, y))
}

// wrapScaleBIntel runs Intel's scalbn (plain-integer scale argument) through the
// public Context.ScaleB. The dectest scaleb (bounded decimal-argument adapter)
// has no public form, so non-Intel cases are reported unsupported.
func wrapScaleBIntel(c *rosettaCase) (outcome, string) {
	if c.src != srcIntel {
		return outUnsupported, "scaleb decimal-argument adapter has no public form"
	}
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	n, err := strconv.Atoi(c.operands[1])
	if err != nil {
		return outUnsupported, "non-integer scalbn argument"
	}
	ctx := NewContextWithRounding(rnd)
	return wrapCheck(c, ctx, ctx.ScaleB(decodeOperandWrapper(c.operands[0]), n))
}

// wrapFromWord64 runs a decimal64->decimal128 widening decode (FromBID64/
// FromDPD64) through the public API. The operand is a 16-hex 64-bit word (parsed
// op-specifically, NOT via decodeOperandWrapper); the exact-subset decode never
// rounds or signals, so no flags are expected. Compared bitwise via DPD.
func wrapFromWord64(c *rosettaCase, op func(uint64) Decimal128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	if c.expectedFlags != "" {
		return outUnsupported, "quiet decode cannot track flags"
	}
	got := op(parseWord64(c.operands[0]))
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
		return wrapTernaryCtxRnd(c, (*Context).FMA, Rounding.FMA)
	case "d128_sqrt_ctx", "squareroot":
		return wrapUnaryCtxRnd(c, (*Context).Sqrt, Rounding.Sqrt)
	case "d128_exp_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Exp, Rounding.Exp)
	case "d128_exp10_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Exp10, Rounding.Exp10)
	case "d128_ln_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Ln, Rounding.Ln)
	case "d128_log10_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Log10, Rounding.Log10)
	case "d128_pow_ctx":
		return wrapBinaryCtxRnd(c, (*Context).Pow, Rounding.Pow)
	case "d128_pown_ctx":
		return wrapIntCtxRnd(c, (*Context).PowInt, Rounding.PowInt)
	case "d128_rootn_ctx":
		return wrapIntCtxRnd(c, (*Context).Root, Rounding.Root)
	case "d128_compound_ctx":
		return wrapIntCtxRnd(c, (*Context).Compound, Rounding.Compound)
	case "d128_fitsInt64":
		return wrapPredicate(c, Decimal128.FitsInt64)
	case "d128_fitsUInt64":
		return wrapPredicate(c, Decimal128.FitsUInt64)
	case "d128_square_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Square, Rounding.Square)
	case "d128_from_bid64":
		return wrapFromWord64(c, FromBID64)
	case "d128_from_dpd64":
		return wrapFromWord64(c, FromDPD64)
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
	case "d128_isNormal":
		return wrapPredicate(c, Decimal128.IsNormal)
	case "d128_isSubnormal":
		return wrapPredicate(c, Decimal128.IsSubnormal)
	case "d128_isZero":
		return wrapPredicate(c, Decimal128.IsZero)
	case "d128_isEqual_ctx":
		return wrapBinaryBool(c, Decimal128.Equal)
	case "d128_isLess_ctx":
		return wrapBinaryBool(c, Decimal128.Less)
	case "d128_totalOrder":
		return wrapBinaryBool(c, Decimal128.IsTotallyOrdered)
	case "d128_totalOrderMag":
		return wrapBinaryBool(c, Decimal128.IsTotallyOrderedByMagnitude)
	case "samequantum":
		return wrapBinaryBool(c, Decimal128.SameQuantum)
	case "d128_abs", "copyabs":
		return wrapQuietUnary(c, Decimal128.Abs)
	case "nextplus":
		return wrapQuietUnary(c, Decimal128.NextUp)
	case "nextminus":
		return wrapQuietUnary(c, Decimal128.NextDown)
	case "copysign":
		return wrapQuietBinary(c, Decimal128.CopySign)
	case "min":
		return wrapMinMaxNumber(c, (*Context).MinimumNumber)
	case "max":
		return wrapMinMaxNumber(c, (*Context).MaximumNumber)
	case "minmag":
		return wrapMinMaxNumber(c, (*Context).MinimumMagnitudeNumber)
	case "maxmag":
		return wrapMinMaxNumber(c, (*Context).MaximumMagnitudeNumber)
	case "scaleb":
		return wrapScaleBIntel(c)
	case "d128_round_integral_nearest_even":
		return wrapDirectedRound(c, TiesToEven)
	case "d128_round_integral_nearest_away":
		return wrapDirectedRound(c, TiesToAway)
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
		if shouldSkip(&cases[i]) {
			// Per-line skips (Intel cohort/NaN cases) are counted, not run:
			// decrement total so the tally reflects only dispatched cases.
			r.total--
			continue
		}
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
