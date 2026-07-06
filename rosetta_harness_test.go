package decimal128

// The Rosetta conformance harness (CrossPlatformArchitecture.md §2.4), as
// white-box test-only code so it can dispatch to the unexported engine —
// mirroring the Rust in-crate harness. Phase 0/1 stand up the case model, the
// native-corpus parser, and a zero-arm dispatch; operator arms light up as the
// core lands in later phases. As test scaffolding this code is exempt from the
// §3 constraint regime (switch etc. used freely).

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// source selects the corpus dialect.
type source int

const (
	srcDectest source = iota
	srcFptest
	srcIntel
	srcNative
)

// rosettaCase is one parsed, in-scope conformance case. Operand and result
// strings stay verbatim; decoding is the dispatch layer's job.
type rosettaCase struct {
	src           source
	text          string
	operation     string
	operands      []string
	result        string
	expectedFlags string // canonical xuozi
	rounding      string // canonical rounding token
	traps         string // fptest trap column; empty elsewhere
}

// outcome is the result of running one case.
type outcome int

const (
	outPass outcome = iota
	outFail
	outUnsupported
)

func (o outcome) String() string {
	switch o {
	case outPass:
		return "PASS"
	case outFail:
		return "FAIL"
	case outUnsupported:
		return "UNSUPPORTED"
	default:
		return "?"
	}
}

// flagLettersCanonical returns the xuozi-ordered subset of exception letters in
// s (inexact, underflow, overflow, divByZero, invalid). Unknown chars ignored.
func flagLettersCanonical(s string) string {
	var x, u, o, z, i bool
	for _, ch := range s {
		switch ch {
		case 'x':
			x = true
		case 'u':
			u = true
		case 'o':
			o = true
		case 'z':
			z = true
		case 'i':
			i = true
		}
	}
	var b strings.Builder
	if x {
		b.WriteByte('x')
	}
	if u {
		b.WriteByte('u')
	}
	if o {
		b.WriteByte('o')
	}
	if z {
		b.WriteByte('z')
	}
	if i {
		b.WriteByte('i')
	}
	return b.String()
}

// decodeOperand decodes a corpus operand token into a d128: "#…" is DPD hex,
// "[…]" is Intel BID hex, anything else is a decimal/Infinity/NaN string.
func decodeOperand(s string) d128 {
	if s == "#" {
		// dectest's bare "#" is an operand syntax-error marker: an op receiving it
		// must raise invalidOperation. A signaling NaN produces exactly that.
		return D128_SNAN0
	}
	if len(s) > 0 {
		if s[0] == '#' {
			if u, ok := dpd128ParseDecTestHex(s); ok {
				return d128_from_dpd128(u)
			}
		} else if s[0] == '[' {
			if u, ok := bid128ParseIntelHex(s); ok {
				return d128_from_bid128(u)
			}
		}
	}
	return d128_parseOrNaN(s)
}

