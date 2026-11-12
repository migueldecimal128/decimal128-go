package decimal128

// Rosetta: this port's half of the shared run set (RosettaHarnessAlignmentWorkOrder.md).
// The operator maps, skips, file lists and expected counts are data in
// resources/rosetta (rosetta_shared_test.go); the partition is checked once,
// by resources/rosetta/check_rosetta.py. Here: the operand count of every
// dispatch token (R-7), and the checks only this port can make (R-2).
//
// Test-only code, exempt from the §3 constraint regime.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rosettaArity is the operand count every case of a dispatch token must carry
// (R-7). A token missing here has no dispatch arm. An explicit table so it
// ports verbatim.
var rosettaArity = map[string]int{
	// 1 operand
	"d128_toString":                     1,
	"d128_toExpString":                  1,
	"d128_toEngString":                  1,
	"d128_toRawString":                  1,
	"d128_isSigned":                     1,
	"d128_isInf":                        1,
	"d128_isFinite":                     1,
	"d128_isNaN":                        1,
	"d128_isSignaling":                  1,
	"d128_isQNaN":                       1,
	"d128_isNormal":                     1,
	"d128_isSubnormal":                  1,
	"d128_isZero":                       1,
	"d128_quantexp":                     1,
	"d128_to_int64_rnint":               1,
	"d128_to_int64_rninta":              1,
	"d128_to_int64_ceil":                1,
	"d128_to_int64_floor":               1,
	"d128_to_int64_int":                 1,
	"d128_to_int64_xrnint":              1,
	"d128_to_int64_xrninta":             1,
	"d128_to_int64_xceil":               1,
	"d128_to_int64_xfloor":              1,
	"d128_to_int64_xint":                1,
	"d128_to_uint64_rnint":              1,
	"d128_to_uint64_rninta":             1,
	"d128_to_uint64_ceil":               1,
	"d128_to_uint64_floor":              1,
	"d128_to_uint64_int":                1,
	"d128_to_uint64_xrnint":             1,
	"d128_to_uint64_xrninta":            1,
	"d128_to_uint64_xceil":              1,
	"d128_to_uint64_xfloor":             1,
	"d128_to_uint64_xint":               1,
	"d128_roundToIntegralSignalInexact": 1,
	"tointegralx":                       1,
	"d128_round_integral_nearest_even":  1,
	"d128_round_integral_nearest_away":  1,
	"d128_round_integral_zero":          1,
	"d128_round_integral_positive":      1,
	"d128_round_integral_negative":      1,
	"d128_sqrt_ctx":                     1,
	"squareroot":                        1,
	"d128_stripTrailingZeros":           1,
	"reduce":                            1,
	"d128_logB_ctxnull":                 1,
	"logb":                              1,
	"d128_nextUp_ctxnull":               1,
	"nextplus":                          1,
	"d128_nextDown_ctxnull":             1,
	"nextminus":                         1,
	"d128_copy":                         1,
	"copy":                              1,
	"d128_abs":                          1,
	"copyabs":                           1,
	"d128_negate":                       1,
	"copynegate":                        1,
	"d128_ieeeClass":                    1,
	"class":                             1,
	"d128_canonical":                    1,
	"canonical":                         1,
	"d128_exp_ctx":                      1,
	"d128_exp10_ctx":                    1,
	"d128_ln_ctx":                       1,
	"d128_log10_ctx":                    1,
	"d128_lnp1_ctx":                     1,
	"d128_log2p1_ctx":                   1,
	"d128_log10p1_ctx":                  1,
	"d128_expm1_ctx":                    1,
	"d128_exp10m1_ctx":                  1,
	"d128_fitsInt64":                    1,
	"d128_fitsUInt64":                   1,
	"d128_square_ctx":                   1,
	"d128_from_bid64":                   1,
	"d128_from_dpd64":                   1,
	"d128_parse":                        1,
	"apply":                             1,
	"abs":                               1,
	"minus":                             1,
	"tosci":                             1,
	"toeng":                             1,
	"from_int64":                        1,
	"from_uint64":                       1,
	"parse_ctx":                         1,
	"round_integral_mode":               1,
	"to_int64_xmode":                    1,
	// 2 operands
	"d128_add_ctx":                         2,
	"add":                                  2,
	"d128_subtract_ctx":                    2,
	"subtract":                             2,
	"d128_multiply_ctx":                    2,
	"multiply":                             2,
	"d128_divide_ctx":                      2,
	"divide":                               2,
	"d128_compare_ctx":                     2,
	"compare":                              2,
	"d128_compareSignaling_ctx":            2,
	"comparesig":                           2,
	"d128_isEqual_ctx":                     2,
	"d128_isLess_ctx":                      2,
	"d128_isGreater_ctx":                   2,
	"d128_isGreaterEqual_ctx":              2,
	"d128_isGreaterUnordered_ctx":          2,
	"d128_isLessEqual_ctx":                 2,
	"d128_isLessUnordered_ctx":             2,
	"d128_isNotEqual_ctx":                  2,
	"d128_isNotGreater_ctx":                2,
	"d128_isNotLess_ctx":                   2,
	"d128_isOrdered_ctx":                   2,
	"d128_isUnordered_ctx":                 2,
	"d128_isEqualSignaling_ctx":            2,
	"d128_isGreaterSignaling_ctx":          2,
	"d128_isGreaterEqualSignaling_ctx":     2,
	"d128_isGreaterUnorderedSignaling_ctx": 2,
	"d128_isLessSignaling_ctx":             2,
	"d128_isLessEqualSignaling_ctx":        2,
	"d128_isLessUnorderedSignaling_ctx":    2,
	"d128_isNotEqualSignaling_ctx":         2,
	"d128_isNotGreaterSignaling_ctx":       2,
	"d128_isNotLessSignaling_ctx":          2,
	"d128_isOrderedSignaling_ctx":          2,
	"d128_isUnorderedSignaling_ctx":        2,
	"d128_totalOrderMag":                   2,
	"d128_divideInteger_ctx":               2,
	"divideint":                            2,
	"d128_remainderTrunc_ctx":              2,
	"remainder":                            2,
	"d128_remainderNear_ctx":               2,
	"remaindernear":                        2,
	"d128_quantize_ctx":                    2,
	"quantize":                             2,
	"d128_scaleB_ctxnull":                  2,
	"scaleb":                               2,
	"d128_isSameQuantum":                   2,
	"samequantum":                          2,
	"d128_minimum_ctx":                     2,
	"d128_maximum_ctx":                     2,
	"d128_minimumMagnitude_ctx":            2,
	"d128_maximumMagnitude_ctx":            2,
	"d128_minimumNumber_ctx":               2,
	"min":                                  2,
	"d128_maximumNumber_ctx":               2,
	"max":                                  2,
	"d128_minimumMagnitudeNumber_ctx":      2,
	"minmag":                               2,
	"d128_maximumMagnitudeNumber_ctx":      2,
	"maxmag":                               2,
	"d128_totalOrder":                      2,
	"comparetotal":                         2,
	"comparetotmag":                        2,
	"d128_copySign":                        2,
	"copysign":                             2,
	"d128_pow_ctx":                         2,
	"d128_pown_ctx":                        2,
	"d128_rootn_ctx":                       2,
	"d128_compound_ctx":                    2,
	// 3 operands
	"d128_fma_ctx": 3,
	"fma":          3,
}

