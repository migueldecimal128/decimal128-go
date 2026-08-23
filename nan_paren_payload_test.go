package decimal128

import (
	"strings"
	"testing"
)

// The parenthesised NaN payload -- NanParenPayloadWorkOrder.md §5.
//
// "NaN(123)" is an alternative spelling of "NaN123", positional and balanced:
// the '(' must follow the word immediately, at least one digit must appear,
// and the ')' must be the last character of the input. The two spellings are
// required to produce bit-identical values, so every acceptance case here
// asserts against its bare twin rather than against a second table of expected
// encodings -- the forms cannot drift apart.
//
// The reject set fails as psFailureParsingNan, which the boundary API surfaces
// only as a false ok; the status does not cross d128_parse.

// Parenthesised spelling paired with the bare spelling it must equal.
var nanParenEquivalentPairs = [][2]string{
	{"NaN(123)", "NaN123"},
	{"sNaN(5)", "sNaN5"},
	{"-qNaN(7)", "-qNaN7"},
	{"nan(0)", "nan0"},
	{"NaN(0001)", "NaN0001"},
	{"-sNaN(45)", "-sNaN45"},
}

var nanParenRejects = []string{
	"NaN()",      // no digit inside the parentheses
	"NaN(",       // unbalanced, no digit
	"NaN(1",      // unbalanced
	"NaN(1]",     // mismatched closer
	"NaN(1)2(3)", // the legacy base read this as payload 123; not this form
	"NaN(1)x",    // ')' is not the last character
	"NaN)1(",     // reversed
	"NaN (1)",    // '(' does not follow the word immediately
	"NaN[1]",     // square brackets are not this form
	"NaN{1}",     // curly braces are not this form
	"NaN(abc)",   // the payload alphabet is decimal digits, not strtod's
	"NaN(0x1c8)", // likewise
	"NaN(1_2)",   // the underscore separator does not reach NaN payloads
}

// The spellings the restructured digit loop must not regress.
var nanParenBareStillAccepted = []string{
	"NaN", "nan", "-NaN", "qNaN", "sNaN", "-sNaN",
	"NaN123", "qNaN7", "sNaN5", "-sNaN45", "NaN0",
}

func TestNanParenFormEqualsBareForm(t *testing.T) {
	for _, pair := range nanParenEquivalentPairs {
		paren, okP := d128_parse(pair[0])
		bare, okB := d128_parse(pair[1])
		if !okP {
			t.Errorf("d128_parse(%q) expected to succeed", pair[0])
			continue
		}
		if !okB {
			t.Errorf("d128_parse(%q) expected to succeed", pair[1])
			continue
		}
		if !d128_bitwiseEQ(paren, bare) {
			t.Errorf("%q and %q must encode identically", pair[0], pair[1])
		}
	}
}

func TestNanParenRejectsMalformed(t *testing.T) {
	for _, input := range nanParenRejects {
		if _, ok := d128_parse(input); ok {
			t.Errorf("d128_parse(%q) expected to fail, but succeeded", input)
		}
	}
}

func TestNanParenBareSpellingsStillParse(t *testing.T) {
	for _, input := range nanParenBareStillAccepted {
		if _, ok := d128_parse(input); !ok {
			t.Errorf("d128_parse(%q) expected to succeed", input)
		}
	}
}

// Rendering never emits parentheses -- a value parsed from the parenthesised
// form prints back in the bare form.
func TestNanParenRendersBare(t *testing.T) {
	cases := [][2]string{
		{"NaN(123)", "NaN123"},
		{"-sNaN(45)", "-sNaN45"},
		{"NaN(0001)", "NaN1"},
		{"NaN(0)", "NaN"},
	}
	for _, c := range cases {
		got := d128_format(d128_parseOrNaN(c[0]), FORMAT_AUTO)
		if got != c[1] {
			t.Errorf("format(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

// The 33-digit limit counts digits AS WRITTEN, inside the parentheses exactly
// as it counts bare ones: 34 digits is non-canonical and reads as payload zero
// (IEEE 754-2019 §3.5.2).
func TestNanParenDigitLimitCountsAsWritten(t *testing.T) {
	nines33 := strings.Repeat("9", 33)
	nines34 := strings.Repeat("9", 34)

	paren33, ok1 := d128_parse("NaN(" + nines33 + ")")
	bare33, ok2 := d128_parse("NaN" + nines33)
	paren34, ok3 := d128_parse("NaN(" + nines34 + ")")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("33/34-digit paren payloads must parse: %v %v %v", ok1, ok2, ok3)
	}
	if !d128_bitwiseEQ(paren33, bare33) {
		t.Error("33-digit paren payload must equal its bare twin")
	}
	if p := d128_nanPayload(paren33); p.dw0 == 0 && p.dw1 == 0 {
		t.Error("33-digit payload must be kept")
	}
	if p := d128_nanPayload(paren34); p.dw0 != 0 || p.dw1 != 0 {
		t.Error("34-digit payload is non-canonical and must read as zero")
	}
}
