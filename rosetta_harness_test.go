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
	"math"
	"os"
	"path/filepath"
	"slices"
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
	line          int    // 1-based source line (native corpus; 0 elsewhere)
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
		return const_D128_SNAN0
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

// parseWord64 parses a decimal64 operand written as a 16-hex word (optionally
// bracketed, "[XXXXXXXXXXXXXXXX]") to its raw uint64 bits. Used by the
// decimal64->decimal128 widening-decode arms (from_bid64/from_dpd64), whose
// operand is a 64-bit word — NOT the 32-hex bid128 / #dpd128 form decodeOperand
// expects.
func parseWord64(s string) uint64 {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	u, _ := strconv.ParseUint(s, 16, 64)
	return u
}

// dispatch runs one case against the engine and reports its Outcome. Arms are
// added as operators land; unsupported operations report outUnsupported (not a
// failure). Currently wired: the codec/parse/print/predicate ops.
func dispatch(c *rosettaCase) (outcome, string) {
	// R-7: an operand count that doesn't fit the op is a failure naming the line,
	// never a silent drop. Skip-list the line (with a reason) to accept it.
	if !fitOperands(c) {
		n, ok := rosettaArity[c.operation]
		if !ok {
			return outFail, "no dispatch arm for " + c.operation
		}
		return outFail, fmt.Sprintf("operand count %d does not fit op %q (arity %d)", len(c.operands), c.operation, n)
	}
	if c.src == srcFptest {
		return checkFptest(c)
	}
	switch c.operation {
	case "d128_toString":
		return checkFormat(c, const_FORMAT_AUTO)
	case "d128_toExpString":
		return checkFormat(c, const_FORMAT_EXPONENTIAL)
	case "d128_toEngString":
		return checkFormat(c, const_FORMAT_ENGINEERING)
	case "d128_toRawString":
		return checkFormat(c, const_FORMAT_COEFFICIENT_QEXP)
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
	case "d128_isNormal":
		return checkPredicate(c, d128_isNormal)
	case "d128_isSubnormal":
		return checkPredicate(c, d128_isSubnormal)
	case "d128_isZero":
		return checkPredicate(c, d128_isZER)
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
	case "d128_isGreater_ctx":
		return checkBinaryBoolCtx(c, d128_isGreater_ctx)
	case "d128_isGreaterEqual_ctx":
		return checkBinaryBoolCtx(c, d128_isGreaterEqual_ctx)
	case "d128_isGreaterUnordered_ctx":
		return checkBinaryBoolCtx(c, d128_isGreaterUnordered_ctx)
	case "d128_isLessEqual_ctx":
		return checkBinaryBoolCtx(c, d128_isLessEqual_ctx)
	case "d128_isLessUnordered_ctx":
		return checkBinaryBoolCtx(c, d128_isLessUnordered_ctx)
	case "d128_isNotEqual_ctx":
		return checkBinaryBoolCtx(c, d128_isNotEqual_ctx)
	case "d128_isNotGreater_ctx":
		return checkBinaryBoolCtx(c, d128_isNotGreater_ctx)
	case "d128_isNotLess_ctx":
		return checkBinaryBoolCtx(c, d128_isNotLess_ctx)
	case "d128_isOrdered_ctx":
		return checkBinaryBoolCtx(c, d128_isOrdered_ctx)
	case "d128_isUnordered_ctx":
		return checkBinaryBoolCtx(c, d128_isUnordered_ctx)
	case "d128_isEqualSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isEqualSignaling_ctx)
	case "d128_isGreaterSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isGreaterSignaling_ctx)
	case "d128_isGreaterEqualSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isGreaterEqualSignaling_ctx)
	case "d128_isGreaterUnorderedSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isGreaterUnorderedSignaling_ctx)
	case "d128_isLessSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isLessSignaling_ctx)
	case "d128_isLessEqualSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isLessEqualSignaling_ctx)
	case "d128_isLessUnorderedSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isLessUnorderedSignaling_ctx)
	case "d128_isNotEqualSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isNotEqualSignaling_ctx)
	case "d128_isNotGreaterSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isNotGreaterSignaling_ctx)
	case "d128_isNotLessSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isNotLessSignaling_ctx)
	case "d128_isOrderedSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isOrderedSignaling_ctx)
	case "d128_isUnorderedSignaling_ctx":
		return checkBinaryBoolCtx(c, d128_isUnorderedSignaling_ctx)
	case "d128_totalOrderMag":
		return checkBinaryBoolCtx(c, func(x, y d128, _ *context) bool { return d128_isTotallyOrderedMag(x, y) })
	case "d128_quantexp":
		return checkQuantExp(c)
	case "d128_to_int64_rnint":
		return checkToInt64(c, d128_toInt64TiesToEven_ctxnull)
	case "d128_to_int64_rninta":
		return checkToInt64(c, d128_toInt64TiesToAway_ctxnull)
	case "d128_to_int64_ceil":
		return checkToInt64(c, d128_toInt64TowardPositive_ctxnull)
	case "d128_to_int64_floor":
		return checkToInt64(c, d128_toInt64TowardNegative_ctxnull)
	case "d128_to_int64_int":
		return checkToInt64(c, d128_toInt64TowardZero_ctxnull)
	case "d128_to_int64_xrnint":
		return checkToInt64(c, d128_toInt64TiesToEvenSignalInexact_ctxnull)
	case "d128_to_int64_xrninta":
		return checkToInt64(c, d128_toInt64TiesToAwaySignalInexact_ctxnull)
	case "d128_to_int64_xceil":
		return checkToInt64(c, d128_toInt64TowardPositiveSignalInexact_ctxnull)
	case "d128_to_int64_xfloor":
		return checkToInt64(c, d128_toInt64TowardNegativeSignalInexact_ctxnull)
	case "d128_to_int64_xint":
		return checkToInt64(c, d128_toInt64TowardZeroSignalInexact_ctxnull)
	case "d128_to_uint64_rnint":
		return checkToUInt64(c, d128_toUInt64TiesToEven_ctxnull)
	case "d128_to_uint64_rninta":
		return checkToUInt64(c, d128_toUInt64TiesToAway_ctxnull)
	case "d128_to_uint64_ceil":
		return checkToUInt64(c, d128_toUInt64TowardPositive_ctxnull)
	case "d128_to_uint64_floor":
		return checkToUInt64(c, d128_toUInt64TowardNegative_ctxnull)
	case "d128_to_uint64_int":
		return checkToUInt64(c, d128_toUInt64TowardZero_ctxnull)
	case "d128_to_uint64_xrnint":
		return checkToUInt64(c, d128_toUInt64TiesToEvenSignalInexact_ctxnull)
	case "d128_to_uint64_xrninta":
		return checkToUInt64(c, d128_toUInt64TiesToAwaySignalInexact_ctxnull)
	case "d128_to_uint64_xceil":
		return checkToUInt64(c, d128_toUInt64TowardPositiveSignalInexact_ctxnull)
	case "d128_to_uint64_xfloor":
		return checkToUInt64(c, d128_toUInt64TowardNegativeSignalInexact_ctxnull)
	case "d128_to_uint64_xint":
		return checkToUInt64(c, d128_toUInt64TowardZeroSignalInexact_ctxnull)
	case "d128_divideInteger_ctx", "divideint":
		return checkBinaryCtx(c, d128_divideInteger_ctx)
	case "d128_fma_ctx", "fma":
		return checkTernaryCtx(c, d128_fma_ctx)
	case "d128_remainderTrunc_ctx", "remainder":
		return checkBinaryCtx(c, d128_remainderTrunc_ctx)
	case "d128_remainderNear_ctx", "remaindernear":
		return checkBinaryCtx(c, d128_remainderNear_ctx)
	case "d128_roundToIntegralSignalInexact", "tointegralx":
		return checkUnaryCtx(c, d128_roundToIntegralSignalInexact)
	case "d128_round_integral_nearest_even":
		return checkUnaryCtx(c, d128_roundToIntegralTiesToEven_ctx)
	case "d128_round_integral_nearest_away":
		return checkUnaryCtx(c, d128_roundToIntegralTiesToAway_ctx)
	case "d128_round_integral_zero":
		return checkUnaryCtx(c, d128_roundToIntegralTowardZero_ctx)
	case "d128_round_integral_positive":
		return checkUnaryCtx(c, d128_roundToIntegralTowardPositive_ctx)
	case "d128_round_integral_negative":
		return checkUnaryCtx(c, d128_roundToIntegralTowardNegative_ctx)
	case "d128_sqrt_ctx", "squareroot":
		return checkUnaryCtx(c, d128_sqrt_ctx)
	case "d128_stripTrailingZeros", "reduce":
		return checkUnaryCtx(c, func(x d128, ctx *context) d128 { return d128_stripTrailingZeros_ctx(x, 6176, ctx) })
	case "d128_quantize_ctxnull", "quantize":
		return checkBinaryCtx(c, d128_quantize_ctx)
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
		return checkCompareInt(c, d128_compareTotal)
	case "comparetotmag":
		return checkCompareInt(c, d128_compareTotalMag)
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
	case "d128_lnp1_ctx":
		return checkUnaryCtx(c, d128_lnp1_ctx)
	case "d128_log2p1_ctx":
		return checkUnaryCtx(c, d128_log2p1_ctx)
	case "d128_log10p1_ctx":
		return checkUnaryCtx(c, d128_log10p1_ctx)
	case "d128_expm1_ctx":
		return checkUnaryCtx(c, d128_expm1_ctx)
	case "d128_exp10m1_ctx":
		return checkUnaryCtx(c, d128_exp10m1_ctx)
	case "d128_pow_ctx":
		return checkBinaryCtx(c, d128_pow_ctx)
	case "d128_pown_ctx":
		return checkIntExpOp(c, d128_pown_ctx)
	case "d128_rootn_ctx":
		return checkIntExpOp(c, d128_rootn_ctx)
	case "d128_compound_ctx":
		return checkIntExpOp(c, d128_compound_ctx)
	case "d128_fitsInt64":
		return checkPredicate(c, d128_fitsInt64)
	case "d128_fitsUInt64":
		return checkPredicate(c, d128_fitsUInt64)
	case "d128_square_ctx":
		return checkUnaryCtx(c, d128_square_ctx)
	case "d128_from_bid64":
		return checkFromWord64(c, func(u uint64) d128 { return d128_from_bid64(int64(u)) })
	case "d128_from_dpd64":
		return checkFromWord64(c, func(u uint64) d128 { return d128_from_dpd64(int64(u)) })
	case "d128_parse":
		return checkParse(c)
	case "d128_canonical", "canonical":
		// All UBD encodings are canonical; the DPD decoder normalises any
		// non-canonical declets, so canonical(x) is the identity on the value.
		return checkUnaryQuiet(c, d128_copy)
	case "apply":
		return dispatchApply(c)

	// dectest computational abs / minus (value-only, as in Swift: dectest's
	// abs/minus quiet a NaN and signal on sNaN; the sNaN lines are skip-listed).
	case "abs":
		return checkUnaryValueOnly(c, d128_abs)
	case "minus":
		return checkUnaryValueOnly(c, d128_negate)
	// dectest to-string conversions: the operand is parsed in the case's context
	// (rounding to 34 digits and raising flags), then rendered.
	case "tosci":
		return checkDectestFormat(c, const_FORMAT_AUTO)
	case "toeng":
		return checkDectestFormat(c, const_FORMAT_ENGINEERING)

	// Intel-only conversions (RosettaCoveragePartitionWorkOrder.md Phase 1).
	case "from_int64":
		n, err := strconv.ParseInt(c.operands[0], 10, 64)
		if err != nil {
			return outFail, "non-integer from_int64 operand"
		}
		return checkValueFlags(c, d128_fromInt64(int(n)), "")
	case "from_uint64":
		n, err := strconv.ParseUint(c.operands[0], 10, 64)
		if err != nil {
			return outFail, "non-integer from_uint64 operand"
		}
		return checkValueFlags(c, d128_fromUInt64(uint(n)), "")
	case "parse_ctx":
		// Rounding parse: rounds to 34 digits in the case's mode, raises flags. A
		// rejected string yields NaN (+ invalid).
		ctx := newContextRound(roundingFromToken(c.rounding))
		z, ok := d128_parse_ctx(c.operands[0], ctx)
		if !ok {
			z = const_D128_QNAN0
		}
		return checkValueFlags(c, z, ctx.exceptionLetters())
	case "round_integral_mode":
		// Quiet round-to-integral in the case's rounding direction.
		switch c.rounding {
		case "TOWARD_NEGATIVE":
			return checkUnaryCtx(c, d128_roundToIntegralTowardNegative_ctx)
		case "TOWARD_POSITIVE":
			return checkUnaryCtx(c, d128_roundToIntegralTowardPositive_ctx)
		case "TOWARD_ZERO":
			return checkUnaryCtx(c, d128_roundToIntegralTowardZero_ctx)
		case "TIES_TO_AWAY":
			return checkUnaryCtx(c, d128_roundToIntegralTiesToAway_ctx)
		default:
			return checkUnaryCtx(c, d128_roundToIntegralTiesToEven_ctx)
		}
	case "to_int64_xmode":
		// Convert to Int64, signaling inexact, in the case's rounding direction.
		switch c.rounding {
		case "TOWARD_NEGATIVE":
			return checkToInt64(c, d128_toInt64TowardNegativeSignalInexact_ctxnull)
		case "TOWARD_POSITIVE":
			return checkToInt64(c, d128_toInt64TowardPositiveSignalInexact_ctxnull)
		case "TOWARD_ZERO":
			return checkToInt64(c, d128_toInt64TowardZeroSignalInexact_ctxnull)
		case "TIES_TO_AWAY":
			return checkToInt64(c, d128_toInt64TiesToAwaySignalInexact_ctxnull)
		default:
			return checkToInt64(c, d128_toInt64TiesToEvenSignalInexact_ctxnull)
		}
	default:
		// An included op with no dispatch arm is a failure, not a gap (R-7).
		return outFail, "no dispatch arm for " + c.operation
	}
}

