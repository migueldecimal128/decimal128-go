package decimal128

// D48 kernel error bounds (D48GoWorkOrder.md Phase 3; Swift TestD48Bounds,
// Zig d48_bounds_test.zig): for every resources/d48/transc_truth.txt line
// (130-digit truth from Python decimal), the error of the D48 kernel value in
// units of its last digit. Each per-function count and maximum must equal the
// other ports' (all equal Zig's), and every error must be under the bound the
// boundary test assumes (10^4; pow 10^4 + 1000·|u|). Calls the kernels
// directly, so it runs in every build.

import (
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type bigDec struct {
	neg bool
	c   *big.Int
	q   int
}

func parseBigDec(t *testing.T, tok string) bigDec {
	t.Helper()
	neg := strings.HasPrefix(tok, "-")
	body := strings.TrimPrefix(tok, "-")
	e := strings.IndexByte(body, 'E')
	c, ok := new(big.Int).SetString(body[:e], 10)
	q, err := strconv.Atoi(body[e+1:])
	if !ok || (err != nil) {
		t.Fatalf("bad truth %q", tok)
	}
	return bigDec{neg, c, q}
}

// unitsOff is |R − truth| in units of R's last digit (rounded up), R =
// coeff·10^qExp; a huge sentinel on a sign mismatch or an absurd exponent spread.
func unitsOff(neg bool, coeff u320, qExp int, tr bigDec) *big.Int {
	huge := bigPow2(1000)
	if neg != tr.neg {
		return huge
	}
	m := min(qExp, tr.q)
	if (qExp-m > 250) || (tr.q-m > 250) {
		return huge
	}
	a := bigMul(bigOf320(coeff), bigTen(qExp-m))
	b := bigMul(tr.c, bigTen(tr.q-m))
	diff := new(big.Int).Abs(bigSub(a, b))
	unit := bigTen(qExp - m)
	return bigQuo(bigSub(bigAdd(diff, unit), bigOne()), unit)
}

func TestD48KernelErrorsEqualTheOtherPorts(t *testing.T) {
	// Zig prototype (d48_bounds_test.zig), = Swift / C / Rust: count and maximum
	// error per function on the same vectors.
	want := map[string][2]int{
		"exp": {400, 42}, "exp10": {378, 41}, "ln": {400, 93}, "log10": {400, 42}, "pow": {400, 3764},
	}
	data, err := os.ReadFile(filepath.Join("resources", "d48", "transc_truth.txt"))
	if err != nil {
		t.Fatal(err)
	}
	n := map[string]int{}
	maxErr := map[string]*big.Int{}
	over := 0
	bound := big.NewInt(10_000)
	parse := func(s string) d128 {
		d, ok := d128_parse(s)
		if !ok {
			t.Fatalf("bad operand %q", s)
		}
		return d
	}
	for _, line := range strings.Split(string(data), "\n") {
		if (line == "") || strings.HasPrefix(line, "#") {
			continue
		}
		tok := strings.Split(line, " ")
		fn := tok[0]
		x := parse(tok[1])
		var y d128
		arrow := 2
		if tok[2] != "->" {
			y = parse(tok[2])
			arrow = 3
		}
		truth := parseBigDec(t, tok[arrow+1])
		x48 := d48_fromD128(x, d128_signFlag(x))

		var e *big.Int
		limit := bound
		switch fn {
		case "exp":
			if d128_sciExp(x) >= 6 {
				continue
			}
			s := expScaled_d48(x)
			e = unitsOff(false, d48_u320(s.z), d48_qExp(s.z)+s.n, truth)
		case "exp10":
			if d48_toLongOrMin(x48) != math.MinInt64 {
				continue // exact integer power
			}
			s := exp10Scaled_d48(x48)
			e = unitsOff(false, d48_u320(s.z), d48_qExp(s.z)+s.n, truth)
		case "ln", "log10":
			r := d48_logMagnitude(x48, fn == "log10")
			e = unitsOff(d48_signFlag(r), d48_u320(r), d48_qExp(r), truth)
		case "pow":
			u := d48_multiply(d48_fromD128(y, d128_signFlag(y)), d48_logMagnitude(x48, true))
			if d48_toLongOrMin(u) != math.MinInt64 {
				continue
			}
			s := exp10Scaled_d48(u)
			e = unitsOff(false, d48_u320(s.z), d48_qExp(s.z)+s.n, truth)
			limit = bigAdd(bound, bigMul(big.NewInt(1000), bigTen(max(0, d48_sciExp(u)+1))))
		default:
			t.Fatalf("unknown fn %s", fn)
		}
		n[fn]++
		if (maxErr[fn] == nil) || (e.Cmp(maxErr[fn]) > 0) {
			maxErr[fn] = e
		}
		if e.Cmp(limit) > 0 {
			over++
		}
	}
	for _, fn := range []string{"exp", "exp10", "ln", "log10", "pow"} {
		m := maxErr[fn]
		t.Logf("BOUNDS %s: n=%d D48 max %v units (other ports: n=%d max %d)", fn, n[fn], m, want[fn][0], want[fn][1])
		if (n[fn] != want[fn][0]) || (m == nil) || (m.Cmp(big.NewInt(int64(want[fn][1]))) != 0) {
			t.Errorf("%s: n=%d max %v, want n=%d max %d", fn, n[fn], m, want[fn][0], want[fn][1])
		}
	}
	if over != 0 {
		t.Errorf("%d kernel values outside the boundary-test bound", over)
	}
}
