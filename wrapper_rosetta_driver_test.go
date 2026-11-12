package decimal128

import (
	"path/filepath"
	"sort"
	"testing"
)

// Floors: native at the D48 switch (D48GoWorkOrder.md Phase 4); the wrapper
// totals exact at WrapperDispatchCoverageWorkOrder.md Phase 2.
const (
	nativeWrapperFloor = 30981
	wrapperTotalFloor  = 158205
	wrapperPassedFloor = 158205
)

// TestWrapperRosetta drives the conformance corpora through the public
// Decimal128 / Context API. The gate is zero failures. Every op the core runner
// dispatches has a public-API case, so the only skips are the shared lists.
func TestWrapperRosetta(t *testing.T) {
	var total rosettaResult

	// Native corpus: native.index minus this port's wrapper exclusions. Every
	// case must pass (G-10); per-file counts are logged.
	nat := runNativeCorpus(t, "wrapper", nativeCorpusFiles(t, "wrapper"), dispatchWrapper)
	if nat.failed != 0 {
		t.Fatalf("native wrapper: %d/%d failed: %v", nat.failed, nat.total, sampleFailures(nat))
	}
	if nat.total < nativeWrapperFloor {
		t.Fatalf("native wrapper: %d cases, below the floor %d", nat.total, nativeWrapperFloor)
	}
	mergeResult(&total, nat)

	// dectest corpora: the full file list (R-6).
	var dectestTotal rosettaResult
	for _, name := range dectestFiles {
		cases, err := loadDectestFile(filepath.Join(resourcesDir(), "dectest", name))
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		r := runWrapperRosetta(cases)
		mergeResult(&dectestTotal, r)
		t.Logf("dectest %-20s %d cases — %d passed, %d unsupported, %d failed",
			name, r.total, r.passed, r.unsupported, r.failed)
	}
	logRosettaTotal(t, "wrapper dectest", dectestTotal)
	mergeResult(&total, dectestTotal)

	// Intel BID operations corpus (readtest.in): NaN/min-max cohort normalization
	// is applied in wrapCheck; per-line skips are counted, not failed.
	intelCases, err := loadIntelFile(filepath.Join(resourcesDir(), "intel", "readtest.in"))
	if err != nil {
		t.Fatalf("load readtest.in: %v", err)
	}
	{
		r := runWrapperRosetta(intelCases)
		mergeResult(&total, r)
		logRosettaTotal(t, "wrapper intel", r)
	}

	// IEEE fptest corpora (raw cohort results, u-stripped flags; trap-enabled
	// lines un-wrapped and "#" don't-care results flag-checked in
	// wrapCheckFptest, as the core runner does).
	var fptestTotal rosettaResult
	for _, name := range fptestFiles {
		f := filepath.Join(resourcesDir(), "fptest", name)
		cases, err := loadFptestFile(f)
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		r := runWrapperRosetta(cases)
		mergeResult(&fptestTotal, r)
		t.Logf("fptest  %-40s %d cases — %d passed, %d unsupported, %d failed",
			filepath.Base(f), r.total, r.passed, r.unsupported, r.failed)
	}
	logRosettaTotal(t, "wrapper fptest", fptestTotal)
	mergeResult(&total, fptestTotal)

	if total.failed != 0 {
		t.Fatalf("WrapperRosetta: %d/%d failed: %v", total.failed, total.total, sampleFailures(total))
	}
	t.Logf("WrapperRosetta TOTAL: %d cases — %d passed, %d unsupported, 0 failed",
		total.total, total.passed, total.unsupported)
	if (total.total < wrapperTotalFloor) || (total.passed < wrapperPassedFloor) {
		t.Fatalf("WrapperRosetta: %d cases / %d passed, below the floors %d / %d",
			total.total, total.passed, wrapperTotalFloor, wrapperPassedFloor)
	}
}

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
	total.failures = append(total.failures, r.failures...)
	for reason, n := range r.reasons {
		for i := 0; i < n; i++ {
			total.tallyReason(reason)
		}
	}
}
