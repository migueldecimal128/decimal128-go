package decimal128

// Rosetta — the shared run set (RosettaHarnessAlignmentWorkOrder.md R-1..R-4).
//
// What this harness runs from the dectest, fptest and Intel corpora is data in
// the resources submodule (resources/rosetta), read here, not tables in this
// port:
//
//	files.index      the corpus files
//	<source>.map     scope filters + operator include (-> canonical op) / exclude
//	<source>.skip    per-line skips, each with a reason tag
//	runner.exclude   operators one port's runner does not run
//	expected.tsv     exact cases / skipped per port x runner x source x file
//
// The only local table is rosettaCanonical: canonical op name -> this harness's
// dispatch token (Go dispatches on strings). dectest overrides two entries:
// dectest abs and minus run value-only, as Swift's dectestValueOnly rule does
// (their dectest vectors carry parse-time flags the quiet arms don't model).
// An include value missing from the table is a load error;
// TestRosettaIncludeTargetsHaveArity checks every token it names has an arm.
// A load error never panics here: the data is then empty, every count check
// fails, and TestRosettaSharedDataLoads names it.
//
// This file also holds the shared report (R-3, R-4): every runner prints, per
// file and (except native) once per source,
//
//	ROSETTA go <runner> <source> <file> cases=<n> passed=<n> skipped=<n> failed=<n>
//	ROSETTA go <runner> <source> TOTAL cases=<n> passed=<n> skipped=<n> failed=<n>
//	ROSETTA-SKIP go <runner> <source> <count> <reason-tag>
//
// and checks each file against its expected.tsv row.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const rosettaPort = "go"

var rosettaMappedSources = []string{"dectest", "fptest", "intel"}

