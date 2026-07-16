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

// intelInclude is the set of bid128_* operations the Go core supports and that
// the Intel operations runner exercises. Ops outside this set (the whole
// transcendental family, etc.) are skipped at parse time.
var intelInclude = map[string]bool{
	"bid128_abs": true, "bid128_add": true, "bid128_class": true, "bid128_copySign": true,
	"bid128_div": true, "bid128_fma": true, "bid128_isFinite": true, "bid128_isInf": true,
	"bid128_isNaN": true, "bid128_isNormal": true, "bid128_isSignaling": true, "bid128_isSigned": true,
	"bid128_isSubnormal": true, "bid128_isZero": true, "bid128_maxnum": true, "bid128_maxnum_mag": true,
	"bid128_minnum": true, "bid128_minnum_mag": true, "bid128_mul": true, "bid128_nextdown": true,
	"bid128_nextup": true, "bid128_quantexp": true, "bid128_quantize": true, "bid128_quiet_equal": true,
	"bid128_quiet_greater": true, "bid128_quiet_greater_equal": true, "bid128_quiet_greater_unordered": true,
	"bid128_quiet_less": true, "bid128_quiet_less_equal": true, "bid128_quiet_less_unordered": true,
	"bid128_quiet_not_equal": true, "bid128_quiet_not_greater": true, "bid128_quiet_not_less": true,
	"bid128_quiet_ordered": true, "bid128_quiet_unordered": true, "bid128_rem": true,
	"bid128_round_integral_exact": true, "bid128_round_integral_nearest_away": true,
	"bid128_round_integral_nearest_even": true, "bid128_round_integral_negative": true,
	"bid128_round_integral_positive": true, "bid128_round_integral_zero": true, "bid128_sameQuantum": true,
	"bid128_scalbn": true, "bid128_signaling_greater": true, "bid128_signaling_greater_equal": true,
	"bid128_signaling_greater_unordered": true, "bid128_signaling_less": true,
	"bid128_signaling_less_equal": true, "bid128_signaling_less_unordered": true,
	"bid128_signaling_not_greater": true, "bid128_signaling_not_less": true, "bid128_sqrt": true,
	"bid128_sub": true, "bid128_to_int64_rnint": true, "bid128_to_int64_rninta": true,
	"bid128_to_int64_ceil": true, "bid128_to_int64_floor": true, "bid128_to_int64_int": true,
	"bid128_to_int64_xrnint": true, "bid128_to_int64_xrninta": true, "bid128_to_int64_xceil": true,
	"bid128_to_int64_xfloor": true, "bid128_to_int64_xint": true,
	"bid128_to_uint64_rnint": true, "bid128_to_uint64_rninta": true, "bid128_to_uint64_ceil": true,
	"bid128_to_uint64_floor": true, "bid128_to_uint64_int": true, "bid128_to_uint64_xrnint": true,
	"bid128_to_uint64_xrninta": true, "bid128_to_uint64_xceil": true, "bid128_to_uint64_xfloor": true,
	"bid128_to_uint64_xint": true,
	"bid128_totalOrder":     true, "bid128_totalOrderMag": true,
}

