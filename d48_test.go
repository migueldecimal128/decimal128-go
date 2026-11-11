package decimal128

// The D48 domain (D48GoWorkOrder.md Phase 2; Swift TestD48.swift): every D48
// operation against the Python decimal prec=48 oracle in resources/d48/ops.txt
// (22,505 vectors, from decimal128-workorder/d48-prototype/gen_d48_vectors.py),
// bit for bit -- sign, qExp and coefficient; a zero result only has to be zero
// (the domain has no −0 and no zero cohort). Plus the Zig prototype's unit
// tests, ported.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func d48Show(v d48) string {
	sign := ""
	if d48_signFlag(v) {
		sign = "-"
	}
	return fmt.Sprintf("%s%sE%d", sign, bigOf320(d48_u320(v)).String(), d48_qExp(v))
}

func sameD48(got, want d48) bool {
	if d48_isZero(want) {
		return d48_isZero(got)
	}
	return (got.qExpAndSignBit == want.qExpAndSignBit) && (d48_u160(got) == d48_u160(want))
}

// runD48Vectors runs `op args -> expected` lines; returns the count and the
// failing lines with what D48 gave.
func runD48Vectors(t *testing.T, lines []string) (int, []string) {
	t.Helper()
	total := 0
	var failures []string
	for _, line := range lines {
		if (line == "") || strings.HasPrefix(line, "#") {
			continue
		}
		tok := strings.Split(line, " ")
		op := tok[0]
		arrow := -1
		for i, s := range tok {
			if s == "->" {
				arrow = i
				break
			}
		}
		if (arrow < 0) || (arrow+1 >= len(tok)) {
			t.Fatalf("not a D48 vector: %q", line)
		}
		var args []d48
		for _, s := range tok[1:arrow] {
			args = append(args, d48_parse(s))
		}
		expected := tok[arrow+1]
		total++
		var ok bool
		var got string
		switch op {
		case "cmp":
			r := d48_cmpMag(args[0], args[1])
			want, err := strconv.Atoi(expected)
			ok = (err == nil) && (r == want)
			got = strconv.Itoa(r)
		case "tol":
			r := d48_toLongOrMin(args[0])
			var want int64 = math.MinInt64
			if expected != "MIN" {
				var err error
				if want, err = strconv.ParseInt(expected, 10, 64); err != nil {
					t.Fatalf("bad tol expectation: %q", line)
				}
			}
			ok = r == want
			got = strconv.FormatInt(r, 10)
		default:
			var r d48
			switch op {
			case "add":
				r = d48_add(args[0], args[1])
			case "sub":
				r = d48_subtract(args[0], args[1])
			case "mul":
				r = d48_multiply(args[0], args[1])
			case "sqr":
				r = d48_square(args[0])
			case "div":
				r = d48_divide(args[0], args[1])
			case "fma":
				r = d48_fma(args[0], args[1], args[2])
			case "sqrt":
				r = d48_sqrt(args[0])
			case "rta":
				r = d48_roundToIntegralTiesToAway(args[0])
			case "rtn":
				r = d48_roundToIntegralTowardNegative(args[0])
			default:
				t.Fatalf("unknown op %s", op)
			}
			ok = sameD48(r, d48_parse(expected))
			got = d48Show(r)
		}
		if !ok {
			failures = append(failures, line+"  got "+got)
		}
	}
	return total, failures
}

func TestD48MatchesPythonDecimalPrec48Oracle(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("resources", "d48", "ops.txt"))
	if err != nil {
		t.Fatal(err)
	}
	total, failures := runD48Vectors(t, strings.Split(string(data), "\n"))
	for _, f := range failures[:min(len(failures), 20)] {
		t.Errorf("D48 FAIL: %s", f)
	}
	t.Logf("d48/ops.txt: %d vectors, %d failures", total, len(failures))
	if total != 22505 {
		t.Errorf("%d vectors, want 22505", total)
	}
	if len(failures) != 0 {
		t.Errorf("%d of %d D48 vectors failed", len(failures), total)
	}
}