// dispatch runs one case against the engine and reports its Outcome. Arms are
// added as operators land; unsupported operations report outUnsupported (not a
// failure). Currently wired: the serde/parse/print/predicate ops.
func dispatch(c *rosettaCase) (outcome, string) {
	switch c.operation {
	case "d128_toString":
		return checkFormat(c, FORMAT_AUTO)
	case "d128_toExpString":
		return checkFormat(c, FORMAT_EXPONENTIAL)
	case "d128_toEngString":
		return checkFormat(c, FORMAT_ENGINEERING)
	case "d128_toRawString":
		return checkFormat(c, FORMAT_COEFFICIENT_QEXP)
	case "d128_isSigned":
		return checkPredicate(c, d128_isNegative)
	case "d128_isInf":
		return checkPredicate(c, d128_isINF)
	case "d128_isFinite":
		return checkPredicate(c, d128_isFinite)
	case "d128_isNaN":
		return checkPredicate(c, d128_isNAN)
	case "d128_isSignaling":
		return checkPredicate(c, d128_isSNAN)
	case "d128_isQNaN":
		return checkPredicate(c, d128_isQNAN)
	case "d128_add_ctx", "add":
		return checkBinaryCtx(c, d128_add_ctx)
	case "d128_subtract_ctx", "subtract":
		return checkBinaryCtx(c, d128_subtract_ctx)
	case "d128_multiply_ctx", "multiply":
		return checkBinaryCtx(c, d128_multiply_ctx)
	case "d128_divide_ctx", "divide":
		return checkBinaryCtx(c, d128_divide_ctx)
	case "d128_compare_ctx", "compare":
		return checkCompareCtx(c, false)
	case "d128_compareSignaling_ctx", "comparesig":
		return checkCompareCtx(c, true)
	case "d128_isEqual_ctx":
		return checkBinaryBoolCtx(c, d128_isEqual_ctx)
	case "d128_isLess_ctx":
		return checkBinaryBoolCtx(c, d128_isLess_ctx)
	case "d128_divideInteger_ctx", "divideint":
		return checkBinaryCtx(c, d128_divideInteger_ctx)
	case "d128_fma_ctx", "fma":
		return checkTernaryCtx(c, d128_fma_ctx)
	case "d128_remainder_ctx", "remainder":
		return checkBinaryCtx(c, d128_remainderTrunc_ctx)
	case "d128_remainderNear_ctx", "remaindernear":
		return checkBinaryCtx(c, d128_remainderNear_ctx)
	case "d128_roundToIntegralSignalInexact", "tointegralx":
		return checkUnaryCtx(c, d128_roundToIntegralSignalInexact)
	case "d128_round_integral_nearest_even":
		return checkUnaryCtx(c, func(x d128, _ *context) d128 { return d128_roundToIntegralTiesToEven(x) })
	case "d128_round_integral_nearest_away":
		return checkUnaryCtx(c, func(x d128, _ *context) d128 { return d128_roundToIntegralTiesToAway(x) })
	case "d128_round_integral_zero":
		return checkUnaryCtx(c, func(x d128, _ *context) d128 { return d128_roundToIntegralTowardZero(x) })
	case "d128_round_integral_positive":
		return checkUnaryCtx(c, func(x d128, _ *context) d128 { return d128_roundToIntegralTowardPositive(x) })
	case "d128_round_integral_negative":
		return checkUnaryCtx(c, func(x d128, _ *context) d128 { return d128_roundToIntegralTowardNegative(x) })
	case "d128_sqrt_ctx", "squareroot":
		return checkUnaryCtx(c, d128_sqrt_ctx)
	case "d128_stripTrailingZeros", "reduce":
		return checkUnaryCtx(c, func(x d128, ctx *context) d128 { return d128_stripTrailingZeros_ctx(x, 6176, ctx) })
	case "d128_quantize_ctxnull", "quantize":
		return checkBinaryCtx(c, d128_quantize_ctxnull)
	case "d128_logB_ctxnull", "logb":
		return checkUnaryCtx(c, d128_logB_ctxnull)
	case "d128_scaleB_ctxnull", "scaleb":
		return checkScaleB(c)
	case "d128_isSameQuantum", "samequantum":
		return checkBinaryBoolCtx(c, func(x, y d128, _ *context) bool { return d128_isSameQuantum(x, y) })
	case "d128_minimum_ctx":
		return checkBinaryCtx(c, d128_minimum_ctx)
	case "d128_maximum_ctx":
		return checkBinaryCtx(c, d128_maximum_ctx)
	case "d128_minimumMagnitude_ctx":
		return checkBinaryCtx(c, d128_minimumMagnitude_ctx)
	case "d128_maximumMagnitude_ctx":
		return checkBinaryCtx(c, d128_maximumMagnitude_ctx)
	case "d128_minimumNumber_ctx", "min":
		return checkMinMaxNumber(c, d128_minimumNumber_ctx)
	case "d128_maximumNumber_ctx", "max":
		return checkMinMaxNumber(c, d128_maximumNumber_ctx)
	case "d128_minimumMagnitudeNumber_ctx", "minmag":
		return checkMinMaxNumber(c, d128_minimumMagnitudeNumber_ctx)
	case "d128_maximumMagnitudeNumber_ctx", "maxmag":
		return checkMinMaxNumber(c, d128_maximumMagnitudeNumber_ctx)
	case "d128_totalOrder":
		return checkBinaryBoolCtx(c, func(x, y d128, _ *context) bool { return d128_isTotallyOrdered(x, y) })
	case "comparetotal":
		return checkCompareInt(c, compareTotalOrder)
	case "comparetotmag":
		return checkCompareInt(c, compareTotalOrderMag)
	case "d128_nextUp_ctxnull", "nextplus":
		return checkUnaryCtx(c, d128_nextUp_ctxnull)
	case "d128_nextDown_ctxnull", "nextminus":
		return checkUnaryCtx(c, d128_nextDown_ctxnull)
	case "d128_copy", "copy":
		return checkUnaryQuiet(c, d128_copy)
	case "d128_abs", "copyabs":
		return checkUnaryQuiet(c, d128_abs)
	case "d128_negate", "copynegate":
		return checkUnaryQuiet(c, d128_negate)
	case "d128_copySign", "copysign":
		return checkBinaryQuiet(c, d128_copySign)
	case "d128_ieeeClass", "class":
		return checkClass(c)
	case "d128_exp_ctx":
		return checkUnaryCtx(c, d128_exp_ctx)
	case "d128_exp10_ctx":
		return checkUnaryCtx(c, d128_exp10_ctx)
	case "d128_ln_ctx":
		return checkUnaryCtx(c, d128_ln_ctx)
	case "d128_log10_ctx":
		return checkUnaryCtx(c, d128_log10_ctx)
	case "d128_pow_ctx":
		return checkBinaryCtx(c, d128_pow_ctx)
	case "d128_pown_ctx":
		return checkIntExpOp(c, d128_pown_ctx)
	case "d128_rootn_ctx":
		return checkIntExpOp(c, d128_rootn_ctx)
	case "d128_compound_ctx":
		return checkIntExpOp(c, d128_compound_ctx)
	case "d128_canonical", "canonical":
		// All UBD encodings are canonical; the DPD decoder normalises any
		// non-canonical declets, so canonical(x) is the identity on the value.
		return checkUnaryQuiet(c, d128_copy)
	case "apply":
		return dispatchApply(c)
	default:
		return outUnsupported, "no dispatch arm for " + c.operation
	}
}