// intelOpMap maps an included bid128_* operation to the harness dispatch token.
var intelOpMap = map[string]string{
	"bid128_abs":                         "d128_abs",
	"bid128_add":                         "add",
	"bid128_class":                       "class",
	"bid128_copySign":                    "copysign",
	"bid128_div":                         "divide",
	"bid128_fma":                         "fma",
	"bid128_isFinite":                    "d128_isFinite",
	"bid128_isInf":                       "d128_isInf",
	"bid128_isNaN":                       "d128_isNaN",
	"bid128_isNormal":                    "d128_isNormal",
	"bid128_isSignaling":                 "d128_isSignaling",
	"bid128_isSigned":                    "d128_isSigned",
	"bid128_isSubnormal":                 "d128_isSubnormal",
	"bid128_isZero":                      "d128_isZero",
	"bid128_maxnum":                      "max",
	"bid128_maxnum_mag":                  "maxmag",
	"bid128_minnum":                      "min",
	"bid128_minnum_mag":                  "minmag",
	"bid128_mul":                         "multiply",
	"bid128_nextdown":                    "nextminus",
	"bid128_nextup":                      "nextplus",
	"bid128_quantexp":                    "d128_quantexp",
	"bid128_quantize":                    "quantize",
	"bid128_quiet_equal":                 "d128_isEqual_ctx",
	"bid128_quiet_greater":               "d128_isGreater_ctx",
	"bid128_quiet_greater_equal":         "d128_isGreaterEqual_ctx",
	"bid128_quiet_greater_unordered":     "d128_isGreaterUnordered_ctx",
	"bid128_quiet_less":                  "d128_isLess_ctx",
	"bid128_quiet_less_equal":            "d128_isLessEqual_ctx",
	"bid128_quiet_less_unordered":        "d128_isLessUnordered_ctx",
	"bid128_quiet_not_equal":             "d128_isNotEqual_ctx",
	"bid128_quiet_not_greater":           "d128_isNotGreater_ctx",
	"bid128_quiet_not_less":              "d128_isNotLess_ctx",
	"bid128_quiet_ordered":               "d128_isOrdered_ctx",
	"bid128_quiet_unordered":             "d128_isUnordered_ctx",
	"bid128_rem":                         "remaindernear",
	"bid128_round_integral_exact":        "tointegralx",
	"bid128_round_integral_nearest_away": "d128_round_integral_nearest_away",
	"bid128_round_integral_nearest_even": "d128_round_integral_nearest_even",
	"bid128_round_integral_negative":     "d128_round_integral_negative",
	"bid128_round_integral_positive":     "d128_round_integral_positive",
	"bid128_round_integral_zero":         "d128_round_integral_zero",
	"bid128_sameQuantum":                 "samequantum",
	"bid128_scalbn":                      "scaleb",
	"bid128_signaling_greater":           "d128_isGreaterSignaling_ctx",
	"bid128_signaling_greater_equal":     "d128_isGreaterEqualSignaling_ctx",
	"bid128_signaling_greater_unordered": "d128_isGreaterUnorderedSignaling_ctx",
	"bid128_signaling_less":              "d128_isLessSignaling_ctx",
	"bid128_signaling_less_equal":        "d128_isLessEqualSignaling_ctx",
	"bid128_signaling_less_unordered":    "d128_isLessUnorderedSignaling_ctx",
	"bid128_signaling_not_greater":       "d128_isNotGreaterSignaling_ctx",
	"bid128_signaling_not_less":          "d128_isNotLessSignaling_ctx",
	"bid128_sqrt":                        "squareroot",
	"bid128_sub":                         "subtract",
	"bid128_to_int64_rnint":              "d128_to_int64_rnint",
	"bid128_to_int64_rninta":             "d128_to_int64_rninta",
	"bid128_to_int64_ceil":               "d128_to_int64_ceil",
	"bid128_to_int64_floor":              "d128_to_int64_floor",
	"bid128_to_int64_int":                "d128_to_int64_int",
	"bid128_to_int64_xrnint":             "d128_to_int64_xrnint",
	"bid128_to_int64_xrninta":            "d128_to_int64_xrninta",
	"bid128_to_int64_xceil":              "d128_to_int64_xceil",
	"bid128_to_int64_xfloor":             "d128_to_int64_xfloor",
	"bid128_to_int64_xint":               "d128_to_int64_xint",
	"bid128_to_uint64_rnint":             "d128_to_uint64_rnint",
	"bid128_to_uint64_rninta":            "d128_to_uint64_rninta",
	"bid128_to_uint64_ceil":              "d128_to_uint64_ceil",
	"bid128_to_uint64_floor":             "d128_to_uint64_floor",
	"bid128_to_uint64_int":               "d128_to_uint64_int",
	"bid128_to_uint64_xrnint":            "d128_to_uint64_xrnint",
	"bid128_to_uint64_xrninta":           "d128_to_uint64_xrninta",
	"bid128_to_uint64_xceil":             "d128_to_uint64_xceil",
	"bid128_to_uint64_xfloor":            "d128_to_uint64_xfloor",
	"bid128_to_uint64_xint":              "d128_to_uint64_xint",
	"bid128_totalOrder":                  "d128_totalOrder",
	"bid128_totalOrderMag":               "d128_totalOrderMag",
}

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

// getIntelOpCount returns the operand count for an Intel operation.
func getIntelOpCount(op string) int {
	if op == "bid128_fma" {
		return 3
	}
	if op == "bid128_abs" || op == "bid128_class" || strings.HasPrefix(op, "bid128_is") ||
		strings.HasPrefix(op, "bid128_round_integral_") || op == "bid128_sqrt" ||
		strings.HasPrefix(op, "bid128_to_int64_") || strings.HasPrefix(op, "bid128_to_uint64_") ||
		op == "bid128_quantexp" {
		return 1
	}
	return 2
}

