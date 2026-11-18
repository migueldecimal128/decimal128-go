package decimal128

// Rosetta — parser smoke tests: each source parses its real corpus into
// canonical cases, every included operator is actually reached, every produced
// op is an included one (no stray keys), and operands/result survive.

import (
	"path/filepath"
	"testing"
)

func rosettaParsersCheck(t *testing.T, source string, src source, load func(string) ([]rosettaCase, error), files []string) {
	t.Helper()
	var cases []rosettaCase
	for _, f := range files {
		cs, err := load(filepath.Join(resourcesDir(), source, f))
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		cases = append(cases, cs...)
	}
	if len(cases) == 0 {
		t.Fatalf("%s produced no Rosetta cases", source)
	}
	produced := map[string]bool{}
	for _, c := range cases {
		if c.src != src || len(c.operands) == 0 || c.result == "" || c.rounding == "" {
			t.Errorf("malformed case: %s", c.text)
		}
		produced[c.operation] = true
	}
	expected := map[string]bool{}
	for _, canonical := range sharedData().maps[source].include {
		if tok, ok := tokenForCanonical(source, canonical); ok {
			expected[tok] = true
		}
	}
	// No stray ops, and every included operator is reached by the corpus.
	for op := range produced {
		if !expected[op] {
			t.Errorf("op not in include map: %s", op)
		}
	}
	for op := range expected {
		if !produced[op] {
			t.Errorf("included operator produced no cases: %s", op)
		}
	}
}

func TestRosettaParsersFptestParserEmitsCanonicalCases(t *testing.T) {
	rosettaParsersCheck(t, "fptest", srcFptest, loadFptestFile, sharedFiles("fptest"))
}

func TestRosettaParsersIntelParserEmitsCanonicalCases(t *testing.T) {
	rosettaParsersCheck(t, "intel", srcIntel, loadIntelFile, []string{"readtest.in"})
}

func TestRosettaParsersDectestParserEmitsCanonicalCases(t *testing.T) {
	rosettaParsersCheck(t, "dectest", srcDectest, loadDectestFile, sharedFiles("dectest"))
}