func roundingFromToken(tok string) round {
	switch tok {
	case "TIES_TO_EVEN":
		return TIES_TO_EVEN
	case "TIES_TO_AWAY":
		return TIES_TO_AWAY
	case "TOWARD_ZERO":
		return TOWARD_ZERO
	case "TOWARD_POSITIVE":
		return TOWARD_POSITIVE
	case "TOWARD_NEGATIVE":
		return TOWARD_NEGATIVE
	default:
		return TIES_TO_EVEN
	}
}

// checkBinaryCtx runs a two-operand context op and checks the result value
// (bitwise) and the raised exception flags against the corpus expectation.
func checkBinaryCtx(c *rosettaCase, op func(x, y d128, ctx *context) d128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable in decimal128"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(decodeOperand(c.operands[0]), decodeOperand(c.operands[1]), ctx)
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkScaleB adapts decNumber's scaleb — whose second operand is a decimal
// integer argument — to the IEEE d128_scaleB_ctxnull(x, int). decNumber requires that
// argument to be NaN (propagated), or a finite integer with exponent 0 whose
// magnitude is at most 2*(emax+digits)=12356; anything else signals invalid.
func checkScaleB(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	x := decodeOperand(c.operands[0])
	y := decodeOperand(c.operands[1])
	z := scaleBFromDecimalArg(x, y, ctx)
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

func scaleBFromDecimalArg(x, y d128, ctx *context) d128 {
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctx, x, y)
	}
	if d128_isINF(y) || d128_qExp(y) != 0 {
		return ctxSignalInvalidOperation_ctxnull(ctx, causeOther)
	}
	coeff := d128_u128(y)
	if coeff.dw1 != 0 || coeff.dw0 > 12356 {
		return ctxSignalInvalidOperation_ctxnull(ctx, causeOther)
	}
	n := int(coeff.dw0)
	if d128_signFlag(y) {
		n = -n
	}
	return d128_scaleB_ctxnull(x, n, ctx)
}

