package decimal128

// IEEE fptest conformance corpus (resources/rosetta/fptest/*.fptest) parser and
// loader — the Go analog of the C# RosettaTests.cs parseFptestLine. Test-only
// code, exempt from the §3 constraint regime.
//
// An fptest line is:
//
//	<op> <rounding> [<trap-flags>] <operand>... -> <result> [<result-exceptions>]
//
// where <op> is d128 followed by an arithmetic sign (+ - * /), <rounding> is a
// symbolic token (=0 =^ > < 0), the optional trap-flags column (exception
// letters) marks a trap-enabled case, results are in a raw cohort form
// (`+coeff e exp`) or a special (Q/S/+inf/-inf/#), and the trailing column is
// the raised-exception letters. Only the fptest files carry d128 lines; the
// decimal64 (`d64…`) lines are ignored.

import (
	"bufio"
	"os"
	"strings"
)

// isFptestExceptionStr reports whether s is a non-empty run of exception letters
// (used to detect the optional trap-flags column).
func isFptestExceptionStr(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != 'x' && c != 'u' && c != 'o' && c != 'z' && c != 'i' {
			return false
		}
	}
	return true
}

// fptestRoundingToken maps an fptest symbolic rounding token to the canonical
// rounding token.
func fptestRoundingToken(tok string) (string, bool) {
	switch tok {
	case "=0":
		return "TIES_TO_EVEN", true
	case "=^":
		return "TIES_TO_AWAY", true
	case ">":
		return "TOWARD_POSITIVE", true
	case "<":
		return "TOWARD_NEGATIVE", true
	case "0":
		return "TOWARD_ZERO", true
	default:
		return "", false
	}
}

// parseFptestLine parses one fptest line into a rosettaCase. ok=false for
// comments, blanks, and non-d128 (decimal64) lines.
func parseFptestLine(line string) (rosettaCase, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "--") {
		return rosettaCase{}, false
	}
	tokens := strings.Fields(trimmed)
	if len(tokens) < 2 {
		return rosettaCase{}, false
	}
	t0 := tokens[0]
	if !strings.HasPrefix(t0, "d128") {
		return rosettaCase{}, false
	}
	op := t0[len("d128"):]

	rounding, ok := fptestRoundingToken(tokens[1])
	if !ok {
		return rosettaCase{}, false
	}

	idx := 2
	traps := ""
	if idx < len(tokens) && isFptestExceptionStr(tokens[idx]) {
		traps = tokens[idx]
		idx++
	}

	operands := make([]string, 0, 3)
	for idx < len(tokens) && tokens[idx] != "->" {
		operands = append(operands, tokens[idx])
		idx++
	}
	if idx >= len(tokens) {
		return rosettaCase{}, false // no "->"
	}
	idx++ // skip "->"
	if idx >= len(tokens) {
		return rosettaCase{}, false
	}
	result := tokens[idx]
	idx++
	exceptions := ""
	if idx < len(tokens) {
		exceptions = tokens[idx]
	}

	mapped := op
	switch op {
	case "+":
		mapped = "add"
	case "-":
		mapped = "subtract"
	case "*":
		mapped = "multiply"
	case "/":
		mapped = "divide"
	}

	return rosettaCase{
		src:           srcFptest,
		text:          line,
		operation:     mapped,
		operands:      operands,
		result:        result,
		rounding:      rounding,
		expectedFlags: flagLettersCanonical(exceptions),
		traps:         traps,
	}, true
}

// loadFptestFile reads all d128 fptest cases from a file.
func loadFptestFile(path string) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []rosettaCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		if c, ok := parseFptestLine(sc.Text()); ok {
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}
