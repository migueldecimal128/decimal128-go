package decimal128

import (
	"math/big"
	"strings"
	"testing"
)

func TestU128StripTrailingZeroDigitsTestCase(t *testing.T) {
	z := func(n int) string { return strings.Repeat("0", n) }
	for _, tc := range []struct {
		name           string
		c              string // U128 input as a decimal string
		maxToStrip     int
		wantStripped   string
		wantStripCount int
	}{
		// Move a case to the top to debug it easily.

		// --- no trailing zeros: nothing to strip ------------------------------
		{"one_noZeros", "1", 10, "1", 0},
		{"odd_noZeros", "12345", 10, "12345", 0},
		// --- maxToStrip = 0 short-circuit -------------------------------------
		{"budgetZero_evenWithZeros", "1000", 0, "1000", 0},
		// --- single-digit strips (< chunk boundary) ---------------------------
		{"strip1", "10", 5, "1", 1},
		{"strip3", "123000", 5, "123", 3},
		// --- budget limits the strip count below available --------------------
		{"budgetCapsBelowAvailable", "1000000", 2, "10000", 2},
		// --- exactly chunk-sized strip (14 digits) ----------------------------
		{"strip14_exact", "1" + z(14), 14, "1", 14},
		// --- more than one chunk: loops twice ---------------------------------
		{"strip28_twoFullChunks", "7" + z(28), 30, "7", 28},
		// --- rem != 0 path: chunk=14 tried but residue forces tail call -------
		// Trailing zeros = 5, but stripped128's low 64 bits have many factors of 2,
		// so ntzBits >= 14. divRemPow10(14) gives rem != 0 (residue has nonzero digits
		// above the 5 trailing zeros). u64StripTrailingZeroDigits must finish the job.
		{"remNonzero_tailStrips5", "12345" + z(5), 20, "12345", 5},
		// --- budget exhausted exactly inside a chunk --------------------------
		{"budgetExactlyInsideChunk", "1" + z(20), 7, "1" + z(13), 7},
		// --- U128-sized values (> MaxUint64) ----------------------------------
		// 10^20 = 100_000_000_000_000_000_000, well above MaxUint64 (~1.8e19).
		{"above_u64_strip20", "1" + z(20), 25, "1", 20},
		// 10^38, near the top of U128 range.
		{"near_u128_max_strip38", "1" + z(38), 38, "1", 38},
		// U128 value with leading nonzero digits and many trailing zeros.
		{"above_u64_mixed", "987654321" + z(25), 30, "987654321", 25},
	} {
		c := u128Of(bigStr(tc.c))
		r := strip_s128sc_128(c.dw0, c.dw1, tc.maxToStrip)
		if big128(r.strippedDw0, r.strippedDw1).Cmp(bigStr(tc.wantStripped)) != 0 {
			t.Errorf("stripped mismatch: %s", tc.name)
		}
		if int(r.stripCount) != tc.wantStripCount {
			t.Errorf("stripCount mismatch: %s", tc.name)
		}
	}
}

// u128StripCheck is the reference strip: trailing zeros removed up to the cap.
func u128StripCheck(t *testing.T, c *big.Int, maxToStrip int) {
	t.Helper()
	v := new(big.Int).Set(c)
	count := 0
	ten1 := big.NewInt(10)
	for count < maxToStrip && v.Sign() != 0 && new(big.Int).Mod(v, ten1).Sign() == 0 {
		v.Quo(v, ten1)
		count++
	}
	x := u128Of(c)
	r := strip_s128sc_128(x.dw0, x.dw1, maxToStrip)
	if big128(r.strippedDw0, r.strippedDw1).Cmp(v) != 0 || int(r.stripCount) != count {
		t.Errorf("strip %v max=%d", c, maxToStrip)
	}
}

func TestU128StripTrailingZeroDigitsStrip_acrossDigitsAndCaps(t *testing.T) {
	for _, d := range []int64{1, 7, 123, 99_999, 1_000_000_007} {
		for _, k := range []int{0, 1, 2, 5, 13, 20, 30, 36} {
			c := new(big.Int).Mul(big.NewInt(d), ten(k))
			if c.Sign() == 0 || c.Cmp(pow2(128)) >= 0 {
				continue // nonzero, fits 128 bits
			}
			for _, cp := range []int{0, 1, 3, 8, 13, 20, 40} {
				u128StripCheck(t, c, cp)
			}
		}
	}
}

func TestU128StripTrailingZeroDigitsStrip_maxMagnitudeNoTrailingZero(t *testing.T) {
	// 2^128 - 1 ends in ...455 (no trailing zero): nothing stripped.
	u128StripCheck(t, new(big.Int).Sub(pow2(128), big.NewInt(1)), 40)
}
