package decimal128

// Round-trip tests for d128_to_bid128 / d128_from_bid128, bit-exact known
// vectors in the Intel BID hex format, and the hex parser itself.

import "testing"

// (name, input parsed into a d128)
var bid128RoundTripCases = [][2]string{
	{"posZero_q0", "0"},
	{"negZero_q0", "-0"},
	{"posZero_q_neg1", "0.0"},
	{"posZero_q_neg2", "0.00"},
	{"posZero_q_min", "0E-6176"},
	{"posZero_q_max", "0E+6111"},
	{"negZero_q_min", "-0E-6176"},
	{"negZero_q_max", "-0E+6111"},
	{"one", "1"},
	{"negOne", "-1"},
	{"two", "2"},
	{"seven", "7"},
	{"oneDotZero", "1.0"},
	{"oneDotZeroZero", "1.00"},
	{"tenE5", "1E+5"},
	{"negTenE5", "-1E+5"},
	{"coeff_bit113_set", "10384593717069655257060992658440192"},
	{"eighteen_nines", "999999999999999999"},
	{"nineteen_ones", "1111111111111111111"},
	{"thirtyThree_nines", "999999999999999999999999999999999"},
	{"max34_nines", "9999999999999999999999999999999999"},
	{"neg_max34_nines", "-9999999999999999999999999999999999"},
	{"max34_q_max", "9999999999999999999999999999999999E+6111"},
	{"neg_max34_q_max", "-9999999999999999999999999999999999E+6111"},
	{"pi_34digits", "3.141592653589793238462643383279503"},
	{"e_34digits", "2.718281828459045235360287471352662"},
	{"varied_digits_34", "1234567890123456789012345678901234"},
	{"one_q_max", "1E+6111"},
	{"one_q_min", "1E-6176"},
	{"posInfinity", "Infinity"},
	{"negInfinity", "-Infinity"},
	{"posQNaN", "NaN"},
	{"negQNaN", "-NaN"},
	{"posSNaN", "sNaN"},
	{"negSNaN", "-sNaN"},
	{"qnan_payload_1", "NaN1"},
	{"qnan_payload_12345", "NaN12345"},
	{"snan_payload_12345", "sNaN12345"},
	{"negQnan_payload_12345", "-NaN12345"},
	{"negSnan_payload_12345", "-sNaN12345"},
	{"qnan_payload_18digit", "NaN123456789012345678"},
	{"qnan_payload_19digit", "NaN1234567890123456789"},
	{"qnan_payload_max33digit", "NaN999999999999999999999999999999999"},
	{"snan_payload_max33digit", "sNaN999999999999999999999999999999999"},
	{"qnan_payload_leading_eight", "NaN800000000000000000000000000000000"},
	{"qnan_payload_leading_nine", "NaN900000000000000000000000000000000"},
}

func TestBid128RoundTripRoundTrip_allCases(t *testing.T) {
	for _, tc := range bid128RoundTripCases {
		original := mustDec(t, tc[1])
		decoded := d128_from_bid128(d128_to_bid128(original))
		if !d128_bitwiseEQ(decoded, original) {
			t.Errorf("Round-trip mismatch for %s", tc[0])
		}
		// Cross-check classification flags survived (paranoia).
		if d128_isNAN(decoded) != d128_isNAN(original) || d128_isSNAN(decoded) != d128_isSNAN(original) ||
			d128_isINF(decoded) != d128_isINF(original) || d128_isZER(decoded) != d128_isZER(original) ||
			d128_signFlag(decoded) != d128_signFlag(original) {
			t.Errorf("classification mismatch in %s", tc[0])
		}
		if d128_isFNZ(original) && d128_qExp(decoded) != d128_qExp(original) {
			t.Errorf("qExp mismatch in %s", tc[0])
		}
		if d128_isNAN(original) && d128_nanPayload(decoded) != d128_nanPayload(original) {
			t.Errorf("NaN payload mismatch in %s", tc[0])
		}
	}
}

