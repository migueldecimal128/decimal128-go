package decimal128

// Decode tests for the decimal64 (DPD) decoder d128_from_dpd64.
//
// decimal64 is a strict subset of decimal128, so decoding a 64-bit word is exact:
// the resulting d128 must be bitwise-equal to the same value built from the Core
// factories. There is no decimal64 encoder (decode-only scope), so each case pairs
// a hand-derived 64-bit pattern with its expected d128 rather than round-tripping.

import "testing"

func TestDpd64DecodeDecodeAllCases(t *testing.T) {
	cases := []struct {
		name     string
		bits     uint64
		expected d128
	}{
		{"one", 0x2238_0000_0000_0001, decimalFNZ_64x64(false, 0, 1, 0)},
		{"negOne", 0xA238_0000_0000_0001, decimalFNZ_64x64(true, 0, 1, 0)},
		{"seven", 0x2238_0000_0000_0007, decimalFNZ_64x64(false, 0, 7, 0)},
		{"thousand", 0x2238_0000_0000_0400, decimalFNZ_64x64(false, 0, 1000, 0)},
		{"oneDotZeroZero", 0x2230_0000_0000_0080, decimalFNZ_64x64(false, -2, 100, 0)},
		{"tenToThe15_leadingDigit", 0x2638_0000_0000_0000, decimalFNZ_64x64(false, 0, 1_000_000_000_000_000, 0)},
		{"caseB_nineLeading", 0x6E38_0000_0000_0000, decimalFNZ_64x64(false, 0, 9_000_000_000_000_000, 0)},
		{"posZero_q0", 0x2238_0000_0000_0000, d128_zero(false, 0)},
		{"negZero_q0", 0xA238_0000_0000_0000, d128_zero(true, 0)},
		{"posInfinity", 0x7800_0000_0000_0000, d128_infinity(false)},
		{"negInfinity", 0xF800_0000_0000_0000, d128_infinity(true)},
		{"posQNaN", 0x7C00_0000_0000_0000, d128_nan_64x64(false, false, 0, 0)},
		{"posSNaN", 0x7E00_0000_0000_0000, d128_nan_64x64(false, true, 0, 0)},
		{"negQNaN", 0xFC00_0000_0000_0000, d128_nan_64x64(true, false, 0, 0)},
		{"qnan_payload_5", 0x7C00_0000_0000_0005, d128_nan_64x64(false, false, 5, 0)},
		{"qnan_payload_1000", 0x7C00_0000_0000_0400, d128_nan_64x64(false, false, 1000, 0)},
		{"snan_payload_5", 0x7E00_0000_0000_0005, d128_nan_64x64(false, true, 5, 0)},
	}
	for _, tc := range cases {
		decoded := d128_from_dpd64(int64(tc.bits))
		e := tc.expected
		if !d128_bitwiseEQ(decoded, e) {
			t.Errorf("dpd64 decode mismatch for %s", tc.name)
		}
		if d128_isNAN(decoded) != d128_isNAN(e) || d128_isSNAN(decoded) != d128_isSNAN(e) ||
			d128_isINF(decoded) != d128_isINF(e) || d128_isZER(decoded) != d128_isZER(e) ||
			d128_signFlag(decoded) != d128_signFlag(e) {
			t.Errorf("classification mismatch for %s", tc.name)
		}
		if d128_isFNZ(e) && d128_qExp(decoded) != d128_qExp(e) {
			t.Errorf("qExp %s", tc.name)
		}
		if d128_isNAN(e) && d128_nanPayload(decoded) != d128_nanPayload(e) {
			t.Errorf("payload %s", tc.name)
		}
	}
}

// Parser round-trip: the `#` + 16-hex-digit dectest spelling must yield the
// same bits, and malformed input must be rejected.
func TestDpd64DecodeHexParser(t *testing.T) {
	if v, ok := dpd64ParseDecTestHex("#2238000000000001"); !ok || v != 0x2238_0000_0000_0001 {
		t.Error("#2238000000000001")
	}
	if v, ok := dpd64ParseDecTestHex("#7c00000000000005"); !ok || v != 0x7C00_0000_0000_0005 {
		t.Error("#7c00000000000005")
	}
	for _, bad := range []string{
		"2238000000000001",   // missing '#'
		"#22380000000000",    // too short
		"#2238000000000001A", // too long
		"#22380000000000GG",  // non-hex
	} {
		if _, ok := dpd64ParseDecTestHex(bad); ok {
			t.Errorf("accepted %q", bad)
		}
	}
}
