package decimal128

// Bit-for-bit regression against the Kotlin-legacy implementation. The golden
// file (resources/rosetta/golden/kotlin_logexp.txt) is in Intel readtest.in
// format, but RESULT and FLAGS are Kotlin's own output. Each case is run
// through the op with the line's rounding mode; value (bitwiseEQ) and flags
// must match exactly.
//
// Faithful-port oracle: where this port is more accurate than Kotlin, a
// last-digit divergence is reported here rather than hidden. Lines where the
// Kotlin result is known to be wrong are listed in goldenKotlinErrors with the
// correctly rounded result, which this port must give instead.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// goldenKotlinErrors: golden lines whose Kotlin result is wrong, with the
// correct BID result. exp of a tiny positive x under round-toward-positive:
// e^x > 1, so the result is the next decimal128 above 1 (1.000…001, coefficient
// 10^33 + 1), not 1. Kotlin's 38-digit exp rounded 1 + x to exactly 1; the D48
// exp applies the tiny-argument rule.
var goldenKotlinErrors = map[string]string{
	"bid128_exp 2 [00000000000000000000000000000001] [2ffe314dc6448d9338c15b0a00000000] 20": "[2ffe314dc6448d9338c15b0a00000001]",
	"bid128_exp 2 [00000000000000000000000000000003] [2ffe314dc6448d9338c15b0a00000000] 20": "[2ffe314dc6448d9338c15b0a00000001]",
	"bid128_exp 2 [0000106f4216d9dbbd95c90355555555] [2ffe314dc6448d9338c15b0a00000000] 20": "[2ffe314dc6448d9338c15b0a00000001]",
	"bid128_exp 2 [00420000000000000000000000000001] [2ffe314dc6448d9338c15b0a00000000] 20": "[2ffe314dc6448d9338c15b0a00000001]",
	"bid128_exp 2 [00420000000000000000000000000003] [2ffe314dc6448d9338c15b0a00000000] 20": "[2ffe314dc6448d9338c15b0a00000001]",
}

func goldenRun(t *testing.T, op string, f func(d128, *context) d128) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("resources", "rosetta", "golden", "kotlin_logexp.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// Intel rounding codes: 0 nearest-even, 1 down, 2 up, 3 toward zero, 4 nearest-away.
	roundings := []round{const_ROUND_TIES_TO_EVEN, const_ROUND_TOWARD_NEGATIVE, const_ROUND_TOWARD_POSITIVE, const_ROUND_TOWARD_ZERO, const_ROUND_TIES_TO_AWAY}
	// Intel readtest flag bits -> exceptions.
	flagBits := map[int64]exception754{0x01: const_INVALID_OPERATION, 0x04: const_DIVISION_BY_ZERO, 0x08: const_OVERFLOW, 0x10: const_UNDERFLOW, 0x20: const_INEXACT}
	parseBid := func(hex string) d128 {
		bits, ok := bid128ParseIntelHex(hex)
		if !ok {
			t.Fatalf("bad golden hex %s", hex)
		}
		return d128_from_bid128(bits)
	}
	cases, fails := 0, 0
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		// <op> <rounding> [<op1>] [<res>] <flags>
		tok := strings.Fields(line)
		if len(tok) < 5 || tok[0] != op {
			continue
		}
		cases++
		r, _ := strconv.Atoi(tok[1])
		ctx := newContextRound(roundings[r])
		actual := f(parseBid(tok[2]), ctx)
		expected := parseBid(tok[3])
		if fix, ok := goldenKotlinErrors[strings.Join(tok, " ")]; ok {
			expected = parseBid(fix)
		}
		fl, _ := strconv.ParseInt(tok[4], 16, 64)
		flagsMatch := true
		for bit, e := range flagBits {
			if ctx.flags.isSet(e) != (fl&bit != 0) {
				flagsMatch = false
			}
		}
		if !d128_bitwiseEQ(actual, expected) || !flagsMatch {
			fails++
			t.Errorf("Golden %s mismatch: %s", op, line)
		}
	}
	if cases == 0 {
		t.Fatalf("no golden cases for %s", op)
	}
	t.Logf("GOLDEN %s: %d/%d mismatched", op, fails, cases)
}

func TestGoldenLogExpTestLn(t *testing.T)    { goldenRun(t, "bid128_log", d128_ln_ctx) }
func TestGoldenLogExpTestLog10(t *testing.T) { goldenRun(t, "bid128_log10", d128_log10_ctx) }
func TestGoldenLogExpTestExp(t *testing.T)   { goldenRun(t, "bid128_exp", d128_exp_ctx) }
func TestGoldenLogExpTestExp10(t *testing.T) { goldenRun(t, "bid128_exp10", d128_exp10_ctx) }
