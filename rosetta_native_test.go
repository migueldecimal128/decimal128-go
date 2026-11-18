package decimal128

// Rosetta — proof that the native dialect runs through the same dispatch +
// reconcile core as the three corpora: a handful of arithmetic and predicate
// lines, through the single-line and inline-bulk paths.

import "testing"

// rosettaNativeLines runs native lines given inline through the core dispatch:
// every line must pass.
func rosettaNativeLines(t *testing.T, lines ...string) {
	t.Helper()
	var cases []rosettaCase
	for _, l := range lines {
		c, ok := parseNativeLine(l)
		if !ok {
			t.Fatalf("not a native line: %s", l)
		}
		cases = append(cases, c)
	}
	r := runCases(cases, dispatch)
	if r.failed != 0 || r.passed != len(lines) {
		t.Errorf("native lines: %d passed, %d failed: %v", r.passed, r.failed, sampleFailures(r))
	}
}

// Single-case path — one line, full verdict on failure.
func TestRosettaNativeAssertOneArithmetic(t *testing.T) {
	rosettaNativeLines(t, "d128_add_ctx TIES_TO_EVEN 1.5 2.5 -> 4.0")
	rosettaNativeLines(t, "d128_divide_ctx TIES_TO_EVEN 1 3 -> 0.3333333333333333333333333333333333 x")
}

func TestRosettaNativeAssertOnePredicate(t *testing.T) {
	rosettaNativeLines(t, "d128_isLess_ctx TIES_TO_EVEN -Inf 0 -> 1")
}

// Inline-bulk path — arithmetic across all four rounding-relevant ops.
func TestRosettaNativeInlineArithmetic(t *testing.T) {
	rosettaNativeLines(t,
		"d128_add_ctx TIES_TO_EVEN 1 2 -> 3",
		"d128_subtract_ctx TIES_TO_EVEN 5 2 -> 3",
		"d128_multiply_ctx TIES_TO_EVEN 1.5 2 -> 3.0",
		"d128_divide_ctx TIES_TO_EVEN 1 4 -> 0.25",
		"d128_fma_ctx TIES_TO_EVEN 2 3 4 -> 10",
		"d128_sqrt_ctx TIES_TO_EVEN 4 -> 2",
	)
}

// Inline-bulk path — predicates and compares return Bool/Comparison754/Int.
func TestRosettaNativeInlinePredicates(t *testing.T) {
	rosettaNativeLines(t,
		"d128_isNaN TIES_TO_EVEN NaN -> 1",
		"d128_isNaN TIES_TO_EVEN 1 -> 0",
		"d128_isInf TIES_TO_EVEN Inf -> 1",
		"d128_isZero TIES_TO_EVEN 0 -> 1",
		"d128_isSigned TIES_TO_EVEN -1 -> 1",
		"d128_isLess_ctx TIES_TO_EVEN 1 2 -> 1",
		"d128_isGreater_ctx TIES_TO_EVEN 1 2 -> 0",
		"d128_isEqual_ctx TIES_TO_EVEN 1.0 1 -> 1",
		"d128_compare_ctx TIES_TO_EVEN 1 2 -> -1",
		"d128_compare_ctx TIES_TO_EVEN 2 2 -> 0",
		"d128_compareTotal TIES_TO_EVEN 1 2 -> -1",
	)
}
