package decimal128

import (
	"path/filepath"
	"testing"
)

func resourcesDir() string {
	return filepath.Join("testdata", "resources")
}

func nativeCorpusPath(name string) string {
	return filepath.Join("testdata", "resources", "native", name)
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

// TestRosettaNative runs every native-corpus file through the harness. Wired ops
// (serde/parse/print/predicate) must pass; arithmetic ops are still Unsupported
// (Phase 3) and are counted, not failed.
func TestRosettaNative(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(resourcesDir(), "native", "*.txt"))
	if err != nil || len(files) == 0 {
		t.Fatalf("glob native: %v (%d files)", err, len(files))
	}
	var total rosettaResult
	for _, f := range files {
		cases, err := loadNativeFile(f)
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		r := runRosetta(cases)
		total.total += r.total
		total.passed += r.passed
		total.failed += r.failed
		total.unsupported += r.unsupported
		total.failures = append(total.failures, r.failures...)
	}
	if total.failed != 0 {
		t.Fatalf("native corpus: %d/%d failed: %v", total.failed, total.total, sampleFailures(total))
	}
	if total.passed == 0 {
		t.Fatal("expected some supported native cases to pass")
	}
	t.Logf("native corpus: %d cases — %d passed, %d unsupported (arithmetic awaits Phase 3)",
		total.total, total.passed, total.unsupported)
}

// TestRosettaDectestEncode runs the IBM dectest DPD encode/decode vectors
// (dqEncode.decTest) through the serde dispatch — the first validation against
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
	for _, name := range []string{"dqAdd.decTest", "dqSubtract.decTest", "dqMultiply.decTest"} {
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
