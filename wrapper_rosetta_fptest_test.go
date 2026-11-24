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
func fptestRoundingToken(tok string, src source) (string, bool) {
	switch tok {
	case "h>":
		// Cairo spells ties-to-away `h>`; fptest has no such token.
		return "TIES_TO_AWAY", src == srcCairo
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
	return parseFptestSourceLine(line, srcFptest)
}

// parseFptestSourceLine parses one fptest or cairo line (the two share this
// syntax; the source picks the map and the rounding-token dialect).
func parseFptestSourceLine(line string, src source) (rosettaCase, bool) {
	mapName := "fptest"
	if src == srcCairo {
		mapName = "cairo"
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "--") {
		return rosettaCase{}, false
	}
	tokens := strings.Fields(trimmed)
	if len(tokens) < 2 {
		return rosettaCase{}, false
	}
	// Coarse width pre-filter: only lines of fptest.map's `format` (d128).
	format := sharedScope(mapName, "format")
	if format == "" || !strings.HasPrefix(tokens[0], format) {
		return rosettaCase{}, false
	}
	op := tokens[0][len(format):]
	mapped, ok := sharedOpFor(mapName, op)
	if !ok {
		return rosettaCase{}, false
	}

	rounding, ok := fptestRoundingToken(tokens[1], src)
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

	return rosettaCase{
		src:           src,
		text:          line,
		token:         op,
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
	return loadFptestSourceFile(path, srcFptest)
}

// loadCairoFile reads all d128 cairo cases from a file.
func loadCairoFile(path string) ([]rosettaCase, error) {
	return loadFptestSourceFile(path, srcCairo)
}

func loadFptestSourceFile(path string, src source) ([]rosettaCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []rosettaCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		if c, ok := parseFptestSourceLine(sc.Text(), src); ok {
			cases = append(cases, c)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}