// The gate's bite check: a corrupted expected value is reported.
func TestD48CorruptedExpectedValueIsReported(t *testing.T) {
	good := "mul -82004105643687E169 10000000000000E-325 -> -820041056436870000000000000E-156"
	bad := "mul -82004105643687E169 10000000000000E-325 -> -820041056436870000000000001E-156"
	if _, f := runD48Vectors(t, []string{good}); len(f) != 0 {
		t.Errorf("good line failed: %v", f)
	}
	if _, f := runD48Vectors(t, []string{bad}); len(f) != 1 {
		t.Errorf("bad line: %d failures, want 1", len(f))
	}
}

// --- Zig prototype unit tests, ported ---

func d48f(sign bool, qExp int, coeff string) d48 {
	s := ""
	if sign {
		s = "-"
	}
	return d48_parse(fmt.Sprintf("%s%sE%d", s, coeff, qExp))
}

func TestD48PackUnpackSignQExpThreeLimbs(t *testing.T) {
	a := d48f(true, -1234, strings.Repeat("9", 48))
	if (d48_qExp(a) != -1234) || !d48_signFlag(a) || (d48_digitLen(a) != 48) ||
		d48_signFlag(d48_negated(a)) || !d48_isZero(const_D48_ZERO) {
		t.Errorf("pack/unpack: %s", d48Show(a))
	}
}

func TestD48MultiplyKeepsExactProductsRoundsWideOnes(t *testing.T) {
	z := d48_multiply(d48f(false, 2, "6"), d48f(true, 3, "7")) // 6e2 * -7e3 = -42e5
	if !d48_signFlag(z) || (d48_qExp(z) != 5) || (d48Show(z) != "-42E5") {
		t.Errorf("6e2*-7e3 = %s", d48Show(z))
	}
	// (10^48 - 1)^2 = 10^96 - 2·10^48 + 1 → rounds to 99..98 (48 digits), qExp 48.
	sq := d48_square(d48f(false, 0, strings.Repeat("9", 48)))
	if (d48_digitLen(sq) != 48) || (d48_qExp(sq) != 48) || (d48Show(sq) != strings.Repeat("9", 47)+"8E48") {
		t.Errorf("square = %s", d48Show(sq))
	}
	if d48Show(d48_fromInt32(-9)) != "-9E0" {
		t.Errorf("fromInt32(-9) = %s", d48Show(d48_fromInt32(-9)))
	}
}

func TestD48FinalizeRoundsAndRescales(t *testing.T) {
	// 49 nines → 10^48 after rounding → 10^47 at qExp 2.
	nines49 := sub_320_320x320(mulPow10_320(u320FromDw0(1), 49), u320FromDw0(1))
	if got := d48Show(d48_finalize_320(false, 0, nines49, const_RESIDUE_EXACT)); got != "1"+strings.Repeat("0", 47)+"E2" {
		t.Errorf("49 nines: %s", got)
	}
	// 10^95 + 5·10^47: drop 48 → 10^47 (even), exactly half → keep; with a
	// sticky incoming residue → round up.
	c := add_320_320x320(mulPow10_320(u320FromDw0(1), 95), mulPow10_320(u320FromDw0(5), 47))
	if got := d48Show(d48_finalize_320(false, 0, c, const_RESIDUE_EXACT)); got != "1"+strings.Repeat("0", 47)+"E48" {
		t.Errorf("half, exact: %s", got)
	}
	if got := d48Show(d48_finalize_320(false, 0, c, const_RESIDUE_LT_HALF)); got != "1"+strings.Repeat("0", 46)+"1E48" {
		t.Errorf("half, sticky: %s", got)
	}
}

func TestD48CmpMagAlignsExponents(t *testing.T) {
	cases := []struct {
		x, y d48
		want int
	}{
		{d48f(false, 2, "5"), d48f(false, 0, "5"), 1},
		{d48f(false, 0, "12"), d48f(false, -1, "120"), 0},
		{d48f(true, 0, "3"), d48f(false, 0, "9"), -1},
		{d48f(false, 100000, "1"), d48f(false, -100000, "7"), 1}, // huge gap
		{const_D48_ZERO, d48f(false, -100000, "7"), -1},
		{const_D48_ZERO, const_D48_ZERO, 0},
	}
	for i, c := range cases {
		if got := d48_cmpMag(c.x, c.y); got != c.want {
			t.Errorf("case %d: %d, want %d", i, got, c.want)
		}
	}
}

