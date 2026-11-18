package decimal128

// Corpus validation for the decimal64 (DPD) decoder d128_from_dpd64 against the
// Cowlishaw telco benchmark data in the shared decimal128-resources submodule.
// The telco inputs ship the same sequence of call durations in three encodings;
// two of them cross-check the DPD decoder end to end:
//   telco.testb / expon180.1e6b   big-endian uint64 — the plain INTEGER duration
//   telco.testr / expon180.1e6r   decimal64 in DPD — the same value, dpd64-encoded
// Each 8-byte record in .testr decodes (via d128_from_dpd64) to a decimal whose
// value must equal the integer in the matching .testb record. The telco
// durations are canonical small integers (exponent 0), so the expected d128 is
// decimalFNZ(false, 0, duration, 0) (or zero) and the check is bitwise.
// Files are 8 * N bytes plus (for the 10-record smoke set) a trailing 0x0a byte
// that is not a full record; the read loop advances only on complete records.

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// dpd64TelcoCrossCheck returns the record count checked, or -1 if the files are
// absent (the large corpus is not guaranteed present in every checkout).
func dpd64TelcoCrossCheck(t *testing.T, stem string) int {
	t.Helper()
	dir := filepath.Join("resources", "telco")
	dpd, err1 := os.ReadFile(filepath.Join(dir, stem+"r"))
	bin, err2 := os.ReadFile(filepath.Join(dir, stem+"b"))
	if err1 != nil || err2 != nil {
		return -1
	}
	records := min(len(dpd), len(bin)) / 8
	if records == 0 {
		t.Fatalf("no full records in %s", stem)
	}
	for i := 0; i < records; i++ {
		word := binary.BigEndian.Uint64(dpd[i*8:])
		duration := binary.BigEndian.Uint64(bin[i*8:])
		expected := d128_zero(false, 0)
		if duration != 0 {
			expected = decimalFNZ_64x64(false, 0, int64(duration), 0)
		}
		if !d128_bitwiseEQ(d128_from_dpd64(int64(word)), expected) {
			t.Fatalf("record %d of %s: dpd64 %#x decoded != integer %d", i, stem, word, duration)
		}
	}
	return records
}

// Ten-number smoke set (durations 308, 76, 68, 323, 82, 329, 14, 101, 522, 50).
func TestDpd64TelcoTelcoSmoke(t *testing.T) {
	if n := dpd64TelcoCrossCheck(t, "telco.test"); n != 10 {
		t.Errorf("expected 10 smoke records, got %d", n)
	}
}

// One-million-number corpus (exponential distribution, mean 180, max 2641).
func TestDpd64TelcoTelcoOneMillion(t *testing.T) {
	if n := dpd64TelcoCrossCheck(t, "expon180.1e6"); n >= 0 && n != 1_000_000 {
		t.Errorf("expected 1e6 records, got %d", n)
	}
}
