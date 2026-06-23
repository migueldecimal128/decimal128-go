package decimal128

import (
	"path/filepath"
	"testing"
)

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

// TestRosettaSkeleton is the Phase 1 gate for the harness: it loads real native
// cases, dispatches them, and reports zero FAILURES. Every operation is
// Unsupported until its dispatch arm lands, so 0 vectors execute while the
// load→dispatch→tally pipeline runs end-to-end over real corpus data.
func TestRosettaSkeleton(t *testing.T) {
	cases, err := loadNativeFile(nativeCorpusPath("add.txt"))
	if err != nil {
		t.Fatalf("load native add.txt: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("expected to parse native cases, got 0")
	}
	r := runRosetta(cases)
	if r.failed != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", r.failed, r.failures)
	}
	if r.passed != 0 {
		t.Fatalf("expected 0 passes (no arms wired), got %d", r.passed)
	}
	if r.unsupported != r.total {
		t.Fatalf("expected all %d cases unsupported, got %d", r.total, r.unsupported)
	}
	t.Logf("rosetta skeleton OK: %d native cases parsed, %d unsupported, %d failed",
		r.total, r.unsupported, r.failed)
}