// rosettaCanonical maps a canonical op name to this harness's dispatch token.
var rosettaCanonical = map[string]string{
	"d128_abs":                             "d128_abs",
	"d128_add_ctx":                         "add",
	"d128_apply_ctx":                       "apply",
	"d128_compareSignaling_ctx":            "comparesig",
	"d128_compareTotal":                    "comparetotal",
	"d128_compareTotalMag":                 "comparetotmag",
	"d128_compare_ctx":                     "compare",
	"d128_copy":                            "d128_copy",
	"d128_copySign":                        "copysign",
	"d128_divideInteger_ctx":               "divideint",
	"d128_divide_ctx":                      "divide",
	"d128_exp10_ctx":                       "d128_exp10_ctx",
	"d128_exp_ctx":                         "d128_exp_ctx",
	"d128_expm1_ctx":                       "d128_expm1_ctx",
	"d128_fma_ctx":                         "fma",
	"d128_fromInt64":                       "from_int64",
	"d128_fromUInt64":                      "from_uint64",
	"d128_ieeeClass":                       "class",
	"d128_isEqual_ctx":                     "d128_isEqual_ctx",
	"d128_isFinite":                        "d128_isFinite",
	"d128_isGreaterEqualSignaling_ctx":     "d128_isGreaterEqualSignaling_ctx",
	"d128_isGreaterEqual_ctx":              "d128_isGreaterEqual_ctx",
	"d128_isGreaterSignaling_ctx":          "d128_isGreaterSignaling_ctx",
	"d128_isGreaterUnorderedSignaling_ctx": "d128_isGreaterUnorderedSignaling_ctx",
	"d128_isGreaterUnordered_ctx":          "d128_isGreaterUnordered_ctx",
	"d128_isGreater_ctx":                   "d128_isGreater_ctx",
	"d128_isInf":                           "d128_isInf",
	"d128_isLessEqualSignaling_ctx":        "d128_isLessEqualSignaling_ctx",
	"d128_isLessEqual_ctx":                 "d128_isLessEqual_ctx",
	"d128_isLessSignaling_ctx":             "d128_isLessSignaling_ctx",
	"d128_isLessUnorderedSignaling_ctx":    "d128_isLessUnorderedSignaling_ctx",
	"d128_isLessUnordered_ctx":             "d128_isLessUnordered_ctx",
	"d128_isLess_ctx":                      "d128_isLess_ctx",
	"d128_isNaN":                           "d128_isNaN",
	"d128_isNormal":                        "d128_isNormal",
	"d128_isNotEqual_ctx":                  "d128_isNotEqual_ctx",
	"d128_isNotGreaterSignaling_ctx":       "d128_isNotGreaterSignaling_ctx",
	"d128_isNotGreater_ctx":                "d128_isNotGreater_ctx",
	"d128_isNotLessSignaling_ctx":          "d128_isNotLessSignaling_ctx",
	"d128_isNotLess_ctx":                   "d128_isNotLess_ctx",
	"d128_isOrdered_ctx":                   "d128_isOrdered_ctx",
	"d128_isSignaling":                     "d128_isSignaling",
	"d128_isSigned":                        "d128_isSigned",
	"d128_isSubnormal":                     "d128_isSubnormal",
	"d128_isUnordered_ctx":                 "d128_isUnordered_ctx",
	"d128_isZero":                          "d128_isZero",
	"d128_ln_ctx":                          "d128_ln_ctx",
	"d128_lnp1_ctx":                        "d128_lnp1_ctx",
	"d128_log10_ctx":                       "d128_log10_ctx",
	"d128_logB_ctxnull":                    "logb",
	"d128_maximumMagnitudeNumber_ctx":      "maxmag",
	"d128_maximumNumber_ctx":               "max",
	"d128_minimumMagnitudeNumber_ctx":      "minmag",
	"d128_minimumNumber_ctx":               "min",
	"d128_multiply_ctx":                    "multiply",
	"d128_negate":                          "d128_negate",
	"d128_nextDown_ctxnull":                "nextminus",
	"d128_nextUp_ctxnull":                  "nextplus",
	"d128_parse_ctx":                       "parse_ctx",
	"d128_quantexp":                        "d128_quantexp",
	"d128_quantize_ctx":                    "quantize",
	"d128_remainderNear_ctx":               "remaindernear",
	"d128_remainderTrunc_ctx":              "remainder",
	"d128_roundToIntegralSignalInexact":    "tointegralx",
	"d128_roundToIntegral_ctx":             "round_integral_mode",
	"d128_round_integral_nearest_away":     "d128_round_integral_nearest_away",
	"d128_round_integral_nearest_even":     "d128_round_integral_nearest_even",
	"d128_round_integral_negative":         "d128_round_integral_negative",
	"d128_round_integral_positive":         "d128_round_integral_positive",
	"d128_round_integral_zero":             "d128_round_integral_zero",
	"d128_sameQuantum":                     "samequantum",
	"d128_scaleB_ctxnull":                  "scaleb",
	"d128_sqrt_ctx":                        "squareroot",
	"d128_stripTrailingZeros":              "reduce",
	"d128_subtract_ctx":                    "subtract",
	"d128_toEngString":                     "toeng",
	"d128_toInt64SignalInexact_ctx":        "to_int64_xmode",
	"d128_toString":                        "tosci",
	"d128_to_int64_ceil":                   "d128_to_int64_ceil",
	"d128_to_int64_floor":                  "d128_to_int64_floor",
	"d128_to_int64_int":                    "d128_to_int64_int",
	"d128_to_int64_rnint":                  "d128_to_int64_rnint",
	"d128_to_int64_rninta":                 "d128_to_int64_rninta",
	"d128_to_int64_xceil":                  "d128_to_int64_xceil",
	"d128_to_int64_xfloor":                 "d128_to_int64_xfloor",
	"d128_to_int64_xint":                   "d128_to_int64_xint",
	"d128_to_int64_xrnint":                 "d128_to_int64_xrnint",
	"d128_to_int64_xrninta":                "d128_to_int64_xrninta",
	"d128_to_uint64_ceil":                  "d128_to_uint64_ceil",
	"d128_to_uint64_floor":                 "d128_to_uint64_floor",
	"d128_to_uint64_int":                   "d128_to_uint64_int",
	"d128_to_uint64_rnint":                 "d128_to_uint64_rnint",
	"d128_to_uint64_rninta":                "d128_to_uint64_rninta",
	"d128_to_uint64_xceil":                 "d128_to_uint64_xceil",
	"d128_to_uint64_xfloor":                "d128_to_uint64_xfloor",
	"d128_to_uint64_xint":                  "d128_to_uint64_xint",
	"d128_to_uint64_xrnint":                "d128_to_uint64_xrnint",
	"d128_to_uint64_xrninta":               "d128_to_uint64_xrninta",
	"d128_totalOrder":                      "d128_totalOrder",
	"d128_totalOrderMag":                   "d128_totalOrderMag"}

