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

// rosettaSource runs one runner over one source: every file the shared data
// selects (files.index), reported and checked against expected.tsv
// (rosetta_shared_test.go).
func rosettaSource(t *testing.T, runner, source string, dispatchFn func(*rosettaCase) (outcome, string)) {
	rep := newRosettaReport(runner, source)
	for _, name := range sharedFiles(source) {
		path := filepath.Join(resourcesDir(), source, name)
		var cases []rosettaCase
		var err error
		switch source {
		case "dectest":
			cases, err = loadDectestFile(path)
		case "fptest":
			cases, err = loadFptestFile(path)
		case "intel":
			cases, err = loadIntelFile(path)
		}
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		rep.file(t, name, runCases(dropExcluded(cases, runner, source), dispatchFn))
	}
	rep.finish(t)
	rep.check(t)
}

// rosettaNative runs one runner over the native files it selects (native.index
// minus native.exclude), each checked against its own row (no source total,
// R-4); the selection must be exactly expected.tsv's native rows.
func rosettaNative(t *testing.T, runner string, dispatchFn func(*rosettaCase) (outcome, string)) {
	rep := newRosettaReport(runner, "native")
	selected := map[string]bool{}
	for _, p := range nativeCorpusFiles(t, runner) {
		name := filepath.Base(p)
		cases, err := loadNativeFile(p)
		if err != nil {
			t.Fatalf("native %s: %v", runner, err)
		}
		rep.file(t, name, runCases(dropExcluded(cases, runner, "native"), dispatchFn))
		selected[name] = true
	}
	rep.checkRows(selected)
	rep.check(t)
}

// dropExcluded removes the cases whose token runner.exclude lists for runner.
func dropExcluded(cases []rosettaCase, runner, source string) []rosettaCase {
	out := cases[:0:0]
	for _, c := range cases {
		if !sharedExcludes(runner, source, c.token) {
			out = append(out, c)
		}
	}
	return out
}

func TestRosettaCoreDectest(t *testing.T) { rosettaSource(t, "core", "dectest", dispatch) }
func TestRosettaCoreFptest(t *testing.T)  { rosettaSource(t, "core", "fptest", dispatch) }
func TestRosettaCoreIntel(t *testing.T)   { rosettaSource(t, "core", "intel", dispatch) }
func TestRosettaCoreNative(t *testing.T)  { rosettaNative(t, "core", dispatch) }