func TestD48FmaBasicCancellationAndWorstCaseSum(t *testing.T) {
	if got := d48Show(d48_fma(d48f(false, 0, "3"), d48f(false, 0, "4"), d48f(false, 0, "5"))); got != "17E0" {
		t.Errorf("3*4+5 = %s", got)
	}
	if !d48_isZero(d48_fma(d48f(false, 0, "3"), d48f(false, 0, "4"), d48f(true, 0, "12"))) {
		t.Errorf("3*4-12 not zero")
	}
	// x = y = c = 10^48 - 1 at every exponent gap: the like-sign sum stays < 10^96.
	nines := strings.Repeat("9", 48)
	for gap := -200; gap <= 200; gap++ {
		r := d48_fma(d48f(false, 0, nines), d48f(false, 0, nines), d48f(false, gap, nines))
		if d48_digitLen(r) != 48 {
			t.Errorf("gap %d: %s", gap, d48Show(r))
		}
	}
}

func TestD48DividePadsExactQuotientsTo48Digits(t *testing.T) {
	if got := d48Show(d48_divide(d48f(false, 0, "6"), d48f(false, 0, "3"))); got != "2"+strings.Repeat("0", 47)+"E-47" {
		t.Errorf("6/3 = %s", got)
	}
	if got := d48Show(d48_divide(d48f(false, 0, "1"), d48f(false, 0, "3"))); got != strings.Repeat("3", 48)+"E-48" {
		t.Errorf("1/3 = %s", got)
	}
	if got := d48Show(d48_divide(d48f(false, 0, "2"), d48f(true, 0, "3"))); got != "-"+strings.Repeat("6", 47)+"7E-48" {
		t.Errorf("2/-3 = %s", got)
	}
}

func TestD48SquareRoot(t *testing.T) {
	if d48_sciExp(d48_sqrt(d48f(false, 0, "4"))) != 0 {
		t.Errorf("sqrt(4) sciExp")
	}
	if !d48_isZero(d48_sqrt(const_D48_ZERO)) {
		t.Errorf("sqrt(0) not zero")
	}
	if got := d48Show(d48_sqrt(d48f(false, 0, "2"))); got != "141421356237309504880168872420969807856967187538E-47" {
		t.Errorf("sqrt(2) = %s", got)
	}
}

func TestD48RoundToIntegralAndToLong(t *testing.T) {
	shows := []struct {
		got  d48
		want string
	}{
		{d48_roundToIntegralTiesToAway(d48f(false, -1, "25")), "3E0"},
		{d48_roundToIntegralTiesToAway(d48f(true, -1, "25")), "-3E0"},
		{d48_roundToIntegralTowardNegative(d48f(false, -1, "27")), "2E0"},
		{d48_roundToIntegralTowardNegative(d48f(true, -1, "21")), "-3E0"},
		{d48_roundToIntegralTowardNegative(d48f(true, -200, "7")), "-1E0"},
	}
	for i, c := range shows {
		if d48Show(c.got) != c.want {
			t.Errorf("round case %d: %s, want %s", i, d48Show(c.got), c.want)
		}
	}
	if !d48_isZero(d48_roundToIntegralTiesToAway(d48f(false, -200, "7"))) {
		t.Errorf("rta(7E-200) not zero")
	}
	longs := []struct {
		x    d48
		want int64
	}{
		{const_D48_ZERO, 0},
		{d48f(false, 2, "5"), 500},
		{d48f(true, 0, "42"), -42},
		{d48f(false, -1, "25"), math.MinInt64},
		{d48f(false, -2, "100"), 1},
		{d48f(true, 0, "9223372036854775808"), math.MinInt64}, // -2^63 itself
	}
	for i, c := range longs {
		if got := d48_toLongOrMin(c.x); got != c.want {
			t.Errorf("toLong case %d: %d, want %d", i, got, c.want)
		}
	}
}
