package decimal128

// d128_parse — IEEE 754-2019 §5.4.2 convertFromDecimalCharacter, strict form.
//
// The (input, result) coverage lives as native Rosetta cases (parse.txt), run
// through both the core and the public-API harness. What remains here is the
// handful of reject cases the space-delimited native grammar cannot carry: an
// operand that IS empty or contains whitespace can't be written as a single
// native token. These pin that the strict parser rejects them (returns false,
// never a NaN-on-error).

import "testing"

func TestParseResidualRejectsWhitespaceOrEmpty(t *testing.T) {
	for _, input := range []string{
		"",      // empty string
		"1 ",    // trailing space after a valid number
		"nan 1", // space between the NaN word and its payload
	} {
		if _, ok := d128_parse(input); ok {
			t.Errorf("d128_parse(%q) expected to fail (empty/whitespace), but succeeded", input)
		}
	}
}
