package decimal128

// Intel BID conformance corpus (resources/rosetta/intel/readtest.in) parser,
// operation map, result normalization, and per-line skip set — the Go analog of
// the C# RosettaTests.cs Intel machinery. Test-only code, exempt from the §3
// constraint regime.
//
// The Intel readtest vectors encode operands/results as BID hex ("[...]") and
// carry a numeric rounding selector plus a hex exception-flag byte. Intel's
// NaN-propagation and min/max cohort conventions differ from decNumber's, so the
// expected result is normalized (see intelNanBinary/intelNanTernary/intelNumber)
// before the bitwise compare when the case source is srcIntel.

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// intelRoundingToToken maps Intel's numeric rounding selector to the canonical
// rounding token.
func intelRoundingToToken(r int) (string, bool) {
	switch r {
	case 0:
		return "TIES_TO_EVEN", true
	case 1:
		return "TOWARD_NEGATIVE", true
	case 2:
		return "TOWARD_POSITIVE", true
	case 3:
		return "TOWARD_ZERO", true
	case 4:
		return "TIES_TO_AWAY", true
	default:
		return "", false
	}
}

// intelFlagsToExceptionLetters maps Intel's hex exception-flag byte to canonical
// xuozi letters (0x20 inexact, 0x10 underflow, 0x08 overflow, 0x04 divByZero,
// 0x01 invalid; 0x02 denormal has no IEEE flag and is ignored).
func intelFlagsToExceptionLetters(flags int) string {
	var b strings.Builder
	if flags&0x20 != 0 {
		b.WriteByte('x')
	}
	if flags&0x10 != 0 {
		b.WriteByte('u')
	}
	if flags&0x08 != 0 {
		b.WriteByte('o')
	}
	if flags&0x04 != 0 {
		b.WriteByte('z')
	}
	if flags&0x01 != 0 {
		b.WriteByte('i')
	}
	return b.String()
}

// isIntelFlagRegister reports whether tok is a bare 1–2 hex-digit token (the
// exception-flag column), optionally 0x-prefixed.
func isIntelFlagRegister(tok string) bool {
	t := strings.TrimPrefix(strings.TrimPrefix(tok, "0x"), "0X")
	if len(t) < 1 || len(t) > 2 {
		return false
	}
	for i := 0; i < len(t); i++ {
		if !strings.ContainsRune("0123456789abcdefABCDEF", rune(t[i])) {
			return false
		}
	}
	return true
}

// parseIntelLine parses one readtest.in line into a rosettaCase. ok=false for
// comments, blanks, and operations not in intelInclude (excluded or undecided —
// the coverage test catches undecided ones). The operand count comes from the
// line shape, never a per-op table (R-7):
//
//	<func> <rnd-int> <op1> [<op2>] [<op3>] <result> <flags-hex> [tail-attrs]
//
// The flag register is the rightmost bare 1–2 hex-digit token, so the trailing
// attributes (ulp=, longintsize=, str_prefix=|…|) need no parsing.
func parseIntelLine(line string) (rosettaCase, bool) {
	body := line
	if i := strings.Index(body, "--"); i >= 0 {
		body = body[:i]
	}
	tokens := strings.Fields(body)
	if len(tokens) == 0 {
		return rosettaCase{}, false
	}
	mapped, ok := sharedOpFor("intel", tokens[0])
	if !ok {
		return rosettaCase{}, false
	}
	fi := -1
	for j := len(tokens) - 1; j >= 0; j-- {
		if isIntelFlagRegister(tokens[j]) {
			fi = j
			break
		}
	}
	if fi < 4 {
		return rosettaCase{}, false
	}
	r, err := strconv.Atoi(tokens[1])
	if err != nil {
		return rosettaCase{}, false
	}
	rounding, ok := intelRoundingToToken(r)
	if !ok {
		return rosettaCase{}, false
	}
	flags, err := strconv.ParseInt(strings.TrimPrefix(tokens[fi], "0x"), 16, 64)
	if err != nil {
		return rosettaCase{}, false
	}
	return rosettaCase{
		src:           srcIntel,
		text:          line,
		token:         tokens[0],
		operation:     mapped,
		operands:      append([]string(nil), tokens[2:fi-1]...),
		result:        tokens[fi-1],
		rounding:      rounding,
		expectedFlags: intelFlagsToExceptionLetters(int(flags)),
	}, true
}

