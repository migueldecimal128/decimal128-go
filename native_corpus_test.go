package decimal128

// The native corpus files a Rosetta runner runs: every file in
// resources/rosetta/native.index, minus this port's lines in
// resources/rosetta/native.exclude (NativeCorpusExclusionWorkOrder.md). Ports
// never list native/ themselves, so excluding or re-enabling a file is a change
// in decimal128-resources only.
//
// native.exclude lines: `<port>[:<runner>] <file> <reason>`, `#` comments. A
// line without a runner applies to every runner of the port. Errors (D-4): an
// exclusion of ours naming a file not in the index, an exclusion with no
// reason, an index entry with no file on disk. Mirrors the Swift reference
// (Tests/Decimal128Tests/rosetta/NativeCorpus.swift).

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const nativeCorpusPort = "go"

type nativeExclusion struct {
	file   string
	reason string
}

type nativeSelection struct {
	run      []string
	excluded []nativeExclusion
}

type nativeLine struct {
	number int
	text   string
}

// nativeContentLines returns the lines that are not blank or # comments,
// trimmed, with 1-based numbers.
func nativeContentLines(text string) []nativeLine {
	var out []nativeLine
	for i, raw := range strings.Split(text, "\n") {
		t := strings.TrimSpace(raw)
		if (t == "") || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, nativeLine{i + 1, t})
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// selectNativeCorpus is the selection for port/runner from the two files'
// text. Pure: no disk access.
func selectNativeCorpus(index, exclude, port, runner string) (nativeSelection, error) {
	var files []string
	for _, l := range nativeContentLines(index) {
		files = append(files, l.text)
	}
	var sel nativeSelection
	var excluded []string
	for _, l := range nativeContentLines(exclude) {
		fields := strings.Fields(l.text)
		linePort, lineRunner, _ := strings.Cut(fields[0], ":")
		if (linePort != port) || ((lineRunner != "") && (lineRunner != runner)) {
			continue // another port's or another runner's line
		}
		if len(fields) < 3 {
			return sel, fmt.Errorf("native.exclude:%d: exclusion has no reason", l.number)
		}
		file := fields[1]
		if !containsString(files, file) {
			return sel, fmt.Errorf("native.exclude:%d: %s is not in native.index (stale exclusion)", l.number, file)
		}
		if !containsString(excluded, file) {
			// The reason is the rest of the line after the file field.
			rest := strings.TrimSpace(l.text[len(fields[0]):])
			reason := strings.TrimSpace(rest[len(file):])
			excluded = append(excluded, file)
			sel.excluded = append(sel.excluded, nativeExclusion{file, reason})
		}
	}
	for _, f := range files {
		if !containsString(excluded, f) {
			sel.run = append(sel.run, f)
		}
	}
	return sel, nil
}

// nativeCorpusFiles is the selection for one of this port's runners ("core" or
// "wrapper"), read from the resources submodule: the paths to run, in index
// order. Fails the test on any error; logs each exclusion with its reason (D-6).
func nativeCorpusFiles(t *testing.T, runner string) []string {
	t.Helper()
	index, err := os.ReadFile(filepath.Join(resourcesDir(), "native.index"))
	if err != nil {
		t.Fatalf("read native.index: %v", err)
	}
	exclude, err := os.ReadFile(filepath.Join(resourcesDir(), "native.exclude"))
	if err != nil {
		t.Fatalf("read native.exclude: %v", err)
	}
	sel, err := selectNativeCorpus(string(index), string(exclude), nativeCorpusPort, runner)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range sel.run {
		p := filepath.Join(resourcesDir(), "native", f)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("native.index lists %s, which is not in native/", f)
		}
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		t.Fatalf("native.index selects no files for %s:%s", nativeCorpusPort, runner)
	}
	t.Logf("native %s: %d files, %d excluded", runner, len(sel.run), len(sel.excluded))
	for _, e := range sel.excluded {
		t.Logf("    excluded %s: %s", e.file, e.reason)
	}
	return paths
}

// nativeActiveLines counts the lines of a native file that are not blank or a
// "//" comment, read independently of the loader.
func nativeActiveLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		if !isNativeNonCase(line) {
			n++
		}
	}
	return n
}

