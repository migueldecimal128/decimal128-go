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
	if len(tokens) == 0 || !intelInScope(tokens[0]) {
		return rosettaCase{}, false
	}
	mapped, ok := intelInclude[tokens[0]]
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

// intelSkips are per-line Intel cases skipped (counted, not failed): a handful
// where Intel's recorded cohort/NaN choice cannot be reconciled with decNumber's
// convention, and the exp / exp10 lines where Intel marks a tiny-input result
// exact (flags 00) although it is inexact. Mirrors Swift's RosettaIntelSkips.
var intelSkips = map[string]bool{
	"bid128_exp 0 [00000000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 0 [00000000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 0 [0000106F4216D9DBBD95C90355555555] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 0 [00420000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 0 [00420000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 1 [00000000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 1 [00000000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 1 [0000106F4216D9DBBD95C90355555555] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 1 [00420000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 1 [00420000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 2 [00000000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 2 [00000000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 2 [0000106F4216D9DBBD95C90355555555] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 2 [00420000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 2 [00420000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 3 [00000000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 3 [00000000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 3 [0000106F4216D9DBBD95C90355555556] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 3 [00420000000000000000000000000001] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp 3 [00420000000000000000000000000003] [2FFE314DC6448D9338C15B0A00000000] 00 ulp=0.0000000000e-01":                                                     true,
	"bid128_exp10 0 [b0058bf72fa4eea6e98489819368194d] [00000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [b005db5e877e758bfe53dd5e06733086] [00000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [b005b1d75299a5aa378acaa543a24b82] [00000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [b0053af01b8a0125ac5437185b9f74bb] [00000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [b0057e657e0fef7b8b7572acc164e1f3] [00000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [b00552f25788a4e163d781ec08d2be04] [00000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [3005da8d9306b37a08cd74840506b073] [78000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [30059144adac675798f2764aabb361f2] [78000000000000000000000000000000] 20":                                                                        true,
	"bid128_exp10 0 [3005967f3b6dc2b69607e01ce2a77a34] [78000000000000000000000000000000] 20":                                                                        true,
	"bid128_fma 0 [7f06536efdedf3ef,a00d0c0a84122060] [0008101000000000,8c6de2c3707e636c] [fd1f7eefdffff377,023b4c28d1de82a0] [fc000000000000000000000000000000] 01": true,
	"bid128_maxnum 0 [dbf2000000000000,0000000000000000] [9e06000000000000,0000000000000000] [dbf20000000000000000000000000000] 00":                                  true,
	"bid128_maxnum_mag 0 [0ad6000000000000,0000000000000000] [a3f2000000000000,0000000000000000] [a3f20000000000000000000000000000] 00":                              true,
	"bid128_minnum_mag 0 [4a29fdd3add4507a,b6884d65b20ee626] [ed47f4a2ed981eb5,fe78fff8fbfd7fed] [4a280000000000000000000000000000] 00":                              true,
	// from_string: malformed strings. We return NaN and raise invalid; Intel
	// returns NaN with no flag (and reads the valid prefix of "1.1E-2E").
	"bid128_from_string 0 1.. [7c000000000000000000000000000000] 00":        true,
	"bid128_from_string 0 1.1P2 [7c000000000000000000000000000000] 00":      true,
	"bid128_from_string 0 1.1EE [7c000000000000000000000000000000] 00":      true,
	"bid128_from_string 0 1.1P-2 [7c000000000000000000000000000000] 00":     true,
	"bid128_from_string 0 1.1E-2E [303a000000000000000000000000000b] 00":    true,
	"bid128_from_string 0 +INFi [7c000000000000000000000000000000] 00":      true,
	"bid128_from_string 0 +SNANi [7e000000000000000000000000000000] 00":     true,
	"bid128_from_string 0 +INFINITYi [7c000000000000000000000000000000] 00": true,
	"bid128_from_string 0 INFi [7c000000000000000000000000000000] 00":       true,
	"bid128_from_string 0 SNANi [7e000000000000000000000000000000] 00":      true,
	"bid128_from_string 0 INFINITYi [7c000000000000000000000000000000] 00":  true,
}

// isOutOfScopeIntelLine: lrint/lround tagged `longintsize=32` test a 32-bit
// `long` result (like to_int32_*, out of scope). On scalbln the same tag
// describes the exponent argument, so those lines run.
func isOutOfScopeIntelLine(t string) bool {
	return (strings.HasPrefix(t, "bid128_lrint ") || strings.HasPrefix(t, "bid128_lround ")) &&
		strings.Contains(t, " longintsize=32")
}

// shouldSkip reports whether a case is on a per-line skip list (or out of
// scope). Skipped cases are counted (total decremented) rather than failed.
func shouldSkip(c *rosettaCase) bool {
	t := strings.TrimSpace(c.text)
	switch c.src {
	case srcIntel:
		return isOutOfScopeIntelLine(t) || intelSkips[t]
	case srcDectest:
		return dectestSkips[t]
	}
	return false
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