// rosettaCanonicalBySource overrides rosettaCanonical for one source.
var rosettaCanonicalBySource = map[string]map[string]string{
	"dectest": {
		"d128_abs":    "abs",
		"d128_negate": "minus",
	},
}

func tokenForCanonical(source, canonical string) (string, bool) {
	if t, ok := rosettaCanonicalBySource[source][canonical]; ok {
		return t, true
	}
	t, ok := rosettaCanonical[canonical]
	return t, ok
}

type rosettaSourceMap struct {
	scope   map[string]string
	include map[string]string // corpus token -> canonical op name
	exclude map[string]string
}

type rosettaAttrSkip struct {
	tag, attribute string
	ops            map[string]bool
}

type rosettaSkips struct {
	lines map[string]string // exact trimmed line -> reason tag
	attrs []rosettaAttrSkip
}

type rosettaExpected struct{ cases, skipped int }

type rosettaData struct {
	files      map[string][]string
	maps       map[string]*rosettaSourceMap
	skips      map[string]*rosettaSkips
	exclusions []([4]string)              // port, runner ("" = every runner), source, token
	expected   map[string]rosettaExpected // "runner\tsource\tfile", this port's rows
	expKeys    []string
	errors     []string
}

var (
	rosettaDataOnce sync.Once
	rosettaShared   *rosettaData
)

func sharedData() *rosettaData {
	rosettaDataOnce.Do(func() { rosettaShared = loadSharedData() })
	return rosettaShared
}

// sharedOpFor is the dispatch token for an included corpus token of source, or
// ok=false when the token is excluded, out of scope or undecided.
func sharedOpFor(source, token string) (string, bool) {
	m := sharedData().maps[source]
	if m == nil {
		return "", false
	}
	canonical, ok := m.include[token]
	if !ok {
		return "", false
	}
	return tokenForCanonical(source, canonical)
}

func sharedScope(source, rule string) string {
	if m := sharedData().maps[source]; m != nil {
		return m.scope[rule]
	}
	return ""
}

func sharedFiles(source string) []string {
	return sharedData().files[source]
}

// sharedExcludes reports whether this port's runner does not run token of
// source (native: a canonical op name).
func sharedExcludes(runner, source, token string) bool {
	for _, e := range sharedData().exclusions {
		if e[0] == rosettaPort && e[2] == source && e[3] == token && (e[1] == "" || e[1] == runner) {
			return true
		}
	}
	return false
}

// sharedSkipTag is the reason tag of the per-line skip that drops c, or "".
func sharedSkipTag(c *rosettaCase) string {
	var src string
	switch c.src {
	case srcDectest:
		src = "dectest"
	case srcIntel:
		src = "intel"
	case srcFptest:
		src = "fptest"
	default:
		return ""
	}
	sk := sharedData().skips[src]
	if sk == nil {
		return ""
	}
	t := strings.TrimSpace(c.text)
	if len(sk.attrs) > 0 {
		tokens := strings.Fields(t)
		for _, a := range sk.attrs {
			if len(tokens) > 0 && a.ops[tokens[0]] && containsString(tokens, a.attribute) {
				return a.tag
			}
		}
	}
	return sk.lines[t]
}

// MARK: - Load

