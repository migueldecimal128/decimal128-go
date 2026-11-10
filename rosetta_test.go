package decimal128

import (
	"path/filepath"
	"testing"
)

func resourcesDir() string {
	return filepath.Join("resources", "rosetta")
}

func nativeCorpusPath(name string) string {
	return filepath.Join("resources", "rosetta", "native", name)
}

func TestParseNativeLine(t *testing.T) {
	c, ok := parseNativeLine("d128_add_ctx TIES_TO_EVEN Infinity -Infinity -> NaN i")
	if !ok {
		t.Fatal("expected the line to parse")
	}
	if c.operation != "d128_add_ctx" {
		t.Errorf("operation = %q, want d128_add_ctx", c.operation)
	}
	if c.rounding != "TIES_TO_EVEN" {
		t.Errorf("rounding = %q, want TIES_TO_EVEN", c.rounding)
	}
	if len(c.operands) != 2 || c.operands[0] != "Infinity" || c.operands[1] != "-Infinity" {
		t.Errorf("operands = %v, want [Infinity -Infinity]", c.operands)
	}
	if c.result != "NaN" {
		t.Errorf("result = %q, want NaN", c.result)
	}
	if c.expectedFlags != "i" {
		t.Errorf("flags = %q, want i", c.expectedFlags)
	}
	if _, ok := parseNativeLine("// a comment"); ok {
		t.Error("comment line should not parse")
	}
	if _, ok := parseNativeLine("   "); ok {
		t.Error("blank line should not parse")
	}
}

func TestFlagLettersCanonical(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ix", "xi"},
		{"", ""},
		{"zoux", "xuoz"},
		{"xuozi", "xuozi"},
	}
	for _, tc := range cases {
		if got := flagLettersCanonical(tc.in); got != tc.want {
			t.Errorf("flagLettersCanonical(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func sampleFailures(r rosettaResult) []string {
	return r.failures[:min(len(r.failures), 10)]
}

// nativeCoreFloor is the core runner's native case count at the D48 switch
// (D48GoWorkOrder.md Phase 4): all 40 native files, nothing excluded.
const nativeCoreFloor = 30981

// TestRosettaNative runs the native-corpus files (native.index minus this
// port's exclusions) through the core harness. Every case must pass: a
// malformed line, an unknown op or an unsupported case is a failure (G-10).
func TestRosettaNative(t *testing.T) {
	total := runNativeCorpus(t, "core", nativeCorpusFiles(t, "core"), dispatch)
	if total.failed != 0 {
		t.Fatalf("native core: %d/%d failed: %v", total.failed, total.total, sampleFailures(total))
	}
	if total.total < nativeCoreFloor {
		t.Fatalf("native core: %d cases, below the floor %d", total.total, nativeCoreFloor)
	}
}

// TestRosettaDectestEncode runs the IBM dectest DPD encode/decode vectors
// (dqEncode.decTest) through the codec dispatch — the first validation against
// real IBM conformance data.
func TestRosettaDectestEncode(t *testing.T) {
	cases, err := loadDectestFile(filepath.Join(resourcesDir(), "dectest", "dqEncode.decTest"))
	if err != nil {
		t.Fatalf("load dqEncode: %v", err)
	}
	r := runRosetta(cases)
	if r.failed != 0 {
		t.Fatalf("dqEncode: %d/%d failed: %v", r.failed, r.total, sampleFailures(r))
	}
	if r.passed == 0 {
		t.Fatal("expected dqEncode apply cases to pass")
	}
	t.Logf("dqEncode: %d cases — %d passed, %d unsupported", r.total, r.passed, r.unsupported)
}

// TestRosettaDectestAddSub runs the IBM dectest add/subtract corpora through the
// arithmetic dispatch (value + exception flags).
func TestRosettaDectestAddSub(t *testing.T) {
	for _, name := range []string{"dqAdd.decTest", "dqSubtract.decTest", "dqMultiply.decTest", "dqDivide.decTest", "dqCompare.decTest", "dqCompareSig.decTest", "dqToIntegral.decTest", "dqDivideInt.decTest", "dqFMA.decTest", "dqRemainder.decTest", "dqRemainderNear.decTest", "dqReduce.decTest", "dqQuantize.decTest", "dqLogB.decTest", "dqScaleB.decTest", "dqSameQuantum.decTest", "dqMin.decTest", "dqMax.decTest", "dqMinMag.decTest", "dqMaxMag.decTest", "dqCompareTotal.decTest", "dqCompareTotalMag.decTest", "dqNextPlus.decTest", "dqNextMinus.decTest", "dqCopy.decTest", "dqCopyAbs.decTest", "dqCopyNegate.decTest", "dqCopySign.decTest", "dqClass.decTest", "dqCanonical.decTest"} {
		cases, err := loadDectestFile(filepath.Join(resourcesDir(), "dectest", name))
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		r := runRosetta(cases)
		if r.failed != 0 {
			t.Fatalf("%s: %d/%d failed: %v", name, r.failed, r.total, sampleFailures(r))
		}
		if r.passed == 0 {
			t.Fatalf("%s: no cases passed", name)
		}
		t.Logf("%s: %d cases — %d passed, %d unsupported", name, r.total, r.passed, r.unsupported)
	}
}

// TestRosettaIntel runs the Intel BID operations corpus (readtest.in ~14k
// bid128_* vectors) through the harness. Intel result normalization (NaN cohort
// + min/max conventions) is applied in the shared compare helpers; per-line
// skips are counted, not failed. Gates on 0 failures.
func TestRosettaIntel(t *testing.T) {
	cases, err := loadIntelFile(filepath.Join(resourcesDir(), "intel", "readtest.in"))
	if err != nil {
		t.Fatalf("load readtest.in: %v", err)
	}
	r := runRosetta(cases)
	t.Logf("intel readtest.in: %d cases — %d passed, %d unsupported, %d failed",
		r.total, r.passed, r.unsupported, r.failed)
	if r.failed != 0 {
		t.Fatalf("intel: %d/%d failed: %v", r.failed, r.total, sampleFailures(r))
	}
	if r.passed == 0 {
		t.Fatal("expected some intel cases to pass")
	}
}
