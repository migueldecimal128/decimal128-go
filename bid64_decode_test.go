package decimal128

// Decode tests for the decimal64 (BID) decoder d128_from_bid64.
//
// decimal64 is a strict subset of decimal128, so decoding a 64-bit word is exact:
// the resulting d128 must be bitwise-equal to the same value built from the Core
// factories. There is no decimal64 encoder (decode-only scope), so each case pairs
// a hand-derived 64-bit pattern with its expected d128 rather than round-tripping.

import "testing"

func TestBid64DecodeDecodeAllCases(t *testing.T) {
	cases := []struct {
		name     string
		bits     uint64
		expected d128
	}{
		{"one", 0x31C0_0000_0000_0001, decimalFNZ_64x64(false, 0, 1, 0)},
		{"negOne", 0xB1C0_0000_0000_0001, decimalFNZ_64x64(true, 0, 1, 0)},
		{"hundred_q0", 0x31C0_0000_0000_0064, decimalFNZ_64x64(false, 0, 100, 0)},
		{"oneDotZeroZero", 0x3180_0000_0000_0064, decimalFNZ_64x64(false, -2, 100, 0)},
		{"seven", 0x31C0_0000_0000_0007, decimalFNZ_64x64(false, 0, 7, 0)},
		{"caseB_prefix8", (6940 << 50) | 5, decimalFNZ_64x64(false, 0, (8<<50)|5, 0)},
		{"caseB_prefix9_noncanonical", (6941 << 50) | 0, d128_zero(false, 0)},
		{"posZero_q0", 0x31C0_0000_0000_0000, d128_zero(false, 0)},
		{"posInfinity", 0x7800_0000_0000_0000, d128_infinity(false)},
		{"negInfinity", 0xF800_0000_0000_0000, d128_infinity(true)},
		{"posQNaN", 0x7C00_0000_0000_0000, d128_nan_64x64(false, false, 0, 0)},
		{"posSNaN", 0x7E00_0000_0000_0000, d128_nan_64x64(false, true, 0, 0)},
		{"negQNaN", 0xFC00_0000_0000_0000, d128_nan_64x64(true, false, 0, 0)},
		{"qnan_payload_12345", 0x7C00_0000_0000_3039, d128_nan_64x64(false, false, 12345, 0)},
		{"snan_payload_12345", 0x7E00_0000_0000_3039, d128_nan_64x64(false, true, 12345, 0)},
		{"qnan_payload_max15", 0x7C00_0000_0000_0000 | 999_999_999_999_999, d128_nan_64x64(false, false, 999_999_999_999_999, 0)},
		{"qnan_payload_noncanonical", 0x7C00_0000_0000_0000 | 1_000_000_000_000_000, d128_nan_64x64(false, false, 0, 0)},
	}
	for _, tc := range cases {
		decoded := d128_from_bid64(int64(tc.bits))
		e := tc.expected
		if !d128_bitwiseEQ(decoded, e) {
			t.Errorf("bid64 decode mismatch for %s", tc.name)
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
