package rosetta

import (
	"path/filepath"
	"testing"
)

// resourcesDir locates the copied four-corpus tree, relative to this package
// directory (the test's working directory).
func resourcesDir() string {
	return filepath.Join("..", "..", "testdata", "resources")
}

func TestParseNativeLine(t *testing.T) {
	c, ok := ParseNativeLine("d128_add_ctx TIES_TO_EVEN Infinity -Infinity -> NaN i")
	if !ok {
		t.Fatal("expected the line to parse")
	}
	if c.Operation != "d128_add_ctx" {
		t.Errorf("operation = %q, want d128_add_ctx", c.Operation)
	}
	if c.Rounding != "TIES_TO_EVEN" {
		t.Errorf("rounding = %q, want TIES_TO_EVEN", c.Rounding)
	}
	if len(c.Operands) != 2 || c.Operands[0] != "Infinity" || c.Operands[1] != "-Infinity" {
		t.Errorf("operands = %v, want [Infinity -Infinity]", c.Operands)
	}
	if c.Result != "NaN" {
		t.Errorf("result = %q, want NaN", c.Result)
	}
	if c.ExpectedFlags != "i" {
		t.Errorf("flags = %q, want i", c.ExpectedFlags)
	}

	if _, ok := ParseNativeLine("// a comment"); ok {
		t.Error("comment line should not parse")
	}
	if _, ok := ParseNativeLine("   "); ok {
		t.Error("blank line should not parse")
	}
}

func TestFlagLettersCanonical(t *testing.T) {
	if got := FlagLettersCanonical("ix"); got != "xi" {
		t.Errorf("FlagLettersCanonical(ix) = %q, want xi", got)
	}
	if got := FlagLettersCanonical(""); got != "" {
		t.Errorf("FlagLettersCanonical(\"\") = %q, want empty", got)
	}
	if got := FlagLettersCanonical("zoux"); got != "xuoz" {
		t.Errorf("FlagLettersCanonical(zoux) = %q, want xuoz", got)
	}
}

// TestHarnessSkeleton is the Phase 0 gate: the harness loads real native cases,
// dispatches them, and reports zero FAILURES. Every operation is Unsupported
// until its dispatch arm lands in a later phase, so the run executes 0 vectors
// (none Pass/Fail) while proving the load→dispatch→tally pipeline compiles and
// runs end-to-end over real corpus data.
func TestHarnessSkeleton(t *testing.T) {
	path := filepath.Join(resourcesDir(), "native", "add.txt")
	cases, err := LoadNativeFile(path)
	if err != nil {
		t.Fatalf("load native add.txt: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("expected to parse native cases, got 0")
	}

	r := Run(cases)
	if r.Failed != 0 {
		t.Fatalf("phase 0 expects 0 failures, got %d: %v", r.Failed, r.Failures)
	}
	if r.Passed != 0 {
		t.Fatalf("phase 0 expects 0 passes (no arms wired), got %d", r.Passed)
	}
	if r.Unsupported != r.Total {
		t.Fatalf("phase 0 expects all %d cases Unsupported, got %d", r.Total, r.Unsupported)
	}
	t.Logf("phase 0 harness OK: %d native cases parsed, %d unsupported, %d failed",
		r.Total, r.Unsupported, r.Failed)
}
