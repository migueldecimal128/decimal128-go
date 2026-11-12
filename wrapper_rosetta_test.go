package decimal128

import (
	"fmt"
	"math"
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
		got.Text(StyleCoefficientExponent), flags,
		want.Text(StyleCoefficientExponent), c.expectedFlags)
}

// intelNormalizeWant applies the Intel result-normalization (NaN cohort choice
// for binary/ternary ops, min/max cohort selection) to the expected value before
// the bitwise compare, reusing the core runner's intelNanBinary / intelNanTernary
// / intelNumber by reference (WrapperIntelNormalizeWorkOrder.md WN-2). Values
// cross the boundary through the public BID interchange encoding (BID128 /
// FromBID128), never the wrapper's stored rep; only the expected value changes.
// Mirrors the c.src == srcIntel branches in the core harness's checkBinaryCtx /
// checkTernaryCtx / checkMinMaxNumber.
func intelNormalizeWant(c *rosettaCase, want Decimal128) Decimal128 {
	switch len(c.operands) {
	case 2:
		o1 := bidToCore(decodeOperandWrapper(c.operands[0]))
		o2 := bidToCore(decodeOperandWrapper(c.operands[1]))
		w := intelNanBinary(o1, o2, bidToCore(want))
		if isMinMaxOp(c.operation) {
			w = intelNumber(o1, o2, w, strings.Contains(c.operation, "max"))
		}
		return coreToBid(w)
	case 3:
		o1 := bidToCore(decodeOperandWrapper(c.operands[0]))
		o2 := bidToCore(decodeOperandWrapper(c.operands[1]))
		o3 := bidToCore(decodeOperandWrapper(c.operands[2]))
		return coreToBid(intelNanTernary(o1, o2, o3, bidToCore(want)))
	default:
		return want
	}
}

// bidToCore / coreToBid move a value across the wrapper boundary through its
// public BID128 encoding, for the Intel expectation-normalize only.
func bidToCore(v Decimal128) d128 {
	hi, lo := v.BID128()
	return d128_from_bid128(u128{int64(lo), int64(hi)})
}