// checkValueFlags compares an already-computed result (bitwise) and the raised
// flags against the corpus expectation.
func checkValueFlags(c *rosettaCase, z d128, flags string) (outcome, string) {
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && flags == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), flags,
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkUnaryValueOnly runs a unary op and compares the value only (bitwise).
func checkUnaryValueOnly(c *rosettaCase, op func(x d128) d128) (outcome, string) {
	z := op(decodeOperand(c.operands[0]))
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) {
		return outPass, ""
	}
	return outFail, "got " + d128_format(z, const_FORMAT_COEFFICIENT_QEXP) + " want " + d128_format(expected, const_FORMAT_COEFFICIENT_QEXP)
}

// checkDectestFormat parses the operand in the case's context (round + flags)
// and renders it, comparing the string and the parse flags.
func checkDectestFormat(c *rosettaCase, style int) (outcome, string) {
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	x, ok := d128_parse_ctx(c.operands[0], ctx)
	if !ok {
		x = const_D128_QNAN0
	}
	got := d128_format(x, style)
	if got == c.result && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q", got, ctx.exceptionLetters(), c.result, c.expectedFlags)
}

// MARK: - fptest (core)

const fptestTrapWrapBias = 9216

// checkFptest runs one fptest arithmetic case through the core and reconciles
// it the IEEE-test-suite way: a "#" result is a don't-care value; a trap-enabled
// overflow/underflow result is compared after unwrapping the 754 trap bias; the
// underflow flag is stripped from both sides (fptest detects tininess before
// rounding), but we must not report underflow fptest did not.
func checkFptest(c *rosettaCase) (outcome, string) {
	var op func(x, y d128, ctx *context) d128
	switch c.operation {
	case "add":
		op = d128_add_ctx
	case "subtract":
		op = d128_subtract_ctx
	case "multiply":
		op = d128_multiply_ctx
	case "divide":
		op = d128_divide_ctx
	default:
		return outFail, "no fptest dispatch arm for " + c.operation
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(decodeNativeResult(c.operands[0]), decodeNativeResult(c.operands[1]), ctx)
	flags := ctx.exceptionLetters()
	if c.result != "#" {
		expected := decodeNativeResult(c.result)
		if !d128_bitwiseEQ(z, expected) {
			delta := fptestTrapWrapDelta(c, z, strings.Contains(flags, "o"))
			if delta == 0 {
				return outFail, "got " + d128_format(z, const_FORMAT_COEFFICIENT_QEXP) + " want " + d128_format(expected, const_FORMAT_COEFFICIENT_QEXP)
			}
			unwrapped := d128_scaleB_ctxnull(expected, delta, newContextRound(roundingFromToken(c.rounding)))
			if !d128_bitwiseEQ(unwrapped, z) {
				return outFail, "trap-unwrap mismatch: got " + d128_format(z, const_FORMAT_COEFFICIENT_QEXP)
			}
			return outPass, ""
		}
	}
	if !fptestFlagsMatch(flags, c.expectedFlags) {
		return outFail, fmt.Sprintf("flags got %q want %q (u-stripped)", flags, c.expectedFlags)
	}
	return outPass, ""
}

// fptestTrapWrapDelta is the scaleB exponent delta to un-wrap a trap-enabled
// fptest vector, or 0 if it is not one.
func fptestTrapWrapDelta(c *rosettaCase, z d128, overflowed bool) int {
	if strings.Contains(c.traps, "o") && strings.Contains(c.expectedFlags, "o") && overflowed {
		return fptestTrapWrapBias
	}
	if strings.Contains(c.traps, "u") && strings.Contains(c.expectedFlags, "u") && d128_isFinite(z) && !d128_isNormal(z) {
		return -fptestTrapWrapBias
	}
	return 0
}

// fptestTrapUnwrapPublic is the fptest trap un-wrap for the public-API (wrapper)
// runner (WrapperSkipParityWorkOrder.md WS-2): the same delta and scaleB as
// checkFptest, with values crossing as public BID128 patterns. It returns the
// un-wrapped expected value and true, or false if the line is not a trapped
// overflow/underflow (then the value compares as written). On true the caller
// asserts the value only: trapped flags follow 1985 conventions.
func fptestTrapUnwrapPublic(c *rosettaCase, observed, expected Decimal128, overflowed bool) (Decimal128, bool) {
	delta := fptestTrapWrapDelta(c, bidToCore(observed), overflowed)
	if delta == 0 {
		return Decimal128{}, false
	}
	unwrapped := d128_scaleB_ctxnull(bidToCore(expected), delta, newContextRound(roundingFromToken(c.rounding)))
	return coreToBid(unwrapped), true
}

// dectestSkips are per-line dectest cases skipped (counted, not failed), with
// reasons. The lines come from Swift's RosettaDectestSkips (the ones this
// harness reaches; Go's scaleb decimal-argument adapter passes the scaleb lines
// Swift skips).
// Matched by trimmed line text.
var dectestSkips = map[string]bool{
	// dqabs abs: dectest abs quiets a NaN and signals on sNaN (and reads "#" as a
	// syntax error); IEEE abs is a quiet bit operation. Value-only arm, so these
	// sNaN / NaN-sign lines are out of its domain.
	"dqabs523 abs  sNaN    ->  NaN   Invalid_operation": true,
	"dqabs525 abs  sNaN33  ->  NaN33 Invalid_operation": true,
	"dqabs526 abs  -NaN22  -> -NaN22":                   true,
	"dqabs527 abs -sNaN33  -> -NaN33 Invalid_operation": true,
	"dqabs900 abs  # -> NaN Invalid_operation":          true,

	// dqbas toSci: strings this parser accepts as NaN that decNumber rejects as
	// Conversion_syntax ("qNaN", over-long payloads).
	"dqbas561 toSci \"qNaN\"            -> NaN Conversion_syntax":                      true,
	"dqbas725 toSci 'NaN1234567890123456781234567890123456' -> NaN Conversion_syntax":  true,
	"dqbas745 toSci 'sNaN1234567890123456787234561234567890' -> NaN Conversion_syntax": true,

	// dqmax / dqmin / dqmxg / dqmng: decNumber min/max is 754-2008 minNum, which
	// returns NaN for (sNaN, number); IEEE 754-2019 minimumNumber returns the number
	// (still signalling invalid), as the library does.
	"dqmax161 max  sNaN -Inf   ->  NaN  Invalid_operation":        true,
	"dqmax162 max  sNaN -1000  ->  NaN  Invalid_operation":        true,
	"dqmax163 max  sNaN -1     ->  NaN  Invalid_operation":        true,
	"dqmax164 max  sNaN -0     ->  NaN  Invalid_operation":        true,
	"dqmax165 max  sNaN  0     ->  NaN  Invalid_operation":        true,
	"dqmax166 max  sNaN  1     ->  NaN  Invalid_operation":        true,
	"dqmax167 max  sNaN  1000  ->  NaN  Invalid_operation":        true,
	"dqmax171 max -Inf  sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax172 max -1000 sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax173 max -1    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax174 max -0    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax175 max  0    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax176 max  1    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax177 max  1000 sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax178 max  Inf  sNaN   ->  NaN  Invalid_operation":        true,
	"dqmax191 max  sNaN99 -Inf    ->  NaN99 Invalid_operation":    true,
	"dqmax192 max  sNaN98 -1      ->  NaN98 Invalid_operation":    true,
	"dqmax196 max -Inf    sNaN92  ->  NaN92 Invalid_operation":    true,
	"dqmax197 max  0      sNaN91  ->  NaN91 Invalid_operation":    true,
	"dqmax198 max  Inf   -sNaN90  -> -NaN90 Invalid_operation":    true,
	"dqmin161 min  sNaN -Inf   ->  NaN  Invalid_operation":        true,
	"dqmin162 min  sNaN -1000  ->  NaN  Invalid_operation":        true,
	"dqmin163 min  sNaN -1     ->  NaN  Invalid_operation":        true,
	"dqmin164 min  sNaN -0     ->  NaN  Invalid_operation":        true,
	"dqmin165 min -sNaN  0     -> -NaN  Invalid_operation":        true,
	"dqmin166 min -sNaN  1     -> -NaN  Invalid_operation":        true,
	"dqmin167 min  sNaN  1000  ->  NaN  Invalid_operation":        true,
	"dqmin171 min -Inf  sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin172 min -1000 sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin173 min -1    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin174 min -0    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin175 min  0    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin176 min  1    sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin177 min  1000 sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin178 min  Inf  sNaN   ->  NaN  Invalid_operation":        true,
	"dqmin191 min  sNaN99 -Inf    ->  NaN99 Invalid_operation":    true,
	"dqmin192 min  sNaN98 -11     ->  NaN98 Invalid_operation":    true,
	"dqmin196 min -Inf    sNaN92  ->  NaN92 Invalid_operation":    true,
	"dqmin197 min  088    sNaN91  ->  NaN91 Invalid_operation":    true,
	"dqmin198 min  Inf   -sNaN90  -> -NaN90 Invalid_operation":    true,
	"dqmng161 minmag  sNaN -Inf   ->  NaN  Invalid_operation":     true,
	"dqmng162 minmag  sNaN -1000  ->  NaN  Invalid_operation":     true,
	"dqmng163 minmag  sNaN -1     ->  NaN  Invalid_operation":     true,
	"dqmng164 minmag  sNaN -0     ->  NaN  Invalid_operation":     true,
	"dqmng165 minmag -sNaN  0     -> -NaN  Invalid_operation":     true,
	"dqmng166 minmag -sNaN  1     -> -NaN  Invalid_operation":     true,
	"dqmng167 minmag  sNaN  1000  ->  NaN  Invalid_operation":     true,
	"dqmng171 minmag -Inf  sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng172 minmag -1000 sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng173 minmag -1    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng174 minmag -0    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng175 minmag  0    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng176 minmag  1    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng177 minmag  1000 sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng178 minmag  Inf  sNaN   ->  NaN  Invalid_operation":     true,
	"dqmng191 minmag  sNaN99 -Inf    ->  NaN99 Invalid_operation": true,
	"dqmng192 minmag  sNaN98 -11     ->  NaN98 Invalid_operation": true,
	"dqmng196 minmag -Inf    sNaN92  ->  NaN92 Invalid_operation": true,
	"dqmng197 minmag  088    sNaN91  ->  NaN91 Invalid_operation": true,
	"dqmng198 minmag  Inf   -sNaN90  -> -NaN90 Invalid_operation": true,
	"dqmxg161 maxmag  sNaN -Inf   ->  NaN  Invalid_operation":     true,
	"dqmxg162 maxmag  sNaN -1000  ->  NaN  Invalid_operation":     true,
	"dqmxg163 maxmag  sNaN -1     ->  NaN  Invalid_operation":     true,
	"dqmxg164 maxmag  sNaN -0     ->  NaN  Invalid_operation":     true,
	"dqmxg165 maxmag  sNaN  0     ->  NaN  Invalid_operation":     true,
	"dqmxg166 maxmag  sNaN  1     ->  NaN  Invalid_operation":     true,
	"dqmxg167 maxmag  sNaN  1000  ->  NaN  Invalid_operation":     true,
	"dqmxg171 maxmag -Inf  sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg172 maxmag -1000 sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg173 maxmag -1    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg174 maxmag -0    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg175 maxmag  0    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg176 maxmag  1    sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg177 maxmag  1000 sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg178 maxmag  Inf  sNaN   ->  NaN  Invalid_operation":     true,
	"dqmxg191 maxmag  sNaN99 -Inf    ->  NaN99 Invalid_operation": true,
	"dqmxg192 maxmag  sNaN98 -1      ->  NaN98 Invalid_operation": true,
	"dqmxg196 maxmag -Inf    sNaN92  ->  NaN92 Invalid_operation": true,
	"dqmxg197 maxmag  0      sNaN91  ->  NaN91 Invalid_operation": true,
	"dqmxg198 maxmag  Inf   -sNaN90  -> -NaN90 Invalid_operation": true,

	// dqmns minus: dectest minus is 0 - x (quiets NaN, signals sNaN, keeps +0);
	// IEEE negate flips the sign bit.
	"dqmns021 minus         NaN  -> NaN":                       true,
	"dqmns022 minus        -NaN  -> -NaN":                      true,
	"dqmns023 minus        sNaN  -> NaN  Invalid_operation":    true,
	"dqmns024 minus       -sNaN  -> -NaN Invalid_operation":    true,
	"dqmns031 minus       NaN13  -> NaN13":                     true,
	"dqmns032 minus      -NaN13  -> -NaN13":                    true,
	"dqmns033 minus      sNaN13  -> NaN13   Invalid_operation": true,
	"dqmns034 minus     -sNaN13  -> -NaN13  Invalid_operation": true,
	"dqmns035 minus       NaN70  -> NaN70":                     true,
	"dqmns036 minus      -NaN70  -> -NaN70":                    true,
	"dqmns037 minus      sNaN101 -> NaN101  Invalid_operation": true,
	"dqmns038 minus     -sNaN101 -> -NaN101 Invalid_operation": true,
	"dqmns111 minus          0   -> 0":                         true,
	"dqmns113 minus       0E+4   -> 0E+4":                      true,
	"dqmns115 minus     0.0000   -> 0.0000":                    true,
	"dqmns117 minus      0E-141  -> 0E-141":                    true,

	// dqrem / dqrmn remainder, remaindernear: IEEE 754-2019 §5.3.1 exact; GDAS signals
	// Division_impossible (RemainderExactWorkOrder.md RX-1/RX-2).
	"dqrem1051 remainder  1e+277  1e-311 ->  NaN Division_impossible":                                true,
	"dqrem1052 remainder  1e+277 -1e-311 ->  NaN Division_impossible":                                true,
	"dqrem1053 remainder -1e+277  1e-311 ->  NaN Division_impossible":                                true,
	"dqrem1054 remainder -1e+277 -1e-311 ->  NaN Division_impossible":                                true,
	"dqrem421 remainder   1E+6144        1  ->   NaN Division_impossible":                            true,
	"dqrem772  remainder  1234568888888887777777777890123456   0.1  ->  NaN Division_impossible":     true,
	"dqrem773  remainder  1234568888888887777777777890123456   0.01 ->  NaN Division_impossible":     true,
	"dqrmn1051 remaindernear  1e+277  1e-311 ->  NaN Division_impossible":                            true,
	"dqrmn1052 remaindernear  1e+277 -1e-311 ->  NaN Division_impossible":                            true,
	"dqrmn1053 remaindernear -1e+277  1e-311 ->  NaN Division_impossible":                            true,
	"dqrmn1054 remaindernear -1e+277 -1e-311 ->  NaN Division_impossible":                            true,
	"dqrmn421 remaindernear   1E+6144        1  ->   NaN Division_impossible":                        true,
	"dqrmn772  remaindernear  1234500000000000000000067890123456   0.1  ->  NaN Division_impossible": true,
	"dqrmn773  remaindernear  1234500000000000000000067890123456   0.01 ->  NaN Division_impossible": true,

	// logx log10: the 35-digit operand is used unrounded by decNumber; as a
	// decimal128 operand it rounds to exactly 1, so log10 is 0.
	"logx1308 log10 1.0000000000000000000000000000000001 -> 4.342944819032518276511289189166051E-35 Inexact Rounded": true,
}

func roundingFromToken(tok string) round {
	switch tok {
	case "TIES_TO_EVEN":
		return const_ROUND_TIES_TO_EVEN
	case "TIES_TO_AWAY":
		return const_ROUND_TIES_TO_AWAY
	case "TOWARD_ZERO":
		return const_ROUND_TOWARD_ZERO
	case "TOWARD_POSITIVE":
		return const_ROUND_TOWARD_POSITIVE
	case "TOWARD_NEGATIVE":
		return const_ROUND_TOWARD_NEGATIVE
	default:
		return const_ROUND_TIES_TO_EVEN
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
	if c.src == srcIntel {
		expected = intelNanBinary(decodeOperand(c.operands[0]), decodeOperand(c.operands[1]), expected)
	}
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
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
	var z d128
	if c.src == srcIntel {
		// Intel's scalbn takes a plain integer scale (which may exceed decNumber's
		// bounded decimal-argument range), applied directly via d128_scaleB.
		n, err := strconv.Atoi(c.operands[1])
		if err != nil {
			return outUnsupported, "non-integer scalbn argument"
		}
		z = d128_scaleB_ctxnull(x, n, ctx)
	} else {
		z = scaleBFromDecimalArg(x, decodeOperand(c.operands[1]), ctx)
	}
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
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

// checkMinMaxNumber runs a min/max "Number" (NaN-as-missing) corpus case.
// decNumber implements 754-2008 minNum, which propagates a signaling NaN as a
// quiet NaN; our 754-2019 minimumNumber instead treats a signaling NaN as a
// missing operand and returns the number (still signalling invalid). The dectest
// lines where the two differ are in dectestSkips.
func checkMinMaxNumber(c *rosettaCase, op func(x, y d128, ctx *context) d128) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
	}
	if c.rounding == roundingUnsupported {
		return outUnsupported, "rounding mode not representable"
	}
	x := decodeOperand(c.operands[0])
	y := decodeOperand(c.operands[1])
	ctx := newContextRound(roundingFromToken(c.rounding))
	z := op(x, y, ctx)
	expected := decodeOperand(c.result)
	if c.src == srcIntel {
		isMax := strings.Contains(c.operation, "max")
		expected = intelNanBinary(x, y, expected)
		expected = intelNumber(x, y, expected, isMax)
	}
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkCompareInt runs a two-operand op returning a memcmp-style -1/0/1 code
// (totalOrder comparison). totalOrder never signals — even on sNaN — so the
// expected flags must be empty.
func checkCompareInt(c *rosettaCase, op func(x, y d128) int) (outcome, string) {
	if len(c.operands) != 2 {
		return outUnsupported, "bad operand count"
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
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
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
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), "",
		d128_format(decodeOperand(c.result), const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkFromWord64 runs a decimal64->decimal128 widening-decode op
// (from_bid64/from_dpd64): the operand is a 16-hex 64-bit word (parsed
// op-specifically, NOT via decodeOperand), the result the decoded decimal128.
// The decode is an exact-subset widening — it never rounds or signals — so the
// expected flags are always empty. Compared bitwise via the canonical d128.
func checkFromWord64(c *rosettaCase, op func(u uint64) d128) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	z := op(parseWord64(c.operands[0]))
	expected := decodeOperand(c.result)
	if d128_bitwiseEQ(z, expected) && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), "",
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// decodeNativeResult decodes a native-corpus result token, honoring the bare
// "Q"/"S" payload-0 NaN spellings (the shared decodeOperand — tuned for the
// dectest/Intel/native operand grammar — does not special-case them; the wrapper
// decodeOperandWrapper does, so this keeps the two harnesses symmetric).
func decodeNativeResult(s string) d128 {
	switch s {
	case "Q":
		return const_D128_QNAN0
	case "S":
		return const_D128_SNAN0
	default:
		return decodeOperand(s)
	}
}

// checkParse runs the strict decimal-string parse d128_parse. Native-only: the
// operand is the RAW string, taken verbatim (NOT via decodeOperand — a failure
// case is deliberately un-decodable). A `FAIL` result asserts the parser rejects
// the input (the observed "did it parse" must be false, i.e. boolResult with
// expected = (result == "1") = false); otherwise the parsed value is compared
// bitwise against the decoded expected. The strict parse is flag-free, so the
// expected flags are always empty. No dectest/Intel corpus pins the grammar.
func checkParse(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	z, ok := d128_parse(c.operands[0])
	if c.result == "FAIL" {
		if !ok {
			return outPass, ""
		}
		return outFail, "expected FAIL but parsed " + d128_format(z, const_FORMAT_COEFFICIENT_QEXP)
	}
	// Success line: a failed parse here is an impl error; poison with sNaN0 so it
	// mismatches the authored expected (never collides except a literal sNaN).
	if !ok {
		z = const_D128_SNAN0
	}
	expected := decodeNativeResult(c.result)
	if d128_bitwiseEQ(z, expected) && c.expectedFlags == "" {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), "",
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
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
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), "",
		d128_format(decodeOperand(c.result), const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
}

// checkQuantExp runs Intel's quantexp: the quantum (encoded) exponent of x as an
// integer. Never signals (finite/special handled internally), but flags are still
// compared for parity with the corpus expectation.
func checkQuantExp(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	got := d128_quantumInt_ctxnull(decodeOperand(c.operands[0]), ctx)
	if got == math.MinInt {
		// The core's non-finite quantum sentinel is the platform int min; Intel's
		// readtest records the 32-bit sentinel. Normalize before compare.
		got = math.MinInt32
	}
	expected, err := strconv.Atoi(c.result)
	if err != nil {
		return outUnsupported, "non-integer quantexp result"
	}
	if got == expected && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %d flags=%q want %d flags=%q", got, ctx.exceptionLetters(), expected, c.expectedFlags)
}

// checkToInt64 runs a directed convert-to-int64 op. On an invalid conversion the
// returned value is an implementation-defined sentinel; the corpus records
// Intel's (which the core matches), but if the result token does not parse as an
// integer we fall back to the signed sentinel math.MinInt64 and rely on the 'i'
// flag to gate the value.
func checkToInt64(c *rosettaCase, op func(x d128, ctxnull *context) int64) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	got := op(decodeOperand(c.operands[0]), ctx)
	expected, err := strconv.ParseInt(c.result, 10, 64)
	if err != nil {
		expected = math.MinInt64
	}
	if got == expected && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %d flags=%q want %d flags=%q", got, ctx.exceptionLetters(), expected, c.expectedFlags)
}

// checkToUInt64 runs a directed convert-to-uint64 op. The expected value can
// exceed Int64.MaxValue (Intel's 2^63 invalid sentinel), so it is parsed as
// uint64. On an invalid conversion the value is implementation-defined -- our
// core returns 0, Intel returns 2^63 -- so once the 'i' flag is raised only the
// flag is checked.
func checkToUInt64(c *rosettaCase, op func(x d128, ctxnull *context) uint64) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	ctx := newContextRound(roundingFromToken(c.rounding))
	got := op(decodeOperand(c.operands[0]), ctx)
	expected, err := strconv.ParseUint(c.result, 10, 64)
	if err != nil {
		expected = 0
	}
	invalid := strings.Contains(ctx.exceptionLetters(), "i")
	valueOK := (got == expected) || (invalid && got == 0)
	if valueOK && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %d flags=%q want %d flags=%q", got, ctx.exceptionLetters(), expected, c.expectedFlags)
}

// decNumber class strings indexed by d128_ieeeClass code (0..9).
var classStrings = [...]string{
	const_CLASS_SIGNALING_NAN:      "sNaN",
	const_CLASS_QUIET_NAN:          "NaN",
	const_CLASS_NEGATIVE_INFINITY:  "-Infinity",
	const_CLASS_NEGATIVE_NORMAL:    "-Normal",
	const_CLASS_NEGATIVE_SUBNORMAL: "-Subnormal",
	const_CLASS_NEGATIVE_ZERO:      "-Zero",
	const_CLASS_POSITIVE_ZERO:      "+Zero",
	const_CLASS_POSITIVE_SUBNORMAL: "+Subnormal",
	const_CLASS_POSITIVE_NORMAL:    "+Normal",
	const_CLASS_POSITIVE_INFINITY:  "+Infinity",
}

// checkClass runs §5.7.2 class: maps the ieeeClass code to decNumber's class
// string. Non-computational — never signals, so expected flags must be empty.
func checkClass(c *rosettaCase) (outcome, string) {
	if len(c.operands) != 1 {
		return outUnsupported, "bad operand count"
	}
	cls := d128_ieeeClass(decodeOperand(c.operands[0]))
	var got string
	if c.src == srcIntel {
		// Intel's bid128_class returns the numeric class code, not a string.
		got = strconv.Itoa(cls)
	} else {
		got = classStrings[cls]
	}
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
	if c.src == srcIntel {
		expected = intelNanTernary(decodeOperand(c.operands[0]), decodeOperand(c.operands[1]), decodeOperand(c.operands[2]), expected)
	}
	if d128_bitwiseEQ(z, expected) && ctx.exceptionLetters() == c.expectedFlags {
		return outPass, ""
	}
	return outFail, fmt.Sprintf("got %s flags=%q want %s flags=%q",
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
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
		d128_format(z, const_FORMAT_COEFFICIENT_QEXP), ctx.exceptionLetters(),
		d128_format(expected, const_FORMAT_COEFFICIENT_QEXP), c.expectedFlags)
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
	if code == const_CMP754_UNORDERED {
		// Unordered (a NaN operand): the op result is the propagated NaN value
		// (selected/quieted without re-signaling — compare already raised flags).
		gotNaN := ctxNanOperand2_ctxnull(nil, x, y)
		match = d128_bitwiseEQ(gotNaN, decodeOperand(c.result))
		got = d128_format(gotNaN, const_FORMAT_AUTO)
	} else {
		n := code - 1 // LT->-1, EQ->0, GT->1
		if len(c.result) > 0 && (c.result[0] == '#' || c.result[0] == '[') {
			// dqCanonical gives the result as a canonical encoding of the
			// -1/0/1 code rather than its decimal text — compare bitwise.
			got = d128_format(d128_fromInt64(n), const_FORMAT_COEFFICIENT_QEXP)
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

// dispatchApply handles the dectest codec round-trip op: a "#…" result means
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
	got := d128_format(val, const_FORMAT_AUTO)
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
	// reasons counts unsupported cases by reason.
	reasons map[string]int
}

func (r *rosettaResult) tallyReason(reason string) {
	if r.reasons == nil {
		r.reasons = map[string]int{}
	}
	r.reasons[reason]++
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

// isNativeNonCase reports whether a native-corpus line is blank or a "//"
// comment: the only lines that are not cases.
func isNativeNonCase(line string) bool {
	t := strings.TrimSpace(line)
	return (t == "") || strings.HasPrefix(t, "//")
}

// loadNativeFile reads all native-corpus cases from a file. Every line that is
// not blank or a "//" comment must parse as a case: a malformed line is an
// error naming file:line, never a silent drop (D48GoWorkOrder.md G-10).
func loadNativeFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cases []rosettaCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		text := sc.Text()
		c, ok := parseNativeLine(text)
		if !ok {
			if isNativeNonCase(text) {
				continue
			}
			return nil, fmt.Errorf("%s:%d: not a native case (want `op ROUNDING operand... -> result [flags]`): %s",
				filepath.Base(path), lineNo, strings.TrimSpace(text))
		}
		c.line = lineNo
		cases = append(cases, c)
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
	if i := strings.Index(line, "--"); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	arrow := strings.Index(line, "->")
	if arrow < 0 {
		return rosettaCase{}, false // directive
	}
	left := dectestTokens(line[:arrow])
	right := dectestTokens(line[arrow+len("->"):])
	if len(left) < 3 || len(right) < 1 {
		return rosettaCase{}, false
	}
	operands := append([]string(nil), left[2:]...)
	// Every operation maps through the dectest include map (Rosetta.md §9); an
	// excluded or undecided token is dropped here and caught by the coverage test.
	op, ok := dectestInclude[strings.ToLower(left[1])]
	if !ok {
		return rosettaCase{}, false
	}
	c := rosettaCase{
		src:       srcDectest,
		text:      line,
		operation: op,
		operands:  operands,
		result:    right[0],
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
		// "up", "05up" and "half_down" are not IEEE 754 decimal modes;
		// loadDectestFile drops the lines under such a directive.
		return roundingUnsupported, true
	}
}

const roundingUnsupported = "UNSUPPORTED_ROUNDING"

// parseDectestPrecision maps a "precision:" directive to its value. ok=false for
// non-directive lines.
func parseDectestPrecision(line string) (int, bool) {
	l := strings.TrimSpace(strings.ToLower(line))
	if !strings.HasPrefix(l, "precision:") {
		return 0, false
	}
	if i := strings.Index(l, "--"); i >= 0 {
		l = l[:i]
	}
	p, err := strconv.Atoi(strings.TrimSpace(l[len("precision:"):]))
	if err != nil {
		return 0, false
	}
	return p, true
}

// dectestTokens splits a dectest line fragment on whitespace, honouring quoted
// tokens: a token starting with ' or " runs to the matching quote (a doubled
// quote inside is a literal quote) and is returned without its quotes, so a
// quoted operand may contain spaces.
func dectestTokens(s string) []string {
	var out []string
	i, n := 0, len(s)
	for i < n {
		for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
			i++
		}
		if i >= n {
			break
		}
		q := s[i]
		if q != '\'' && q != '"' {
			start := i
			for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\r' && s[i] != '\n' {
				i++
			}
			out = append(out, s[start:i])
			continue
		}
		var b strings.Builder
		i++
		for i < n {
			if s[i] == q {
				if i+1 < n && s[i+1] == q {
					b.WriteByte(q)
					i += 2
					continue
				}
				i++
				break
			}
			b.WriteByte(s[i])
			i++
		}
		out = append(out, b.String())
	}
	return out
}

func loadDectestFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []rosettaCase
	rounding := "TIES_TO_EVEN"
	precision := 34
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if r, ok := parseDectestRounding(line); ok {
			rounding = r
			continue
		}
		if p, ok := parseDectestPrecision(line); ok {
			precision = p
			continue
		}
		if precision != 34 {
			// Only decimal128 (precision 34) cases are in scope.
			continue
		}
		// A rounding directive with no IEEE 754 mode (half_down, 05up, ...) has no
		// mode to run the line in, and a bare "#" operand is decTest's null-operand
		// test, not a value: Swift's loader drops both, so every port loads the same
		// lines (WrapperDispatchCoverageWorkOrder.md WC-3, D-1).
		if rounding == roundingUnsupported {
			continue
		}
		if c, ok := parseDectestLine(line); ok && !slices.Contains(c.operands, "#") {
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
		if shouldSkip(&cases[i]) {
			// Per-line skips are counted, not run: decrement total so the tally
			// reflects only the cases actually dispatched.
			r.total--
			continue
		}
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
			r.tallyReason(diag)
		}
	}
	return r
}