func loadSharedData() *rosettaData {
	d := &rosettaData{
		files:    map[string][]string{},
		maps:     map[string]*rosettaSourceMap{},
		skips:    map[string]*rosettaSkips{},
		expected: map[string]rosettaExpected{},
	}
	read := func(name string, optional bool) (string, bool) {
		b, err := os.ReadFile(filepath.Join(resourcesDir(), name))
		if err != nil {
			if !optional {
				d.errors = append(d.errors, fmt.Sprintf("%s: %v", name, err))
			}
			return "", false
		}
		return string(b), true
	}
	if text, ok := read("files.index", false); ok {
		for _, l := range dataLines(text) {
			if f, ok := tabFields(l.text, 2); ok {
				d.files[f[0]] = append(d.files[f[0]], f[1])
			} else {
				d.errors = append(d.errors, fmt.Sprintf("files.index:%d: expected <source>\\t<file>", l.n))
			}
		}
	}
	for _, src := range rosettaMappedSources {
		m := &rosettaSourceMap{scope: map[string]string{}, include: map[string]string{}, exclude: map[string]string{}}
		d.maps[src] = m
		if text, ok := read(src+".map", false); ok {
			for _, l := range dataLines(text) {
				at := fmt.Sprintf("%s.map:%d", src, l.n)
				f, ok := tabFields(l.text, 3)
				if !ok {
					d.errors = append(d.errors, at+": expected 3 tab-separated fields")
					continue
				}
				switch f[0] {
				case "scope":
					m.scope[f[1]] = f[2]
				case "include":
					if _, ok := tokenForCanonical(src, f[2]); !ok {
						d.errors = append(d.errors, fmt.Sprintf("%s: %s includes %s, which has no entry in rosettaCanonical", at, f[1], f[2]))
						continue
					}
					m.include[f[1]] = f[2]
				case "exclude":
					m.exclude[f[1]] = f[2]
				default:
					d.errors = append(d.errors, fmt.Sprintf("%s: unknown entry kind %q", at, f[0]))
				}
			}
		}
		sk := &rosettaSkips{lines: map[string]string{}}
		d.skips[src] = sk
		if text, ok := read(src+".skip", true); ok { // fptest has no skip file
			for _, l := range dataLines(text) {
				at := fmt.Sprintf("%s.skip:%d", src, l.n)
				switch {
				case strings.HasPrefix(l.text, "="):
					// a reason definition; the tag is what we report
				case strings.HasPrefix(l.text, "line\t"):
					if f, ok := tabFields(l.text, 3); ok {
						sk.lines[f[2]] = f[1]
					} else {
						d.errors = append(d.errors, at+": expected line\\t<tag>\\t<line>")
					}
				case strings.HasPrefix(l.text, "attr\t"):
					if f, ok := tabFields(l.text, 4); ok {
						ops := map[string]bool{}
						for _, o := range strings.Split(f[2], ",") {
							ops[o] = true
						}
						sk.attrs = append(sk.attrs, rosettaAttrSkip{tag: f[1], ops: ops, attribute: f[3]})
					} else {
						d.errors = append(d.errors, at+": expected attr\\t<tag>\\t<ops>\\t<attribute>")
					}
				default:
					d.errors = append(d.errors, at+": unknown entry")
				}
			}
		}
	}
	if text, ok := read("runner.exclude", false); ok {
		for _, l := range dataLines(text) {
			f, ok := tabFields(l.text, 4)
			if !ok {
				d.errors = append(d.errors, fmt.Sprintf("runner.exclude:%d: expected 4 tab-separated fields", l.n))
				continue
			}
			port, runner, _ := strings.Cut(f[0], ":")
			d.exclusions = append(d.exclusions, [4]string{port, runner, f[1], f[2]})
		}
	}
	if text, ok := read("expected.tsv", false); ok {
		for _, l := range dataLines(text) {
			f, ok := tabFields(l.text, 6)
			if !ok {
				d.errors = append(d.errors, fmt.Sprintf("expected.tsv:%d: expected 6 tab-separated fields", l.n))
				continue
			}
			if f[0] != rosettaPort {
				continue
			}
			cases, err1 := strconv.Atoi(f[4])
			skipped, err2 := strconv.Atoi(f[5])
			if err1 != nil || err2 != nil {
				d.errors = append(d.errors, fmt.Sprintf("expected.tsv:%d: counts are not integers", l.n))
				continue
			}
			key := f[1] + "\t" + f[2] + "\t" + f[3]
			d.expected[key] = rosettaExpected{cases, skipped}
			d.expKeys = append(d.expKeys, key)
		}
		if len(d.expected) == 0 {
			d.errors = append(d.errors, "expected.tsv: no rows for port "+rosettaPort)
		}
	}
	return d
}

type dataLine struct {
	n    int
	text string
}

// dataLines are the lines that are not blank or # comments; the text keeps its
// tabs (the field separator).
func dataLines(text string) []dataLine {
	var out []dataLine
	for i, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if t != "" && !strings.HasPrefix(t, "#") {
			out = append(out, dataLine{i + 1, l})
		}
	}
	return out
}