// checkMinMaxNumber runs a decNumber min/max (the NaN-as-missing "Number"
// variant) corpus case. decNumber implements 754-2008 minNum, which propagates
// a signaling NaN as a quiet NaN; our 754-2019 minimumNumber instead treats a
// signaling NaN as a missing operand and returns the number (still signalling
// invalid). They agree on quiet NaN, so only the sNaN-versus-number cases
// diverge — those are skipped as unsupported.
func checkMinMaxNumber(c *rosettaCase, op func(x, y d128, ctx *context) d128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	x := decodeOperand(c.operands[0])
	y := decodeOperand(c.operands[1])
	if (d128_isSNAN(x) && !d128_isNAN(y)) || (d128_isSNAN(y) && !d128_isNAN(x)) {
		return outUnsupported, "sNaN handling diverges: 754-2019 minimumNumber vs decNumber minNum"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(x, y, ctx)
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkCompareInt runs a two-operand op returning a memcmp-style -1/0/1 code
// (totalOrder comparison). totalOrder never signals — even on sNaN — so the
// expected flags must be empty.
func checkCompareInt(c *rosettaCase, op func(x, y d128) int) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.operands[0] == "#" || c.operands[1] == "#" {
		// Bare "#" is a conversion-syntax error injected by the test front-end
		// (result NaN + invalidOperation). totalOrder never signals and runs on
		// already-decoded operands, so this case is outside its domain.
		return outUnsupported, "conversion-syntax operand outside totalOrder domain"
	}
	got := strconv.Itoa(op(decodeOperand(c.operands[0]), decodeOperand(c.operands[1])))
	if got == c.result && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q", got, "", c.result, c.expectedFlags)
}

// checkUnaryQuiet runs a non-computational unary op (copy / copyAbs /
// copyNegate) that only moves bits — it never signals, so the result is
// compared bitwise and the expected flags must be empty.
// checkIntExpOp runs a corpus case whose second operand is a plain integer (the
// logBFormat argument of pown / rootn / compound), not a decimal.
func checkIntExpOp(c *rosettaCase, op func(x d128, n int, ctx *context) d128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	n, err := strconv.Atoi(c.operands[1])
	if err != nil {
		return outUnsupported, "non-integer exponent operand"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(decodeOperand(c.operands[0]), n, ctx)
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

func checkUnaryQuiet(c *rosettaCase, op func(x d128) d128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	z := op(decodeOperand(c.operands[0]))
	if d128_bitwiseEQ(z, decodeOperand(c.result)) && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), "",
		d128_format(decodeOperand(c.result), FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkBinaryQuiet runs a non-computational binary op (copySign) — bitwise
// compare, no flags.
func checkBinaryQuiet(c *rosettaCase, op func(x, y d128) d128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	z := op(decodeOperand(c.operands[0]), decodeOperand(c.operands[1]))
	if d128_bitwiseEQ(z, decodeOperand(c.result)) && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), "",
		d128_format(decodeOperand(c.result), FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// decNumber class strings indexed by d128_ieeeClass code (0..9).
var classStrings = [...]string{
	CLASS_SIGNALING_NAN:      "sNaN",
	CLASS_QUIET_NAN:          "NaN",
	CLASS_NEGATIVE_INFINITY:  "-Infinity",
	CLASS_NEGATIVE_NORMAL:    "-Normal",
	CLASS_NEGATIVE_SUBNORMAL: "-Subnormal",
	CLASS_NEGATIVE_ZERO:      "-Zero",
	CLASS_POSITIVE_ZERO:      "+Zero",
	CLASS_POSITIVE_SUBNORMAL: "+Subnormal",
	CLASS_POSITIVE_NORMAL:    "+Normal",
	CLASS_POSITIVE_INFINITY:  "+Infinity",
}

// checkClass runs §5.7.2 class: maps the ieeeClass code to decNumber's class
// string. Non-computational — never signals, so expected flags must be empty.
func checkClass(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	got := classStrings[d128_ieeeClass(decodeOperand(c.operands[0]))]
	if got == c.result && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, "got " + got + " want " + c.result
}

func checkFormat(c *rosettaCase, style int) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	got := d128_format(decodeOperand(c.operands[0]), style)
	if got == c.result && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, "got " + got + " want " + c.result
}

func checkPredicate(c *rosettaCase, pred func(d128) bool) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	got := "0"
	if pred(decodeOperand(c.operands[0])) {
		got = "1"
	}
	if got == c.result {
		return outPass, ""
	}
	return outFail, "got " + got + " want " + c.result
}

// checkTernaryCtx runs a three-operand context op (fma) and checks the result
// (bitwise) and the raised flags.
func checkTernaryCtx(c *rosettaCase, op func(x, y, a d128, ctx *context) d128) (outcome, string) {
	if len(c.operands) != 3 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(decodeOperand(c.operands[0]), decodeOperand(c.operands[1]), decodeOperand(c.operands[2]), ctx)
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkUnaryCtx runs a one-operand context op and checks the result (bitwise)
// and the raised flags.
func checkUnaryCtx(c *rosettaCase, op func(x d128, ctx *context) d128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(decodeOperand(c.operands[0]), ctx)
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkCompareCtx runs the tri-state compare and maps the CMP754 code to the
// corpus's -1/0/1 (or NaN for unordered), checking flags.
func checkCompareCtx(c *rosettaCase, signaling bool) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	x := decodeOperand(c.operands[0])
	y := decodeOperand(c.operands[1])
	var code int
	if signaling {
		code = d128_comparePartialSignaling_ctx(x, y, ctx)
	} else {
		code = d128_comparePartial_ctx(x, y, ctx)
	}
	var match bool
	var got string
	if code == CMP754_UNORDERED {
		// Unordered (a NaN operand): the op result is the propagated NaN value
		// (selected/quieted without re-signaling — compare already raised flags).
		gotNaN := ctxNanOperand2_ctxnull(nil, x, y)
		match = d128_bitwiseEQ(gotNaN, decodeOperand(c.result))
		got = d128_format(gotNaN, FORMAT_AUTO)
	} else {
		n := code - 1 // LT->-1, EQ->0, GT->1
		if len(c.result) > 0 && (c.result[0] == '#' || c.result[0] == '[') {
			// dqCanonical gives the result as a canonical encoding of the
			// -1/0/1 code rather than its decimal text — compare bitwise.
			got = d128_format(d128_fromInt64(n), FORMAT_COEFFICIENT_QEXP)
			match = d128_bitwiseEQ(d128_fromInt64(n), decodeOperand(c.result))
		} else {
			got = strconv.Itoa(n)
			match = got == c.result
		}
	}
	if match && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q", got, ctx.exceptionLetters(), c.result, c.expectedFlags)
}

// checkBinaryBoolCtx runs a two-operand boolean predicate, comparing 0/1.
func checkBinaryBoolCtx(c *rosettaCase, op func(x, y d128, ctx *context) bool) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	got := "0"
	if op(decodeOperand(c.operands[0]), decodeOperand(c.operands[1]), ctx) {
		got = "1"
	}
	if got == c.result && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q", got, ctx.exceptionLetters(), c.result, c.expectedFlags)
}

// dispatchApply handles the dectest serde round-trip op: a "#…" result means
// encode (decimal -> DPD hex); otherwise decode (DPD hex -> decimal string).
func dispatchApply(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	val := decodeOperand(c.operands[0])
	if strings.HasPrefix(c.result, "#") {
		dpd := d128_to_dpd128(val)
		got := fmt.Sprintf("#%016X%016X", uint64(dpd.dw1), uint64(dpd.dw0))
		if strings.EqualFold(got, c.result) {
			return outPass, ""
		}
		return outFail, "encode got " + got + " want " + c.result
	}
	got := d128_format(val, FORMAT_AUTO)
	if got == c.result {
		return outPass, ""
	}
	return outFail, "decode got " + got + " want " + c.result
}

// rosettaResult tallies a harness run.
type rosettaResult struct {
	total       int
	passed      int
	failed      int
	unsupported int
	failures    []string
}

const maxRecordedFailures = 50

// parseNativeLine parses one native-corpus line. ok=false for blank/"//" lines.
// Format: op ROUNDING operand... -> result [flagletters]
func parseNativeLine(line string) (rosettaCase, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return rosettaCase{}, false
	}
	arrow := strings.Index(line, "->")
	if arrow < 0 {
		return rosettaCase{}, false
	}
	left := strings.Fields(line[:arrow])
	right := strings.Fields(line[arrow+len("->"):])
	if len(left) < 2 || len(right) < 1 {
		return rosettaCase{}, false
	}
	c := rosettaCase{
		src:       srcNative,
		text:      line,
		operation: left[0],
		rounding:  left[1],
		operands:  append([]string(nil), left[2:]...),
		result:    right[0],
	}
	if len(right) > 1 {
		c.expectedFlags = flagLettersCanonical(strings.Join(right[1:], ""))
	}
	return c, true
}

// loadNativeFile reads all native-corpus cases from a file.
func loadNativeFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cases []rosettaCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		if c, ok := parseNativeLine(sc.Text()); ok {
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

// parseDectestLine parses an IBM dectest line "id op operand... -> result
// [conditions]". Returns ok=false for comments ("--"), directives (no "->"),
// and blank lines.
func parseDectestLine(line string) (rosettaCase, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "--") {
		return rosettaCase{}, false
	}
	arrow := strings.Index(line, "->")
	if arrow < 0 {
		return rosettaCase{}, false // directive
	}
	left := strings.Fields(line[:arrow])
	right := strings.Fields(line[arrow+len("->"):])
	if len(left) < 3 || len(right) < 1 {
		return rosettaCase{}, false
	}
	operands := make([]string, 0, len(left)-2)
	for _, o := range left[2:] {
		operands = append(operands, unquoteDectest(o))
	}
	c := rosettaCase{
		src:       srcDectest,
		text:      line,
		operation: left[1],
		operands:  operands,
		result:    unquoteDectest(right[0]),
	}
	if len(right) > 1 {
		c.expectedFlags = dectestConditionsToFlags(right[1:])
	}
	return c, true
}

// dectestConditionsToFlags maps dectest condition words to the canonical xuozi
// flag letters (Rounded/Clamped/Subnormal carry no IEEE flag).
func dectestConditionsToFlags(words []string) string {
	var letters strings.Builder
	for _, w := range words {
		switch strings.ToLower(w) {
		case "inexact":
			letters.WriteByte('x')
		case "underflow":
			letters.WriteByte('u')
		case "overflow":
			letters.WriteByte('o')
		case "division_by_zero":
			letters.WriteByte('z')
		case "invalid_operation", "division_undefined", "division_impossible", "conversion_syntax":
			// dectest names the invalid-operation exception several ways.
			letters.WriteByte('i')
		}
	}
	return flagLettersCanonical(letters.String())
}

// parseDectestRounding maps a "rounding:" directive to a rounding token. ok=false
// for non-directive lines.
func parseDectestRounding(line string) (string, bool) {
	l := strings.TrimSpace(strings.ToLower(line))
	if !strings.HasPrefix(l, "rounding:") {
		return "", false
	}
	switch strings.TrimSpace(l[len("rounding:"):]) {
	case "half_even":
		return "TIES_TO_EVEN", true
	case "half_up":
		return "TIES_TO_AWAY", true
	case "down":
		return "TOWARD_ZERO", true
	case "ceiling":
		return "TOWARD_POSITIVE", true
	case "floor":
		return "TOWARD_NEGATIVE", true
	default:
		// "up" and "half_down" are not IEEE-754 decimal modes; decimal128 can't
		// represent them, so those cases are skipped (Unsupported).
		return roundingUnsupported, true
	}
}

const roundingUnsupported = "UNSUPPORTED_ROUNDING"

// unquoteDectest strips the surrounding single quotes dectest uses for string
// results/operands.
func unquoteDectest(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func loadDectestFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []rosettaCase
	rounding := "TIES_TO_EVEN"
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if r, ok := parseDectestRounding(line); ok {
			rounding = r
			continue
		}
		if c, ok := parseDectestLine(line); ok {
			c.rounding = rounding
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

// runRosetta dispatches each case and tallies outcomes. Unsupported cases are
// counted but are not failures.
func runRosetta(cases []rosettaCase) rosettaResult {
	var r rosettaResult
	r.total = len(cases)
	for i := range cases {
		out, diag := dispatch(&cases[i])
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
