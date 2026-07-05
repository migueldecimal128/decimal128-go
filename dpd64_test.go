package decimal128

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// Decode tests for the decimal64 (DPD) decoder d128_from_dpd64. decimal64 is a
// strict subset of decimal128, so decoding a 64-bit DPD word is exact: the result
// must equal the same value built from the Core factories. There is no decimal64
// encoder (decode-only scope). Mirrors bid64_test.go.

func TestDpd64Decode(t *testing.T) {
	check := func(bits uint64, expected d128, label string) {
		decoded := d128_from_dpd64(int64(bits))
		if !d128_bitwiseEQ(decoded, expected) {
			t.Errorf("dpd64 decode [%s]: got hi=%#x lo=%#x want hi=%#x lo=%#x",
				label, decoded.ubdHi64, decoded.ubdLo64, expected.ubdHi64, expected.ubdLo64)
		}
	}

	// Case A finite (leading digit 0..7)
	check(0x2238000000000001, decimalFNZ_64x64(false, 0, 1, 0), "one")
	check(0xA238000000000001, decimalFNZ_64x64(true, 0, 1, 0), "negOne")
	check(0x2238000000000007, decimalFNZ_64x64(false, 0, 7, 0), "seven")
	check(0x2238000000000400, decimalFNZ_64x64(false, 0, 1000, 0), "thousand")
	check(0x2230000000000080, decimalFNZ_64x64(false, -2, 100, 0), "oneDotZeroZero")
	check(0x2638000000000000, decimalFNZ_64x64(false, 0, 1_000_000_000_000_000, 0), "tenToThe15_leadingDigit")

	// Case B (leading digit 8 or 9)
	check(0x6E38000000000000, decimalFNZ_64x64(false, 0, 9_000_000_000_000_000, 0), "caseB_nineLeading")

	// zeros
	check(0x2238000000000000, d128_zero(false, 0), "posZero_q0")
	check(0xA238000000000000, d128_zero(true, 0), "negZero_q0")

	// specials
	check(0x7800000000000000, d128_infinity(false), "posInfinity")
	check(0xF800000000000000, d128_infinity(true), "negInfinity")
	check(0x7C00000000000000, d128_nan_64x64(false, false, 0, 0), "posQNaN")
	check(0x7E00000000000000, d128_nan_64x64(false, true, 0, 0), "posSNaN")
	check(0xFC00000000000000, d128_nan_64x64(true, false, 0, 0), "negQNaN")

	// NaN payloads
	check(0x7C00000000000005, d128_nan_64x64(false, false, 5, 0), "qnan_payload_5")
	check(0x7C00000000000400, d128_nan_64x64(false, false, 1000, 0), "qnan_payload_1000")
	check(0x7E00000000000005, d128_nan_64x64(false, true, 5, 0), "snan_payload_5")
}

// TestDpd64Telco cross-checks d128_from_dpd64 against the shared telco corpus:
// telco.testr holds each duration as a big-endian dpd64 word, telco.testb holds
// the SAME durations as plain big-endian uint64 integers. Decoding record i of
// .testr must bitwise-equal the integer of record i as a decimal with exponent 0.
// The large expon180.1e6 corpus is checked too when present. Files hold whole
// 8-byte records plus a trailing partial byte, which is ignored.
func TestDpd64Telco(t *testing.T) {
	checkPair := func(rPath, bPath string) {
		rData, err := os.ReadFile(rPath)
		if err != nil {
			t.Fatalf("cannot read %s: %v", rPath, err)
		}
		bData, err := os.ReadFile(bPath)
		if err != nil {
			t.Fatalf("cannot read %s: %v", bPath, err)
		}
		n := len(rData) / 8
		if m := len(bData) / 8; m < n {
			n = m
		}
		if n == 0 {
			t.Fatalf("no whole 8-byte records in %s / %s", rPath, bPath)
		}
		for i := 0; i < n; i++ {
			off := i * 8
			dpd := binary.BigEndian.Uint64(rData[off : off+8])
			dur := binary.BigEndian.Uint64(bData[off : off+8])
			decoded := d128_from_dpd64(int64(dpd))
			expected := decimalFNZ_64x64(false, 0, int64(dur), 0)
			if !d128_bitwiseEQ(decoded, expected) {
				t.Fatalf("telco[%s] record %d: dpd64=%#016x dur=%d got hi=%#x lo=%#x want hi=%#x lo=%#x",
					filepath.Base(rPath), i, dpd, dur,
					decoded.ubdHi64, decoded.ubdLo64, expected.ubdHi64, expected.ubdLo64)
			}
		}
	}

	telcoDir := filepath.Join("resources", "telco")
	checkPair(filepath.Join(telcoDir, "telco.testr"), filepath.Join(telcoDir, "telco.testb"))

	// The 1,000,000-record primary workload (guard: skip if absent).
	rBig := filepath.Join(telcoDir, "expon180.1e6r")
	bBig := filepath.Join(telcoDir, "expon180.1e6b")
	if _, err := os.Stat(rBig); err != nil {
		t.Skipf("expon180.1e6r absent, skipping 1e6 cross-check: %v", err)
	}
	if _, err := os.Stat(bBig); err != nil {
		t.Skipf("expon180.1e6b absent, skipping 1e6 cross-check: %v", err)
	}
	checkPair(rBig, bBig)
}