// tabFields splits on tabs into exactly count fields (the last keeps any
// further tabs); ok=false when there are fewer or one is blank.
func tabFields(line string, count int) ([]string, bool) {
	parts := strings.SplitN(line, "\t", count)
	if len(parts) != count {
		return nil, false
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, false
		}
	}
	return parts, true
}

// MARK: - Report

type rosettaReport struct {
	runner, source string
	mismatches     []string
	reported       map[string]bool
	reasons        map[string]int
	total          rosettaResult
}

func newRosettaReport(runner, source string) *rosettaReport {
	r := &rosettaReport{runner: runner, source: source, reported: map[string]bool{}, reasons: map[string]int{}}
	for _, e := range sharedData().errors {
		r.mismatches = append(r.mismatches, "shared data: "+e)
	}
	return r
}

// file records and prints one file's outcome and checks it against expected.tsv.
func (r *rosettaReport) file(t *testing.T, name string, res rosettaResult) {
	t.Logf("ROSETTA %s %s %s %s cases=%d passed=%d skipped=%d failed=%d",
		rosettaPort, r.runner, r.source, name, res.total, res.passed, res.skipped, res.failed)
	mergeResult(&r.total, res)
	for tag, n := range res.skipReasons {
		r.reasons[tag] += n
	}
	r.reported[name] = true
	want, ok := sharedData().expected[r.runner+"\t"+r.source+"\t"+name]
	switch {
	case !ok:
		r.mismatches = append(r.mismatches, fmt.Sprintf("%s: ran, but expected.tsv has no %s %s %s row", name, rosettaPort, r.runner, r.source))
	case want.cases != res.total || want.skipped != res.skipped:
		r.mismatches = append(r.mismatches, fmt.Sprintf("%s: cases=%d skipped=%d, expected cases=%d skipped=%d",
			name, res.total, res.skipped, want.cases, want.skipped))
	}
}

// finish prints the source total and its skip reasons; every expected row of
// this runner and source must have been reported.
func (r *rosettaReport) finish(t *testing.T) {
	t.Logf("ROSETTA %s %s %s TOTAL cases=%d passed=%d skipped=%d failed=%d",
		rosettaPort, r.runner, r.source, r.total.total, r.total.passed, r.total.skipped, r.total.failed)
	tags := make([]string, 0, len(r.reasons))
	for tag := range r.reasons {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		t.Logf("ROSETTA-SKIP %s %s %s %d %s", rosettaPort, r.runner, r.source, r.reasons[tag], tag)
	}
	r.checkRows(r.reported)
}

// checkRows: every expected.tsv row of this runner and source is in files, and
// each of files has a row.
func (r *rosettaReport) checkRows(files map[string]bool) {
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, f := range names {
		if _, ok := sharedData().expected[r.runner+"\t"+r.source+"\t"+f]; !ok {
			r.mismatches = append(r.mismatches, fmt.Sprintf("%s: selected, but expected.tsv has no %s %s %s row", f, rosettaPort, r.runner, r.source))
		}
	}
	prefix := r.runner + "\t" + r.source + "\t"
	rows := 0
	for _, key := range sharedData().expKeys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rows++
		if f := key[len(prefix):]; !files[f] {
			r.mismatches = append(r.mismatches, fmt.Sprintf("expected.tsv row never run: %s %s %s", r.runner, r.source, f))
		}
	}
	if rows == 0 {
		r.mismatches = append(r.mismatches, fmt.Sprintf("expected.tsv has no %s %s %s rows", rosettaPort, r.runner, r.source))
	}
}

// check is the test-side verdict: cases ran, no failures, no count mismatches.
func (r *rosettaReport) check(t *testing.T) {
	for _, m := range r.mismatches {
		t.Logf("ROSETTA-MISMATCH %s %s %s %s", rosettaPort, r.runner, r.source, m)
	}
	if r.total.total == 0 {
		t.Fatalf("%s %s: no Rosetta cases ran", r.runner, r.source)
	}
	if r.total.failed != 0 {
		t.Fatalf("%s %s: %d Rosetta cases failed: %v", r.runner, r.source, r.total.failed, sampleFailures(r.total))
	}
	if len(r.mismatches) != 0 {
		t.Fatalf("%s %s: %d mismatch(es) against expected.tsv (see ROSETTA-MISMATCH)", r.runner, r.source, len(r.mismatches))
	}
}