// parseIntelLine parses one readtest.in line into a rosettaCase. ok=false for
// comments, blanks, and operations outside intelInclude.
func parseIntelLine(line string) (rosettaCase, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "--") {
		return rosettaCase{}, false
	}
	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return rosettaCase{}, false
	}
	op := tokens[0]
	if !strings.HasPrefix(op, "bid128_") || !intelInclude[op] {
		return rosettaCase{}, false
	}

	opCount := getIntelOpCount(op)
	// tokens: op, rounding, operands..., result, flagHex
	if len(tokens) < 2+opCount+2 {
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

	operands := make([]string, 0, opCount)
	for i := 0; i < opCount; i++ {
		operands = append(operands, tokens[2+i])
	}

	result := tokens[2+opCount]
	flagHexStr := strings.TrimPrefix(tokens[2+opCount+1], "0x")
	flags, err := strconv.ParseInt(flagHexStr, 16, 64)
	if err != nil {
		return rosettaCase{}, false
	}

	mapped, ok := intelOpMap[op]
	if !ok {
		mapped = op
	}

	return rosettaCase{
		src:           srcIntel,
		text:          line,
		operation:     mapped,
		operands:      operands,
		result:        result,
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
		comparePartialQuiet754(op1, op2) == CMP754_EQ &&
		comparePartialQuiet754(op1, intelZero) != CMP754_EQ &&
		!d128_bitwiseEQ(op1, op2) {
		negative := comparePartialQuiet754(op1, intelZero) == CMP754_LT
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
// convention. Mirrors the C# intelSkips set (entries for non-included ops such
// as bid128_exp are inert here — those ops never parse — but are kept for parity).
var intelSkips = map[string]bool{
	"bid128_fma 0 [7f06536efdedf3ef,a00d0c0a84122060] [0008101000000000,8c6de2c3707e636c] [fd1f7eefdffff377,023b4c28d1de82a0] [fc000000000000000000000000000000] 01": true,
	"bid128_maxnum 0 [dbf2000000000000,0000000000000000] [9e06000000000000,0000000000000000] [dbf20000000000000000000000000000] 00":                                  true,
	"bid128_maxnum_mag 0 [0ad6000000000000,0000000000000000] [a3f2000000000000,0000000000000000] [a3f20000000000000000000000000000] 00":                              true,
	"bid128_minnum_mag 0 [4a29fdd3add4507a,b6884d65b20ee626] [ed47f4a2ed981eb5,fe78fff8fbfd7fed] [4a280000000000000000000000000000] 00":                              true,
	"bid128_rem 0 [26741c0590811072,ffffffff6ef7ffff] [0000000000000000,0300040220100840] [8000000000000000015515ab6c6033c0] 00":                                     true,
	"bid128_rem 0 [397cdc9a254ff562,99bc2af7b316f57c] [268d807293a0dcac,afddbd631e831cbb] [a68c1d3c3991cbb19eee601c8bdcb12c] 00":                                     true,
	"bid128_rem 0 [2000040000002200,f7fffffffd7fffff] [0100000000000000,0000000000000001] [01000000000000000000000000000000] 00":                                     true,
	"bid128_rem 0 [4082000215010400,1f7dbeffb7fdff7d] [bfffbfff7ffa7fbf,240c30b8380ad037] [bffe0708356252ac4b444f585a0c6d93] 00":                                     true,
	"bid128_rem 0 [97f3ca2bca3dcd34,321ba8822721f58f] [8000000000000000,bae106a1402505d6] [000000000000000025bf9a7041265ffe] 00":                                     true,
	"bid128_rem 0 [9c5f9589d6c5fa51,caa6dcb43a12852e] [0080000000000000,0001000000000000] [80800000000000000000000000000000] 00":                                     true,
	"bid128_rem 0 [a7f734b7f7e7a516,229ddcc454df042c] [0000000000000002,9fc80ebfc7fed6d1] [800000000000000118eb45cf8f8cc645] 00":                                     true,
	"bid128_rem 1 [397cdc9a254ff562,99bc2af7b316f57c] [268d807293a0dcac,afddbd631e831cbb] [a68c1d3c3991cbb19eee601c8bdcb12c] 00":                                     true,
	"bid128_rem 2 [397cdc9a254ff562,99bc2af7b316f57c] [268d807293a0dcac,afddbd631e831cbb] [a68c1d3c3991cbb19eee601c8bdcb12c] 00":                                     true,
	"bid128_rem 3 [397cdc9a254ff562,99bc2af7b316f57c] [268d807293a0dcac,afddbd631e831cbb] [a68c1d3c3991cbb19eee601c8bdcb12c] 00":                                     true,
	"bid128_rem 4 [397cdc9a254ff562,99bc2af7b316f57c] [268d807293a0dcac,afddbd631e831cbb] [a68c1d3c3991cbb19eee601c8bdcb12c] 00":                                     true,
}

// shouldSkip reports whether a case is on a per-line skip list. Skipped cases
// are counted (total decremented) rather than failed.
func shouldSkip(c *rosettaCase) bool {
	if c.src == srcIntel {
		return intelSkips[strings.TrimSpace(c.text)]
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
