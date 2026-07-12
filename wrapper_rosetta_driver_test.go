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

// TestWrapperRosetta drives the conformance corpora through the public
// Decimal128 / Context API. The gate is zero failures; operations without a
// public-API arm are reported unsupported (and covered by the engine harness).
func TestWrapperRosetta(t *testing.T) {
	var total rosettaResult

	// Native corpus.
	natFiles, err := filepath.Glob(filepath.Join(resourcesDir(), "native", "*.txt"))
	if err != nil || len(natFiles) == 0 {
		t.Fatalf("glob native: %v (%d files)", err, len(natFiles))
	}
	for _, f := range natFiles {
		cases, err := loadNativeFile(f)
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		r := runWrapperRosetta(cases)
		mergeResult(&total, r)
		t.Logf("native %-18s %d cases — %d passed, %d unsupported, %d failed",
			filepath.Base(f), r.total, r.passed, r.unsupported, r.failed)
	}

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
}

func mergeResult(total *rosettaResult, r rosettaResult) {
	total.total += r.total
	total.passed += r.passed
	total.failed += r.failed
	total.unsupported += r.unsupported
	total.failures = append(total.failures, r.failures...)
}
