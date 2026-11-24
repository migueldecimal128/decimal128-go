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
		case "cairo":
			cases, err = loadCairoFile(path)
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
// R-4).
func rosettaNative(t *testing.T, runner string, dispatchFn func(*rosettaCase) (outcome, string)) {
	rep := newRosettaReport(runner, "native")
	for _, p := range nativeCorpusFiles(t, runner) {
		cases, err := loadNativeFile(p)
		if err != nil {
			t.Fatalf("native %s: %v", runner, err)
		}
		rep.file(t, filepath.Base(p), runCases(dropExcluded(cases, runner, "native"), dispatchFn))
	}
	rep.check(t)
}

// rosettaNativeSelection: the native files a runner runs are exactly
// expected.tsv's native rows.
func rosettaNativeSelection(t *testing.T, runner string) {
	rep := newRosettaReport(runner, "native")
	selected := map[string]bool{}
	for _, p := range nativeCorpusFiles(t, runner) {
		selected[filepath.Base(p)] = true
	}
	rep.checkRows(selected)
	rep.checkRowsOnly(t)
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
