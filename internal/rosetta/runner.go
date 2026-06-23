package rosetta

import (
	"bufio"
	"os"
	"strings"
)

// Result tallies a harness run.
type Result struct {
	Total       int
	Passed      int
	Failed      int
	Unsupported int
	Failures    []string // bounded sample of "<line> :: <diag>" for failed cases
}

// maxRecordedFailures bounds the Failures sample so a wholesale regression does
// not produce an unbounded diagnostic dump.
const maxRecordedFailures = 50

// ParseNativeLine parses one native-corpus line into a Case. It returns ok=false
// for blank lines and "//" comments. The native format is:
//
//	op ROUNDING operand... -> result [flagletters]
//
// for example:
//
//	d128_add_ctx TIES_TO_EVEN Infinity -Infinity -> NaN i
//
// Operands are every token between the rounding mode and "->"; the result is the
// first token after "->", and any trailing tokens are exception-flag letters.
func ParseNativeLine(line string) (Case, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return Case{}, false
	}
	arrow := strings.Index(line, "->")
	if arrow < 0 {
		return Case{}, false
	}
	left := strings.Fields(line[:arrow])
	right := strings.Fields(line[arrow+len("->"):])
	if len(left) < 2 || len(right) < 1 {
		return Case{}, false
	}
	c := Case{
		Source:    Native,
		Text:      line,
		Operation: left[0],
		Rounding:  left[1],
		Operands:  append([]string(nil), left[2:]...),
		Result:    right[0],
	}
	if len(right) > 1 {
		c.ExpectedFlags = FlagLettersCanonical(strings.Join(right[1:], ""))
	}
	return c, true
}

// LoadNativeFile reads all native-corpus cases from a file.
func LoadNativeFile(path string) ([]Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cases []Case
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20) // some corpus lines are long
	for sc.Scan() {
		if c, ok := ParseNativeLine(sc.Text()); ok {
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

// Run dispatches each case and tallies outcomes. Unsupported cases are counted
// separately and are not failures.
func Run(cases []Case) Result {
	var r Result
	r.Total = len(cases)
	for i := range cases {
		out, diag := Dispatch(&cases[i])
		switch out {
		case Pass:
			r.Passed++
		case Fail:
			r.Failed++
			if len(r.Failures) < maxRecordedFailures {
				r.Failures = append(r.Failures, cases[i].Text+" :: "+diag)
			}
		case Unsupported:
			r.Unsupported++
		}
	}
	return r
}