// intelNanBinary normalizes a two-operand NaN result to decNumber's
// first-signaling-then-first-operand cohort convention. Intel's readtest.in
// records the operand-1 quieted NaN even when decNumber would pick operand 2, so
// when the Intel-expected value matches Intel's choice we substitute our choice.
func intelNanBinary(op1, op2, expected d128) d128 {
	if !(d128_isNAN(expected) && d128_isNAN(op1) && d128_isNAN(op2)) {
		return expected
	}
	intelChoice := d128_quieted(op1)
	if d128_bitwiseEQ(expected, intelChoice) {
		if d128_isSNAN(op1) {
			return d128_quieted(op1)
		}
		if d128_isSNAN(op2) {
			return d128_quieted(op2)
		}
		return d128_quieted(op1)
	}
	return expected
}

// intelNanTernary normalizes fma's NaN result to decNumber's cohort convention.
func intelNanTernary(op1, op2, op3, expected d128) d128 {
	if !d128_isNAN(expected) {
		return expected
	}
	if !(d128_isNAN(op1) || d128_isNAN(op2) || d128_isNAN(op3)) {
		return expected
	}
	src := op3
	if d128_isNAN(op2) {
		src = op2
	}
	if d128_isNAN(op1) {
		src = op1
	}
	intelChoice := d128_quieted(src)
	if !d128_bitwiseEQ(expected, intelChoice) {
		return expected
	}
	if d128_isSNAN(op1) {
		return d128_quieted(op1)
	}
	if d128_isSNAN(op2) {
		return d128_quieted(op2)
	}
	if d128_isSNAN(op3) {
		return d128_quieted(op3)
	}
	if d128_isNAN(op1) {
		return d128_quieted(op1)
	}
	if d128_isNAN(op2) {
		return d128_quieted(op2)
	}
	return d128_quieted(op3)
}

var intelZero = d128_fromInt64(0)

// intelNumber normalizes min/max NaN-as-missing and equal-value cohort selection
// to decNumber's conventions (which pick a specific operand among a min/max
// cohort of equal values differing only in exponent).
func intelNumber(op1, op2, expected d128, isMax bool) d128 {
	xNaN := d128_isNAN(op1)
	yNaN := d128_isNAN(op2)

	if xNaN != yNaN && d128_isNAN(expected) {
		if xNaN && d128_isSNAN(op1) {
			return op2
		}
		if yNaN && d128_isSNAN(op2) {
			return op1
		}
	}

	if !xNaN && !yNaN &&
		comparePartialQuiet754(op1, op2) == const_CMP754_EQ &&
		comparePartialQuiet754(op1, intelZero) != const_CMP754_EQ &&
		!d128_bitwiseEQ(op1, op2) {
		negative := comparePartialQuiet754(op1, intelZero) == const_CMP754_LT
		var wantLargerExponent bool
		if isMax {
			wantLargerExponent = !negative
		} else {
			wantLargerExponent = negative
		}
		e1 := d128_quantumInt_ctxnull(op1, nil)
		e2 := d128_quantumInt_ctxnull(op2, nil)
		var keepFirst bool
		if wantLargerExponent {
			keepFirst = e1 >= e2
		} else {
			keepFirst = e1 <= e2
		}
		if keepFirst {
			return op1
		}
		return op2
	}

	return expected
}

// loadIntelFile reads all in-scope Intel operation cases from readtest.in.
func loadIntelFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []rosettaCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		if c, ok := parseIntelLine(sc.Text()); ok {
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}