// fitOperands applies the R-7 operand-count rule. Too few operands is always an
// error. Intel's own runner (readtest.c) splits a line by token count and passes
// each function only the operands it takes, so an Intel line may carry extra
// operands the op ignores (18 nextup/nextdown lines do): those are trimmed here.
// Every other source must match exactly.
func fitOperands(c *rosettaCase) bool {
	n, ok := rosettaArity[c.operation]
	if !ok {
		return false
	}
	if c.src == srcIntel && len(c.operands) > n {
		c.operands = c.operands[:n]
	}
	return len(c.operands) == n
}

// MARK: - corpus token scans (before any vector filter or skip list)

// MARK: - coverage meta-tests

// TestRosettaSharedDataLoads: files.index, the maps, the skips, runner.exclude
// and expected.tsv load, and every include value has a rosettaCanonical entry
// (RosettaHarnessAlignmentWorkOrder.md R-2). The partition itself is checked
// once, in resources/rosetta/check_rosetta.py.
func TestRosettaSharedDataLoads(t *testing.T) {
	if errs := sharedData().errors; len(errs) != 0 {
		t.Fatalf("shared Rosetta data:\n  %s", strings.Join(errs, "\n  "))
	}
	for _, src := range rosettaMappedSources {
		if len(sharedFiles(src)) == 0 {
			t.Errorf("files.index lists no %s files", src)
		}
		if len(sharedData().maps[src].include) == 0 {
			t.Errorf("%s.map includes nothing", src)
		}
	}
}

// TestRosettaIncludeTargetsHaveArity: every dispatch token an include value
// maps to has an arm with an arity, so an included operator can't reach the
// runner unrunnable (Go dispatches on strings).
func TestRosettaIncludeTargetsHaveArity(t *testing.T) {
	n := 0
	for _, src := range rosettaMappedSources {
		for tok, canonical := range sharedData().maps[src].include {
			op, _ := tokenForCanonical(src, canonical)
			n++
			if _, ok := rosettaArity[op]; !ok {
				t.Errorf("%s: %s -> %s -> %s has no arity", src, tok, canonical, op)
			}
		}
	}
	if n == 0 {
		t.Fatal("no include targets")
	}
}

// TestRosettaDectestScopeMatchesParser: the dectest loader applies
// dectest.map's `rounding` and `null-operand` scope rules itself; they must be
// exactly the data's. (`precision` it reads from the data, as the fptest parser
// does its `format`.)
func TestRosettaDectestScopeMatchesParser(t *testing.T) {
	listed := map[string]bool{}
	for _, m := range strings.Split(sharedScope("dectest", "rounding"), ",") {
		listed[m] = true
	}
	for _, m := range []string{"ceiling", "down", "floor", "half_down", "half_even", "half_up", "up", "05up"} {
		r, ok := parseDectestRounding("rounding: " + m)
		runs := ok && r != roundingUnsupported
		if runs != listed[m] {
			t.Errorf("dectest rounding %q: dectest.map lists it=%v, the parser runs it=%v", m, listed[m], runs)
		}
	}
	if got := sharedScope("dectest", "null-operand"); got != "#" {
		t.Errorf("dectest.map null-operand is %q, but the loader drops a bare '#'", got)
	}
	f := filepath.Join(t.TempDir(), "hash.decTest")
	if err := os.WriteFile(f, []byte("rounding: half_even\nt1 add # 1 -> NaN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cases, err := loadDectestFile(f); err != nil || len(cases) != 0 {
		t.Errorf("the loader no longer drops a bare '#' operand: %d cases, err %v", len(cases), err)
	}
	if got := sharedScope("dectest", "precision"); got != "34" {
		t.Errorf("dectest.map precision is %q, not 34", got)
	}
}
