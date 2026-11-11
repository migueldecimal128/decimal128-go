package decimal128

// D48 gap-line identity (D48GoWorkOrder.md Phase 3 / 4 gate; Swift TestD48GapLines,
// Zig d48_gap_lines_test.zig). The sweep files in resources/rosetta/native
// comment out each line the D48 configuration doesn't get right, recording the
// Zig prototype's D48 answer:
//
//	// gap: <line with the correct result> | d48: <D48 result>
//	// pow-cohort: …   // compound-1+x: …   (punchlist bugs, same note shape)
//
// Go must give exactly the recorded D48 answer on every such line (bitwise:
// value and cohort). A difference -- including a line Go now gets right -- is
// a bug in one of the ports.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var gapLineModes = map[string]round{
	"TIES_TO_EVEN": ROUND_TIES_TO_EVEN, "TIES_TO_AWAY": ROUND_TIES_TO_AWAY,
	"TOWARD_ZERO": ROUND_TOWARD_ZERO, "TOWARD_POSITIVE": ROUND_TOWARD_POSITIVE,
	"TOWARD_NEGATIVE": ROUND_TOWARD_NEGATIVE,
}

// gapLineEvaluate runs one native-dialect line's operation through the core
// _rnd entry.
func gapLineEvaluate(t *testing.T, op string, mode round, args []string) d128 {
	t.Helper()
	parse := func(s string) d128 {
		d, ok := d128_parse(s)
		if !ok {
			t.Fatalf("bad operand %q", s)
		}
		return d
	}
	atoi := func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("bad n %q", s)
		}
		return n
	}
	x := parse(args[0])
	switch op {
	case "d128_exp_ctx":
		return d128_exp_rnd(x, mode)
	case "d128_exp10_ctx":
		return d128_exp10_rnd(x, mode)
	case "d128_ln_ctx":
		return d128_ln_rnd(x, mode)
	case "d128_log10_ctx":
		return d128_log10_rnd(x, mode)
	case "d128_pow_ctx":
		return d128_pow_rnd(x, parse(args[1]), mode)
	case "d128_pown_ctx":
		return d128_pown_rnd(x, atoi(args[1]), mode)
	case "d128_rootn_ctx":
		return d128_rootn_rnd(x, atoi(args[1]), mode)
	case "d128_compound_ctx":
		return d128_compound_rnd(x, atoi(args[1]), mode)
	}
	t.Fatalf("unknown op %s", op)
	return d128{}
}

func TestD48GapLinesEveryRecordedD48AnswerMatches(t *testing.T) {
	tags := []string{"gap", "pow-cohort", "compound-1+x"}
	// Swift / C / Rust / Zig: gap 49 / 350 / 5, pow-cohort 238 / 36 / 0 (the 16
	// pow-cohort rootn lines are active since RootnGeneralNWorkOrder.md). The 1,330
	// compound-1+x lines moved to transcendental_compound.txt
	// (CompoundD48OnePlusXWorkOrder.md CX-6), where they are active.
	// transcendental_large_n.txt: 21 gap lines, the large-n route's near-1 family
	// (PownLargeNWorkOrder.md LN-6). transcendental_rootn_sweep.txt: 54 gap lines, the
	// general-n rootn route's near-1 family. 753 in all.
	expected := []struct {
		file   string
		counts [3]int
	}{
		{"transcendental_sweep.txt", [3]int{49, 238, 0}},
		{"transcendental_stress.txt", [3]int{350, 36, 0}},
		{"transcendental_pown_rootn.txt", [3]int{5, 0, 0}},
		{"transcendental_large_n.txt", [3]int{21, 0, 0}},
		{"transcendental_rootn_sweep.txt", [3]int{54, 0, 0}},
	}
	total, diffs := 0, 0
	for _, e := range expected {
		data, err := os.ReadFile(filepath.Join(resourcesDir(), "native", e.file))
		if err != nil {
			t.Fatal(err)
		}
		var counts [3]int
		fileDiffs := 0
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "// ") {
				continue
			}
			colon := strings.IndexByte(line, ':')
			bar := strings.Index(line, " | d48: ")
			if (colon < 0) || (bar < colon) {
				continue
			}
			tag := line[3:colon]
			ti := -1
			for i, x := range tags {
				if x == tag {
					ti = i
				}
			}
			if ti < 0 {
				continue
			}
			body := strings.Fields(line[colon+1 : bar])
			d48Text := line[bar+len(" | d48: "):]
			arrow := -1
			for i, s := range body {
				if s == "->" {
					arrow = i
					break
				}
			}
			mode, ok := gapLineModes[body[1]]
			if (arrow < 0) || !ok {
				t.Fatalf("%s: bad gap line %q", e.file, line)
			}
			got := gapLineEvaluate(t, body[0], mode, body[2:arrow])
			want, ok := d128_parse(d48Text)
			if !ok {
				t.Fatalf("%s: bad d48 answer %q", e.file, line)
			}
			counts[ti]++
			if !d128_bitwiseEQ(got, want) {
				fileDiffs++
				if fileDiffs <= 20 {
					t.Errorf("%s: %s | d48: %s | go: %s", e.file, strings.Join(body, " "), d48Text,
						d128_format(got, FORMAT_AUTO))
				}
			}
		}
		t.Logf("%s recorded D48 answers: gap %d, pow-cohort %d, compound-1+x %d; %d differ",
			e.file, counts[0], counts[1], counts[2], fileDiffs)
		if counts != e.counts {
			t.Errorf("%s: counts %v, want %v", e.file, counts, e.counts)
		}
		total += counts[0] + counts[1] + counts[2]
		diffs += fileDiffs
	}
	if total != 753 {
		t.Errorf("%d recorded lines, want 753", total)
	}
	if diffs != 0 {
		t.Errorf("%d lines where Go differs from the recorded D48 answer", diffs)
	}
}
