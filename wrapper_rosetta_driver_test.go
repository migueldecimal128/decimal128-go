package decimal128

import (
	"sort"
	"testing"
)

// The wrapper runner drives the same files through the public Decimal128 /
// Context API (rosettaSource / rosettaNative, rosetta_test.go).
func TestRosettaWrapperDectest(t *testing.T) { rosettaSource(t, "wrapper", "dectest", dispatchWrapper) }
func TestRosettaWrapperFptest(t *testing.T)  { rosettaSource(t, "wrapper", "fptest", dispatchWrapper) }
func TestRosettaWrapperIntel(t *testing.T)   { rosettaSource(t, "wrapper", "intel", dispatchWrapper) }
func TestRosettaWrapperNative(t *testing.T)  { rosettaNative(t, "wrapper", dispatchWrapper) }

// logRosettaTotal logs a corpus total and its unsupported cases by reason.
func logRosettaTotal(t *testing.T, label string, r rosettaResult) {
	t.Helper()
	t.Logf("%s: %d cases — %d passed, %d unsupported, %d failed",
		label, r.total, r.passed, r.unsupported, r.failed)
	reasons := make([]string, 0, len(r.reasons))
	for reason := range r.reasons {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		t.Logf("    skip %dx  %s", r.reasons[reason], reason)
	}
}

func mergeResult(total *rosettaResult, r rosettaResult) {
	total.total += r.total
	total.passed += r.passed
	total.failed += r.failed
	total.unsupported += r.unsupported
	total.skipped += r.skipped
	total.failures = append(total.failures, r.failures...)
	for reason, n := range r.reasons {
		for i := 0; i < n; i++ {
			total.tallyReason(reason)
		}
	}
}
