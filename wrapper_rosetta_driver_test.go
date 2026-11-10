package decimal128

import (
	"path/filepath"
	"testing"
)

// The dectest corpora driven through the public API (same files as the
// white-box gate that have a public-API arm).
var wrapperDectestFiles = []string{
	"dqAdd.decTest", "dqSubtract.decTest", "dqMultiply.decTest", "dqDivide.decTest",
	"dqFMA.decTest", "dqDivideInt.decTest", "dqRemainder.decTest", "dqRemainderNear.decTest",
	"dqQuantize.decTest", "dqReduce.decTest", "dqToIntegral.decTest", "dqEncode.decTest",
}

// Floors at the D48 switch (D48GoWorkOrder.md Phase 4): all 40 native files,
// nothing excluded.
const (
	nativeWrapperFloor = 30981
	wrapperTotalFloor  = 82054
	wrapperPassedFloor = 70820
)

// TestWrapperRosetta drives the conformance corpora through the public
// Decimal128 / Context API. The gate is zero failures; operations without a
// public-API arm are reported unsupported (and covered by the engine harness).
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

	// dectest corpora.
	for _, name := range wrapperDectestFiles {
		cases, err := loadDectestFile(filepath.Join(resourcesDir(), "dectest", name))
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		r := runWrapperRosetta(cases)
		mergeResult(&total, r)
		t.Logf("dectest %-20s %d cases — %d passed, %d unsupported, %d failed",
			name, r.total, r.passed, r.unsupported, r.failed)
	}

	// Intel BID operations corpus (readtest.in): NaN/min-max cohort normalization
	// is applied in wrapCheck; per-line skips are counted, not failed.
	intelCases, err := loadIntelFile(filepath.Join(resourcesDir(), "intel", "readtest.in"))
	if err != nil {
		t.Fatalf("load readtest.in: %v", err)
	}
	{
		r := runWrapperRosetta(intelCases)
		mergeResult(&total, r)
		t.Logf("intel   %-20s %d cases — %d passed, %d unsupported, %d failed",
			"readtest.in", r.total, r.passed, r.unsupported, r.failed)
	}

	// IEEE fptest corpora (raw cohort results, u-stripped flags, trap-enabled
	// lines and "#" don't-care results handled in wrapCheck).
	fptestFiles, err := filepath.Glob(filepath.Join(resourcesDir(), "fptest", "*.fptest"))
	if err != nil || len(fptestFiles) == 0 {
		t.Fatalf("glob fptest: %v (%d files)", err, len(fptestFiles))
	}
	for _, f := range fptestFiles {
		cases, err := loadFptestFile(f)
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		r := runWrapperRosetta(cases)
		mergeResult(&total, r)
		t.Logf("fptest  %-40s %d cases — %d passed, %d unsupported, %d failed",
			filepath.Base(f), r.total, r.passed, r.unsupported, r.failed)
	}

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

func mergeResult(total *rosettaResult, r rosettaResult) {
	total.total += r.total
	total.passed += r.passed
	total.failed += r.failed
	total.unsupported += r.unsupported
	total.failures = append(total.failures, r.failures...)
}