func coreToBid(x d128) Decimal128 {
	u := d128_to_bid128(x)
	return FromBID128(uint64(u.dw1), uint64(u.dw0))
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
// (`+coeff e exp`) or a special (Q/S/+inf/-inf/#). A "#" result is a don't-care
// value (flags only). A trap-enabled overflow/underflow line is un-wrapped with
// the core runner's fptestTrapUnwrapPublic and asserted by value only
// (WrapperSkipParityWorkOrder.md WS-1/WS-2). Flags are u-stripped on both sides
// (fptest's tininess-before-rounding underflow convention differs from ours).
// Mirrors the Swift wrapper compareFptest.
func wrapCheckFptest(c *rosettaCase, flags string, got Decimal128) (outcome, string) {
	if c.result == "#" {
		if fptestFlagsMatch(flags, c.expectedFlags) {
			return outPass, ""
		}
		return outFail, fmt.Sprintf("flags got %q want %q (u-stripped)", flags, c.expectedFlags)
	}
	valueOK := fptestValueMatch(c.result, got)
	if !valueOK && c.traps != "" {
		if want, err := Parse(fptestToCoefExp(c.result)); err == nil {
			if unwrapped, ok := fptestTrapUnwrapPublic(c, got, want, strings.Contains(flags, "o")); ok {
				if unwrapped == got {
					return outPass, ""
				}
				return outFail, fmt.Sprintf("trap-unwrap mismatch: got %s want %s",
					got.Text(StyleCoefficientExponent), unwrapped.Text(StyleCoefficientExponent))
			}
		}
	}
	if valueOK && fptestFlagsMatch(flags, c.expectedFlags) {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s→%s flags=%q",
		got.Text(StyleCoefficientExponent), flags,
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
		return fptestToCoefExp(exp) == got.Text(StyleCoefficientExponent)
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
			rndRes.Text(StyleCoefficientExponent), ctxRes.Text(StyleCoefficientExponent)), true
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
	got := decodeOperandWrapper(c.operands[0]).Text(style)
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

// wrapPredCmp evaluates an IEEE comparison predicate two independent ways and
// requires they agree: (1) DERIVE it from the three-way ComparePartial/ComparePartialSignaling
// result — the public reduction the family is defined by, and also the flag
// source (the quiet three-way signals on a signaling-NaN operand, the signaling
// three-way on any NaN operand, accumulated into ctx); and (2) VALIDATE the
// ACTUAL public wrapper predicate method (x.NotGreater(y), ctx.GreaterSignaling(
// x,y), …). The quiet methods live on the value type and are flag-free (the
// three-way call is the sole flag source); the signaling methods live on
// *Context and raise into the same ctx (sticky-OR, so re-raising is harmless).
// The method must equal the derivation; both are then checked against the corpus
// (value + flags). A mismatch is a HARD FAILURE.
func wrapPredCmp(c *rosettaCase, signaling bool) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := NewContextWithRounding(rnd)
	x := decodeOperandWrapper(c.operands[0])
	y := decodeOperandWrapper(c.operands[1])
	var cmp Comparison
	if signaling {
		cmp = ctx.ComparePartialSignaling(x, y)
	} else {
		cmp = ctx.ComparePartial(x, y)
	}
	derived := predicateDerived(cmp, c.operation)
	var actual bool
	if signaling {
		actual = signalingPredicate(c.operation, ctx, x, y)
	} else {
		actual = quietPredicate(c.operation, x, y)
	}
	if actual != derived {
		return outFail, fmt.Sprintf("%s: wrapper method = %v but three-way reduction = %v :: %s",
			c.operation, actual, derived, c.text)
	}
	got := "0"
	if actual {
		got = "1"
	}
	flags := wrapperFlagLetters(ctx.RaisedFlags())
	if got == c.result && flags == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q", got, flags, c.result, c.expectedFlags)
}

// predicateDerived reduces a three-way Comparison to each IEEE comparison
// predicate's boolean (CmpLess = ascending, CmpGreater = descending).
func predicateDerived(cmp Comparison, op string) bool {
	switch op {
	case "d128_isEqual_ctx":
		return cmp == CmpEqual
	case "d128_isNotEqual_ctx":
		return cmp != CmpEqual
	case "d128_isLess_ctx", "d128_isLessSignaling_ctx":
		return cmp == CmpLess
	case "d128_isLessEqual_ctx", "d128_isLessEqualSignaling_ctx":
		return cmp == CmpLess || cmp == CmpEqual
	case "d128_isGreater_ctx", "d128_isGreaterSignaling_ctx":
		return cmp == CmpGreater
	case "d128_isGreaterEqual_ctx", "d128_isGreaterEqualSignaling_ctx":
		return cmp == CmpGreater || cmp == CmpEqual
	case "d128_isNotGreater_ctx", "d128_isNotGreaterSignaling_ctx":
		return cmp != CmpGreater
	case "d128_isNotLess_ctx", "d128_isNotLessSignaling_ctx":
		return cmp != CmpLess
	case "d128_isOrdered_ctx":
		return cmp != CmpUnordered
	case "d128_isUnordered_ctx":
		return cmp == CmpUnordered
	case "d128_isLessUnordered_ctx", "d128_isLessUnorderedSignaling_ctx":
		return cmp == CmpLess || cmp == CmpUnordered
	case "d128_isGreaterUnordered_ctx", "d128_isGreaterUnorderedSignaling_ctx":
		return cmp == CmpGreater || cmp == CmpUnordered
	default:
		return false
	}
}

// quietPredicate calls the actual quiet public wrapper predicate method (value
// type receiver, flag-free) for the given op.
func quietPredicate(op string, x, y Decimal128) bool {
	switch op {
	case "d128_isEqual_ctx":
		return x.Equal(y)
	case "d128_isNotEqual_ctx":
		return x.NotEqual(y)
	case "d128_isLess_ctx":
		return x.Less(y)
	case "d128_isLessEqual_ctx":
		return x.LessOrEqual(y)
	case "d128_isGreater_ctx":
		return x.Greater(y)
	case "d128_isGreaterEqual_ctx":
		return x.GreaterOrEqual(y)
	case "d128_isUnordered_ctx":
		return x.IsUnordered(y)
	case "d128_isNotGreater_ctx":
		return x.NotGreater(y)
	case "d128_isNotLess_ctx":
		return x.NotLess(y)
	case "d128_isOrdered_ctx":
		return x.IsOrdered(y)
	case "d128_isGreaterUnordered_ctx":
		return x.GreaterUnordered(y)
	case "d128_isLessUnordered_ctx":
		return x.LessUnordered(y)
	default:
		return false
	}
}

// signalingPredicate calls the actual signaling public wrapper predicate method
// (*Context receiver, raises InvalidOperation into ctx on any NaN) for the op.
func signalingPredicate(op string, ctx *Context, x, y Decimal128) bool {
	switch op {
	case "d128_isLessSignaling_ctx":
		return ctx.LessSignaling(x, y)
	case "d128_isLessEqualSignaling_ctx":
		return ctx.LessOrEqualSignaling(x, y)
	case "d128_isGreaterSignaling_ctx":
		return ctx.GreaterSignaling(x, y)
	case "d128_isGreaterEqualSignaling_ctx":
		return ctx.GreaterOrEqualSignaling(x, y)
	case "d128_isNotGreaterSignaling_ctx":
		return ctx.NotGreaterSignaling(x, y)
	case "d128_isNotLessSignaling_ctx":
		return ctx.NotLessSignaling(x, y)
	case "d128_isGreaterUnorderedSignaling_ctx":
		return ctx.GreaterUnorderedSignaling(x, y)
	case "d128_isLessUnorderedSignaling_ctx":
		return ctx.LessUnorderedSignaling(x, y)
	default:
		return false
	}
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
	return outFail, "got " + got.Text(StyleCoefficientExponent) + " want " + want.Text(StyleCoefficientExponent)
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
	return outFail, "got " + got.Text(StyleCoefficientExponent) + " want " + want.Text(StyleCoefficientExponent)
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
	return outFail, "got " + got.Text(StyleCoefficientExponent) + " want " + want.Text(StyleCoefficientExponent)
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
	// dectest's bare min/max follow decNumber minNum; Intel's are 754-2019
	// minimumNumber, reconciled by intelNormalizeWant in wrapCheck.
	if c.src == srcDectest && ((x.IsSignalingNaN() && !y.IsNaN()) || (y.IsSignalingNaN() && !x.IsNaN())) {
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
	return outFail, "got " + got.Text(StyleCoefficientExponent) + " want " + want.Text(StyleCoefficientExponent)
}

// wrapParse runs the public failable Parse (strict; a value OR a non-nil error).
// Native-only: the operand is the RAW string (NOT decodeOperandWrapper). A `FAIL`
// result asserts Parse returns an error (unconditional pass on error, hard fail
// if it unexpectedly succeeds); otherwise the parsed value is compared bitwise
// (via DPD) against the expected. The strict parse is flag-free.
func wrapParse(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	v, err := Parse(c.operands[0])
	if c.result == "FAIL" {
		if err != nil {
			return outPass, ""
		}
		return outFail, "d128_parse: '" + c.operands[0] + "' should have failed to parse"
	}
	if err != nil {
		return outFail, "d128_parse: '" + c.operands[0] + "' unexpectedly failed to parse"
	}
	want := decodeOperandWrapper(c.result)
	if dpdEqual(v, want) {
		return outPass, ""
	}
	return outFail, "got " + v.Text(StyleCoefficientExponent) + " want " + want.Text(StyleCoefficientExponent)
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

// wrapToUInt64 runs a directed convert-to-uint64 op through the PUBLIC Context
// receiver form. The DIRECTION is fixed by the operation (dir), driven through the
// public surface by setting the context's rounding; the corpus rounding token is
// irrelevant (the directed conversion ignores it). exact selects the
// inexact-signaling form (ToUInt64Exact). The expected value can exceed
// Int64.MaxValue (Intel's 2^63 invalid sentinel), so it is parsed as uint64. On an
// invalid conversion the returned value is implementation-defined -- our wrapper
// returns 0, Intel returns 2^63 -- so once the 'i' flag is raised only the flag is
// checked.
func wrapToUInt64(c *rosettaCase, dir Rounding, exact bool) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	ctx := NewContextWithRounding(dir)
	x := decodeOperandWrapper(c.operands[0])
	var got uint64
	if exact {
		got = ctx.ToUInt64Exact(x)
	} else {
		got = ctx.ToUInt64(x)
	}
	flags := wrapperFlagLetters(ctx.RaisedFlags())
	expected, err := strconv.ParseUint(c.result, 10, 64)
	if err != nil {
		expected = 0
	}
	invalid := strings.Contains(flags, "i")
	valueOK := (got == expected) || (invalid && got == 0)
	if valueOK && flags == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %d flags=%q want %d flags=%q", got, flags, expected, c.expectedFlags)
}

// dispatchWrapper runs a case through the public API.
func dispatchWrapper(c *rosettaCase) (outcome, string) {
	// Intel lines may carry operands the op ignores (R-7); trim them as the core
	// harness does. The operand-count failure itself is the core runner's job.
	fitOperands(c)
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
	case "d128_lnp1_ctx":
		return wrapUnaryCtxRnd(c, (*Context).LnP1, Rounding.LnP1)
	case "d128_log2p1_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Log2P1, Rounding.Log2P1)
	case "d128_log10p1_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Log10P1, Rounding.Log10P1)
	case "d128_expm1_ctx":
		return wrapUnaryCtxRnd(c, (*Context).ExpM1, Rounding.ExpM1)
	case "d128_exp10m1_ctx":
		return wrapUnaryCtxRnd(c, (*Context).Exp10M1, Rounding.Exp10M1)
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
	case "d128_parse":
		return wrapParse(c)
	case "d128_quantize_ctxnull", "quantize":
		return wrapBinaryCtx(c, (*Context).Quantize)
	case "d128_remainderTrunc_ctx", "remainder":
		return wrapBinaryCtx(c, (*Context).RemainderTrunc)
	case "d128_remainderNear_ctx", "remaindernear":
		return wrapBinaryCtx(c, (*Context).RemainderNear)
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
	// IEEE comparison predicates (bool): each is derived from the three-way
	// ComparePartial/ComparePartialSignaling result AND validated against the actual public
	// wrapper predicate method (wrapPredCmp hard-fails on any disagreement).
	case "d128_isEqual_ctx", "d128_isNotEqual_ctx", "d128_isLess_ctx",
		"d128_isLessEqual_ctx", "d128_isGreater_ctx", "d128_isGreaterEqual_ctx",
		"d128_isUnordered_ctx", "d128_isNotGreater_ctx", "d128_isNotLess_ctx",
		"d128_isOrdered_ctx", "d128_isGreaterUnordered_ctx", "d128_isLessUnordered_ctx":
		return wrapPredCmp(c, false)
	case "d128_isLessSignaling_ctx", "d128_isLessEqualSignaling_ctx",
		"d128_isGreaterSignaling_ctx", "d128_isGreaterEqualSignaling_ctx",
		"d128_isNotGreaterSignaling_ctx", "d128_isNotLessSignaling_ctx",
		"d128_isGreaterUnorderedSignaling_ctx", "d128_isLessUnorderedSignaling_ctx":
		return wrapPredCmp(c, true)
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
		return wrapQuietBinary(c, Decimal128.Copysign)
	case "min":
		return wrapMinMaxNumber(c, (*Context).MinimumNumber)
	case "max":
		return wrapMinMaxNumber(c, (*Context).MaximumNumber)
	case "minmag":
		return wrapMinMaxNumber(c, (*Context).MinimumMagnitudeNumber)
	case "maxmag":
		return wrapMinMaxNumber(c, (*Context).MaximumMagnitudeNumber)
	// Native minmax corpus (754-2019 semantics, flag-checked via *Context).
	// The NaN-propagating forms and the NaN-ignoring "Number" forms both signal
	// on sNaN identically; the native vectors are quiet-only so they run through
	// wrapBinaryCtx directly (value + flags).
	case "d128_minimum_ctx":
		return wrapBinaryCtx(c, (*Context).Minimum)
	case "d128_maximum_ctx":
		return wrapBinaryCtx(c, (*Context).Maximum)
	case "d128_minimumMagnitude_ctx":
		return wrapBinaryCtx(c, (*Context).MinimumMagnitude)
	case "d128_maximumMagnitude_ctx":
		return wrapBinaryCtx(c, (*Context).MaximumMagnitude)
	case "d128_minimumNumber_ctx":
		return wrapBinaryCtx(c, (*Context).MinimumNumber)
	case "d128_maximumNumber_ctx":
		return wrapBinaryCtx(c, (*Context).MaximumNumber)
	case "d128_minimumMagnitudeNumber_ctx":
		return wrapBinaryCtx(c, (*Context).MinimumMagnitudeNumber)
	case "d128_maximumMagnitudeNumber_ctx":
		return wrapBinaryCtx(c, (*Context).MaximumMagnitudeNumber)
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
	case "d128_to_uint64_rnint":
		return wrapToUInt64(c, TiesToEven, false)
	case "d128_to_uint64_rninta":
		return wrapToUInt64(c, TiesToAway, false)
	case "d128_to_uint64_ceil":
		return wrapToUInt64(c, TowardPositive, false)
	case "d128_to_uint64_floor":
		return wrapToUInt64(c, TowardNegative, false)
	case "d128_to_uint64_int":
		return wrapToUInt64(c, TowardZero, false)
	case "d128_to_uint64_xrnint":
		return wrapToUInt64(c, TiesToEven, true)
	case "d128_to_uint64_xrninta":
		return wrapToUInt64(c, TiesToAway, true)
	case "d128_to_uint64_xceil":
		return wrapToUInt64(c, TowardPositive, true)
	case "d128_to_uint64_xfloor":
		return wrapToUInt64(c, TowardNegative, true)
	case "d128_to_uint64_xint":
		return wrapToUInt64(c, TowardZero, true)
	case "d128_compare_ctx":
		return wrapComparePartial(c, (*Context).ComparePartial)
	case "d128_compareSignaling_ctx":
		return wrapComparePartial(c, (*Context).ComparePartialSignaling)
	// Intel-only conversions (RosettaCoveragePartitionWorkOrder.md Phase 1).
	// from_uint64 has no public constructor, so it stays unsupported here.
	case "from_int64":
		n, err := strconv.ParseInt(c.operands[0], 10, 64)
		if err != nil {
			return outUnsupported, "non-integer from_int64 operand"
		}
		return wrapCheck(c, NewContext(), FromInt64(n))
	case "parse_ctx":
		// Rounding parse: the context's direction, flags into the context.
		rnd, ok := tokenToRounding(c.rounding)
		if !ok {
			return outUnsupported, "rounding mode not representable"
		}
		ctx := NewContextWithRounding(rnd)
		return wrapCheck(c, ctx, ctx.ParseOrNaN(c.operands[0]))
	case "round_integral_mode":
		// Quiet round-to-integral in the case's direction (flag-free public form).
		rnd, ok := tokenToRounding(c.rounding)
		if !ok {
			return outUnsupported, "rounding mode not representable"
		}
		return wrapDirectedRound(c, rnd)
	case "to_int64_xmode":
		// Convert to Int64, signaling inexact, in the case's direction.
		rnd, ok := tokenToRounding(c.rounding)
		if !ok {
			return outUnsupported, "rounding mode not representable"
		}
		ctx := NewContextWithRounding(rnd)
		got := ctx.ToInt64Exact(decodeOperandWrapper(c.operands[0]))
		flags := wrapperFlagLetters(ctx.RaisedFlags())
		expected, err := strconv.ParseInt(c.result, 10, 64)
		if err != nil {
			expected = math.MinInt64
		}
		if got == expected && flags == c.expectedFlags {
			return outPass, ""
		}
		return outFail, fmt.Sprintf("got %d flags=%q want %d flags=%q", got, flags, expected, c.expectedFlags)
	default:
		return outUnsupported, "no public-API arm for " + c.operation
	}
}

// wrapComparePartial runs a three-way numeric comparison through the public
// Context method and checks the Comparison and the flags. The corpus gives
// -1 / 0 / 1, or for a NaN operand GDAS compare's NaN result ("that NaN"),
// which the wrapper reports as CmpUnordered (Swift WrapperRosetta compareCmp).
func wrapComparePartial(c *rosettaCase, op func(*Context, Decimal128, Decimal128) Comparison) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	rnd, ok := tokenToRounding(c.rounding)
	if !ok {
		return outUnsupported, "rounding mode not representable"
	}
	var want Comparison
	switch {
	case c.result == "-1":
		want = CmpLess
	case c.result == "0":
		want = CmpEqual
	case c.result == "1":
		want = CmpGreater
	case strings.Contains(strings.ToUpper(c.result), "NAN"):
		want = CmpUnordered
	default:
		return outUnsupported, "compare result form " + c.result
	}
	ctx := NewContextWithRounding(rnd)
	got := op(ctx, decodeOperandWrapper(c.operands[0]), decodeOperandWrapper(c.operands[1]))
	flags := wrapperFlagLetters(ctx.RaisedFlags())
	if (got == want) && (flags == c.expectedFlags) {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %v flags=%q want %v flags=%q", got, flags, want, c.expectedFlags)
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
