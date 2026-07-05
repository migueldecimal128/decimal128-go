package decimal128

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Decode tests for the decimal64 (BID) decoder d128_from_bid64. decimal64 is a
// strict subset of decimal128, so decoding a 64-bit BID word is exact: the
// result must equal the same value built from the Core factories. There is no
// decimal64 encoder (decode-only scope).

func TestBid64Decode(t *testing.T) {
	check := func(bits uint64, expected d128, label string) {
		decoded := d128_from_bid64(int64(bits))
		if !d128_bitwiseEQ(decoded, expected) {
			t.Errorf("bid64 decode [%s]: got hi=%#x lo=%#x want hi=%#x lo=%#x",
				label, decoded.ubdHi64, decoded.ubdLo64, expected.ubdHi64, expected.ubdLo64)
		}
	}

	// Case A finite (leading digit 0..7)
	check(0x31C0000000000001, decimalFNZ_64x64(false, 0, 1, 0), "one")
	check(0xB1C0000000000001, decimalFNZ_64x64(true, 0, 1, 0), "negOne")
	check(0x31C0000000000064, decimalFNZ_64x64(false, 0, 100, 0), "hundred_q0")
	check(0x3180000000000064, decimalFNZ_64x64(false, -2, 100, 0), "oneDotZeroZero")
	check(0x31C0000000000007, decimalFNZ_64x64(false, 0, 7, 0), "seven")

	// Case B (leading digit 8): combination 6940 -> prefix 8, coeff = 8*2^50 + 5.
	check((6940<<50)|5, decimalFNZ_64x64(false, 0, (8<<50)|5, 0), "caseB_prefix8")

	// Non-canonical finite (prefix 9, coeff > 10^16-1) canonicalizes to zero.
	check(6941<<50, d128_zero(false, 0), "caseB_prefix9_noncanonical")

	check(0x31C0000000000000, d128_zero(false, 0), "posZero_q0")

	// specials
	check(0x7800000000000000, d128_infinity(false), "posInfinity")
	check(0xF800000000000000, d128_infinity(true), "negInfinity")
	check(0x7C00000000000000, d128_nan_64x64(false, false, 0, 0), "posQNaN")
	check(0x7E00000000000000, d128_nan_64x64(false, true, 0, 0), "posSNaN")
	check(0xFC00000000000000, d128_nan_64x64(true, false, 0, 0), "negQNaN")

	// NaN payloads
	check(0x7C00000000003039, d128_nan_64x64(false, false, 12345, 0), "qnan_payload_12345")
	check(0x7E00000000003039, d128_nan_64x64(false, true, 12345, 0), "snan_payload_12345")
	// 15-digit max canonical payload (10^15-1); non-canonical (10^15) -> zero.
	check(0x7C00000000000000|999_999_999_999_999,
		d128_nan_64x64(false, false, 999_999_999_999_999, 0), "qnan_payload_max15")
	check(0x7C00000000000000|1_000_000_000_000_000,
		d128_nan_64x64(false, false, 0, 0), "qnan_payload_noncanonical")
}

// Validates d128_from_bid64 against the Intel readtest.in reference
// bid64_to_bid128 conversion vectors (exact widening). Two NaN classes are
// skipped: sNaN inputs (convertFormat quiets + signals INVALID -> nonzero flags)
// and quiet-NaN payloads that libbid left-justifies (x10^18; we preserve the raw
// payload). Raw NaN-payload decode is covered by TestBid64Decode.
func TestBid64ToBid128Oracle(t *testing.T) {
	// operands whose expected result left-justifies the NaN payload (libbid bug).
	skipCases := map[string]bool{
		"7c00000000000001": true, "7c00000100000000": true, "7c00003b9aca0000": true,
		"7cf0000000000001": true, "fc00000000000001": true, "fc00000100000000": true,
		"fcf0000000000001": true,
	}

	path := filepath.Join(resourcesDir(), "intel", "readtest.in")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open %s: %v", path, err)
	}
	defer f.Close()

	checked := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "bid64_to_bid128" {
			continue
		}
		op16 := strings.Trim(fields[2], "[]")
		// (1) convertFormat signaled (sNaN -> qNaN + invalid): raw decode differs.
		flagVal, ferr := strconv.ParseUint(fields[4], 16, 32)
		if ferr != nil || flagVal != 0 {
			continue
		}
		// (2) libbid NaN-payload left-justify bug: preserve raw payload instead.
		if skipCases[op16] {
			continue
		}
		bid64u, perr := strconv.ParseUint(op16, 16, 64)
		if perr != nil {
			continue
		}
		expectedBits, ok := bid128ParseIntelHex(fields[3])
		if !ok {
			continue
		}
		expected := d128_from_bid128(expectedBits)
		decoded := d128_from_bid64(int64(bid64u))
		if !d128_bitwiseEQ(decoded, expected) {
			t.Errorf("bid64_to_bid128 mismatch: %s", line)
		}
		checked++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	// 173 lines - 14 sNaN (flag 01) - 7 skipCases = 152 exact widening vectors.
	if checked < 150 {
		t.Fatalf("expected ~152 bid64_to_bid128 vectors, only checked %d", checked)
	}
}
