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
				return d128_fromDPD(u)
			}
		} else if s[0] == '[' {
			if u, ok := bid128ParseIntelHex(s); ok {
				return d128_fromBID(u)
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
	case "d128_quantize", "quantize":
		return checkBinaryCtx(c, d128_quantize)
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
		code = d128_compareSignaling_ctx(x, y, ctx)
	} else {
		code = d128_compare_ctx(x, y, ctx)
	}
	var match bool
	var got string
	if code == CMP754_UNORDERED {
		// Unordered (a NaN operand): the op result is the propagated NaN value
		// (selected/quieted without re-signaling — compare already raised flags).
		gotNaN := ctxNanOperand2(nil, x, y)
		match = d128_bitwiseEQ(gotNaN, decodeOperand(c.result))
		got = d128_format(gotNaN, FORMAT_AUTO)
	} else {
		got = strconv.Itoa(code - 1) // LT->-1, EQ->0, GT->1
		match = got == c.result
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
		dpd := d128_toDPD(val)
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