// runNativeCorpus runs the native files through one runner's dispatch, with no
// silent loss (D48GoWorkOrder.md G-10): a malformed line fails the load with
// file:line; a case the runner reports unsupported (including an op with no
// dispatch arm) is a failure; each file's case count must equal its active
// lines. Logs per-file counts and returns the total.
func runNativeCorpus(t *testing.T, runner string, paths []string,
	dispatchFn func(*rosettaCase) (outcome, string)) rosettaResult {
	t.Helper()
	var total rosettaResult
	for _, p := range paths {
		name := filepath.Base(p)
		cases, err := loadNativeFile(p)
		if err != nil {
			t.Fatalf("native %s: %v", runner, err)
		}
		if active := nativeActiveLines(t, p); len(cases) != active {
			t.Fatalf("native %s %s: %d cases loaded, %d active lines", runner, name, len(cases), active)
		}
		var r rosettaResult
		r.total = len(cases)
		for i := range cases {
			c := &cases[i]
			out, diag := dispatchFn(c)
			if out == outPass {
				r.passed++
				continue
			}
			r.failed++
			if out == outUnsupported {
				diag = "unsupported: " + diag
			}
			if len(r.failures) < maxRecordedFailures {
				r.failures = append(r.failures, fmt.Sprintf("%s:%d: %s :: %s", name, c.line, c.text, diag))
			}
		}
		t.Logf("native %-7s %-24s %5d cases, %5d passed, %d failed", runner, name, r.total, r.passed, r.failed)
		mergeResult(&total, r)
	}
	t.Logf("native %-7s TOTAL: %d files, %d cases, %d passed, %d failed",
		runner, len(paths), total.total, total.passed, total.failed)
	return total
}

// MARK: - Tests

const testNativeIndex = "# header comment\nadd.txt\npow.txt\n\npown.txt\n"

func mustSelect(t *testing.T, exclude, runner string) nativeSelection {
	t.Helper()
	sel, err := selectNativeCorpus(testNativeIndex, exclude, "go", runner)
	if err != nil {
		t.Fatal(err)
	}
	return sel
}

func TestNativeCorpusNoExclusionsRunsTheIndexInOrder(t *testing.T) {
	sel := mustSelect(t, "# nothing excluded\n", "core")
	if !reflect.DeepEqual(sel.run, []string{"add.txt", "pow.txt", "pown.txt"}) || (len(sel.excluded) != 0) {
		t.Errorf("got %v / %v", sel.run, sel.excluded)
	}
}

func TestNativeCorpusPortWideExclusionAppliesToEveryRunner(t *testing.T) {
	for _, runner := range []string{"core", "wrapper"} {
		sel := mustSelect(t, "go  pow.txt  still on D38; remove at the D48 switch\n", runner)
		if !reflect.DeepEqual(sel.run, []string{"add.txt", "pown.txt"}) {
			t.Errorf("%s: run %v", runner, sel.run)
		}
		want := []nativeExclusion{{"pow.txt", "still on D38; remove at the D48 switch"}}
		if !reflect.DeepEqual(sel.excluded, want) {
			t.Errorf("%s: excluded %v", runner, sel.excluded)
		}
	}
}

func TestNativeCorpusRunnerExclusionAppliesToThatRunnerOnly(t *testing.T) {
	ex := "go:wrapper pown.txt not exposed on the wrapper\n"
	if got := mustSelect(t, ex, "wrapper").run; !reflect.DeepEqual(got, []string{"add.txt", "pow.txt"}) {
		t.Errorf("wrapper: %v", got)
	}
	if got := mustSelect(t, ex, "core").run; !reflect.DeepEqual(got, []string{"add.txt", "pow.txt", "pown.txt"}) {
		t.Errorf("core: %v", got)
	}
}

func TestNativeCorpusOtherPortsLinesAreIgnored(t *testing.T) {
	ex := "kotlin pow.txt a Kotlin reason\ngolang pow.txt a longer name is another port\nc gone.txt another port's stale line is not ours to report\n"
	if got := mustSelect(t, ex, "core").run; !reflect.DeepEqual(got, []string{"add.txt", "pow.txt", "pown.txt"}) {
		t.Errorf("got %v", got)
	}
}

func TestNativeCorpusStaleExclusionFails(t *testing.T) {
	_, err := selectNativeCorpus(testNativeIndex, "# comment\ngo gone.txt removed long ago\n", "go", "core")
	if (err == nil) || (err.Error() != "native.exclude:2: gone.txt is not in native.index (stale exclusion)") {
		t.Errorf("got %v", err)
	}
}

func TestNativeCorpusExclusionWithoutReasonFails(t *testing.T) {
	_, err := selectNativeCorpus(testNativeIndex, "go:core pow.txt\n", "go", "core")
	if (err == nil) || (err.Error() != "native.exclude:1: exclusion has no reason") {
		t.Errorf("got %v", err)
	}
}