// (name, hex wire form, canonical string spelling of the same value)
var bid128KnownVectors = [][3]string{
	{"posZero_q0", "[30400000000000000000000000000000]", "0"},
	{"negZero_q0", "[B0400000000000000000000000000000]", "-0"},
	{"posOne_q0", "[30400000000000000000000000000001]", "1"},
	{"negOne_q0", "[B0400000000000000000000000000001]", "-1"},
	{"posInfinity", "[78000000000000000000000000000000]", "Infinity"},
	{"negInfinity", "[F8000000000000000000000000000000]", "-Infinity"},
	{"posQNaN", "[7C000000000000000000000000000000]", "NaN"},
	{"negQNaN", "[FC000000000000000000000000000000]", "-NaN"},
	{"posSNaN", "[7E000000000000000000000000000000]", "sNaN"},
	{"negSNaN", "[FE000000000000000000000000000000]", "-sNaN"},
	{"posOne_q0_commaForm", "[3040000000000000,0000000000000001]", "1"},
	{"qnan_payload_1", "[7C000000000000000000000000000001]", "NaN1"},
	{"snan_payload_12345", "[7E000000000000000000000000003039]", "sNaN12345"},
}

func TestBid128RoundTripKnownVectors(t *testing.T) {
	for _, tc := range bid128KnownVectors {
		expected, ok := bid128ParseIntelHex(tc[1])
		if !ok {
			t.Fatalf("Failed to parse hex %q for %s", tc[1], tc[0])
		}
		// Decode the wire form, then parse the string spelling, and check they match.
		parsed := mustDec(t, tc[2])
		if !d128_bitwiseEQ(d128_from_bid128(expected), parsed) {
			t.Errorf("decode vs. string-parse mismatch for %s", tc[0])
		}
		// Re-encode and check that we recover the same bits.
		if d128_to_bid128(parsed) != expected {
			t.Errorf("re-encode mismatch for %s", tc[0])
		}
	}
}

// (name, input, accepted, expected value in decimal)
var bid128HexParserCases = []struct {
	name     string
	input    string
	accepted bool
	expected string
}{
	{"zero_single", "[00000000000000000000000000000000]", true, "0"},
	{"zero_comma", "[0000000000000000,0000000000000000]", true, "0"},
	{"one_single", "[00000000000000000000000000000001]", true, "1"},
	{"one_comma", "[0000000000000000,0000000000000001]", true, "1"},
	{"lowercase_hex", "[bc92000000000000,0000000000000000]", true, "250652938548910268939681179796770914304"},
	{"uppercase_hex", "[BC92000000000000,0000000000000000]", true, "250652938548910268939681179796770914304"},
	{"mixed_case_hex", "[291a7e63609Fe325,01E21199F946252B]", true, "54635911002605458138065406555425482027"},
	{"all_ones", "[FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF]", true, "340282366920938463463374607431768211455"},
	{"rejected_empty", "", false, ""},
	{"rejected_missing_open", "30400000000000000000000000000000]", false, ""},
	{"rejected_missing_close", "[30400000000000000000000000000000", false, ""},
	{"rejected_no_brackets", "30400000000000000000000000000000", false, ""},
	{"rejected_too_short", "[3040000000000000000000000000000]", false, ""},
	{"rejected_too_long", "[304000000000000000000000000000000]", false, ""},
	{"rejected_comma_wrong_pos", "[304000000000000,00000000000000000]", false, ""},
	{"rejected_non_hex_char", "[30400000000000G00000000000000000]", false, ""},
	{"rejected_non_hex_comma_form", "[3040000000000000,000000000000000G]", false, ""},
}

func TestBid128RoundTripParseIntelHex_allCases(t *testing.T) {
	for _, tc := range bid128HexParserCases {
		got, ok := bid128ParseIntelHex(tc.input)
		if ok != tc.accepted || (ok && big128(got.dw0, got.dw1).Cmp(bigStr(tc.expected)) != 0) {
			t.Errorf("parse result mismatch for %s", tc.name)
		}
	}
}
