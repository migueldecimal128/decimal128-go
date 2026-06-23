package decimal128

// The Rosetta conformance harness (CrossPlatformArchitecture.md §2.4), as
// white-box test-only code so it can dispatch to the unexported engine —
// mirroring the Rust in-crate harness. Phase 0/1 stand up the case model, the
// native-corpus parser, and a zero-arm dispatch; operator arms light up as the
// core lands in later phases. As test scaffolding this code is exempt from the
// §3 constraint regime (switch etc. used freely).

import (
	"bufio"
	"os"
	"strings"
)

// source selects the corpus dialect.
type source int

const (
	srcDectest source = iota
	srcFptest
	srcIntel
	srcNative
)

// rosettaCase is one parsed, in-scope conformance case. Operand and result
// strings stay verbatim; decoding is the dispatch layer's job.
type rosettaCase struct {
	src           source
	text          string
	operation     string
	operands      []string
	result        string
	expectedFlags string // canonical xuozi
	rounding      string // canonical rounding token
	traps         string // fptest trap column; empty elsewhere
}

// outcome is the result of running one case.
type outcome int

const (
	outPass outcome = iota
	outFail
	outUnsupported
)

func (o outcome) String() string {
	switch o {
	case outPass:
		return "PASS"
	case outFail:
		return "FAIL"
	case outUnsupported:
		return "UNSUPPORTED"
	default:
		return "?"
	}
}

// flagLettersCanonical returns the xuozi-ordered subset of exception letters in
// s (inexact, underflow, overflow, divByZero, invalid). Unknown chars ignored.
func flagLettersCanonical(s string) string {
	var x, u, o, z, i bool
	for _, ch := range s {
		switch ch {
		case 'x':
			x = true
		case 'u':
			u = true
		case 'o':
			o = true
		case 'z':
			z = true
		case 'i':
			i = true
		}
	}
	var b strings.Builder
	if x {
		b.WriteByte('x')
	}
	if u {
		b.WriteByte('u')
	}
	if o {
		b.WriteByte('o')
	}
	if z {
		b.WriteByte('z')
	}
	if i {
		b.WriteByte('i')
	}
	return b.String()
}

// dispatch runs one case against the engine. Per-operation arms (keyed on
// operation) are added as operators land. Phase 1: every case is Unsupported.
func dispatch(c *rosettaCase) (outcome, string) {
	_ = c
	return outUnsupported, "no dispatch arm wired"
}

// rosettaResult tallies a harness run.
type rosettaResult struct {
	total       int
	passed      int
	failed      int
	unsupported int
	failures    []string
}

const maxRecordedFailures = 50

// parseNativeLine parses one native-corpus line. ok=false for blank/"//" lines.
// Format: op ROUNDING operand... -> result [flagletters]
func parseNativeLine(line string) (rosettaCase, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return rosettaCase{}, false
	}
	arrow := strings.Index(line, "->")
	if arrow < 0 {
		return rosettaCase{}, false
	}
	left := strings.Fields(line[:arrow])
	right := strings.Fields(line[arrow+len("->"):])
	if len(left) < 2 || len(right) < 1 {
		return rosettaCase{}, false
	}
	c := rosettaCase{
		src:       srcNative,
		text:      line,
		operation: left[0],
		rounding:  left[1],
		operands:  append([]string(nil), left[2:]...),
		result:    right[0],
	}
	if len(right) > 1 {
		c.expectedFlags = flagLettersCanonical(strings.Join(right[1:], ""))
	}
	return c, true
}

// loadNativeFile reads all native-corpus cases from a file.
func loadNativeFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cases []rosettaCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		if c, ok := parseNativeLine(sc.Text()); ok {
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

// runRosetta dispatches each case and tallies outcomes. Unsupported cases are
// counted but are not failures.
func runRosetta(cases []rosettaCase) rosettaResult {
	var r rosettaResult
	r.total = len(cases)
	for i := range cases {
		out, diag := dispatch(&cases[i])
		switch out {
		case outPass:
			r.passed++
		case outFail:
			r.failed++
			if len(r.failures) < maxRecordedFailures {
				r.failures = append(r.failures, cases[i].text+" :: "+diag)
			}
		case outUnsupported:
			r.unsupported++
		}
	}
	return r
}
