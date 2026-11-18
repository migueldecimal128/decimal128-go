package decimal128

// Rosetta — what this port checks about the shared run set
// (RosettaHarnessAlignmentWorkOrder.md R-2).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sharedDataLoads: files.index, the maps, the skips, runner.exclude
// and expected.tsv load, and every include value has a rosettaCanonical entry
// (RosettaHarnessAlignmentWorkOrder.md R-2). The partition itself is checked
// once, in resources/rosetta/check_rosetta.py.
func TestRosettaCoverageSharedDataLoads(t *testing.T) {
	if errs := sharedData().errors; len(errs) != 0 {
		t.Fatalf("shared Rosetta data:\n  %s", strings.Join(errs, "\n  "))
	}
	for _, src := range rosettaMappedSources {
		if len(sharedFiles(src)) == 0 {
			t.Errorf("files.index lists no %s files", src)
		}
		if len(sharedData().maps[src].include) == 0 {
			t.Errorf("%s.map includes nothing", src)
		}
	}
	// Every dispatch token an include value maps to has an arm with an arity,
	// so an included operator can't reach the runner unrunnable (Go dispatches
	// on strings; Swift's compiler checks this through CanonicalOp).
	n := 0
	for _, src := range rosettaMappedSources {
		for tok, canonical := range sharedData().maps[src].include {
			op, _ := tokenForCanonical(src, canonical)
			n++
			if _, ok := rosettaArity[op]; !ok {
				t.Errorf("%s: %s -> %s -> %s has no arity", src, tok, canonical, op)
			}
		}
	}
	if n == 0 {
		t.Fatal("no include targets")
	}
}

// dectestScopeMatchesParser: the dectest loader applies
// dectest.map's `rounding` and `null-operand` scope rules itself; they must be
// exactly the data's. (`precision` it reads from the data, as the fptest parser
// does its `format`.)
func TestRosettaCoverageDectestScopeMatchesParser(t *testing.T) {
	listed := map[string]bool{}
	for _, m := range strings.Split(sharedScope("dectest", "rounding"), ",") {
		listed[m] = true
	}
	for _, m := range []string{"ceiling", "down", "floor", "half_down", "half_even", "half_up", "up", "05up"} {
		r, ok := parseDectestRounding("rounding: " + m)
		runs := ok && r != roundingUnsupported
		if runs != listed[m] {
			t.Errorf("dectest rounding %q: dectest.map lists it=%v, the parser runs it=%v", m, listed[m], runs)
		}
	}
	if got := sharedScope("dectest", "null-operand"); got != "#" {
		t.Errorf("dectest.map null-operand is %q, but the loader drops a bare '#'", got)
	}
	f := filepath.Join(t.TempDir(), "hash.decTest")
	if err := os.WriteFile(f, []byte("rounding: half_even\nt1 add # 1 -> NaN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cases, err := loadDectestFile(f); err != nil || len(cases) != 0 {
		t.Errorf("the loader no longer drops a bare '#' operand: %d cases, err %v", len(cases), err)
	}
	if got := sharedScope("dectest", "precision"); got != "34" {
		t.Errorf("dectest.map precision is %q, not 34", got)
	}
}
