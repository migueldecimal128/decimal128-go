package decimal128

// pown / rootn at the int edges of n (D48GoWorkOrder.md G-9; Zig
// d48_n_edges_test.zig, Rust d48_n_edges.rs). Go's n is int (64-bit), so
// n = math.MinInt must follow the rules -- overflow, full underflow, ±1 at the
// preferred exponent, the rootn chain -- with no panic and no wrapped
// arithmetic. D38 recursed until the stack was exhausted on pown(2, MinInt)
// (Phase 0b). Expected values are written out from the rules; the rootn values
// are Python decimal at 120 digits, rounded to 34 in each mode (Rust's / Zig's
// values). No corpus lines: the 32-bit-n ports can't read them.

import (
	"math"
	"strings"
	"testing"
)

var edgeModes = []struct {
	name string
	rnd  round
}{
	{"TIES_TO_EVEN", const_ROUND_TIES_TO_EVEN},
	{"TIES_TO_AWAY", const_ROUND_TIES_TO_AWAY},
	{"TOWARD_ZERO", const_ROUND_TOWARD_ZERO},
	{"TOWARD_POSITIVE", const_ROUND_TOWARD_POSITIVE},
	{"TOWARD_NEGATIVE", const_ROUND_TOWARD_NEGATIVE},
}

func edgeDec(t *testing.T, s string) d128 {
	t.Helper()
	d, ok := d128_parse(s)
	if !ok {
		t.Fatalf("bad literal %s", s)
	}
	return d
}

// edgeOverflow is overflow by mode for a result of the given sign.
func edgeOverflow(t *testing.T, neg bool, mode string) d128 {
	towardInf := strings.HasPrefix(mode, "TIES") ||
		((mode == "TOWARD_POSITIVE") && !neg) || ((mode == "TOWARD_NEGATIVE") && neg)
	s := "9.999999999999999999999999999999999E+6144"
	if towardInf {
		s = "Infinity"
	}
	if neg {
		s = "-" + s
	}
	return edgeDec(t, s)
}

// edgeUnderflow is full underflow by mode for a result of the given sign.
func edgeUnderflow(t *testing.T, neg bool, mode string) d128 {
	away := ((mode == "TOWARD_POSITIVE") && !neg) || ((mode == "TOWARD_NEGATIVE") && neg)
	s := "0E-6176"
	if away {
		s = "1E-6176"
	}
	if neg {
		s = "-" + s
	}
	return edgeDec(t, s)
}

func edgeCheck(t *testing.T, what, x, mode string, got, want d128) {
	t.Helper()
	if !d128_bitwiseEQ(got, want) {
		t.Errorf("%s(%s) %s: got %s, want %s", what, x, mode, d128_format(got, const_FORMAT_AUTO), d128_format(want, const_FORMAT_AUTO))
	}
}

func TestD48PownAtTheIntEdges(t *testing.T) {
	for _, m := range edgeModes {
		// n = MinInt (even, negative), MinInt + 1 (odd, negative), MaxInt (odd,
		// positive). |x| > 1: x^(negative) underflows, x^(positive) overflows;
		// |x| < 1 the other way round. Odd n keeps a negative base's sign.
		for _, x := range []string{"2", "-2", "0.5", "-0.5"} {
			negBase := x[0] == '-'
			big := strings.TrimPrefix(x, "-")[0] != '0' // |x| > 1
			xd := edgeDec(t, x)
			nMin := d128_pown_rnd(xd, math.MinInt, m.rnd)
			nMin1 := d128_pown_rnd(xd, math.MinInt+1, m.rnd)
			nMax := d128_pown_rnd(xd, math.MaxInt, m.rnd)
			if big {
				edgeCheck(t, "pown MIN", x, m.name, nMin, edgeUnderflow(t, false, m.name))
				edgeCheck(t, "pown MIN+1", x, m.name, nMin1, edgeUnderflow(t, negBase, m.name))
				edgeCheck(t, "pown MAX", x, m.name, nMax, edgeOverflow(t, negBase, m.name))
			} else {
				edgeCheck(t, "pown MIN", x, m.name, nMin, edgeOverflow(t, false, m.name))
				edgeCheck(t, "pown MIN+1", x, m.name, nMin1, edgeOverflow(t, negBase, m.name))
				edgeCheck(t, "pown MAX", x, m.name, nMax, edgeUnderflow(t, negBase, m.name))
			}
		}
		// |x| = 1: ±1 at the preferred exponent n·Q(x), clamped to [−33, 0].
		edgeCheck(t, "pown MIN", "1", m.name, d128_pown_rnd(edgeDec(t, "1"), math.MinInt, m.rnd), edgeDec(t, "1"))
		edgeCheck(t, "pown MIN", "1.00", m.name, d128_pown_rnd(edgeDec(t, "1.00"), math.MinInt, m.rnd), edgeDec(t, "1"))
		edgeCheck(t, "pown MAX", "1.00", m.name, d128_pown_rnd(edgeDec(t, "1.00"), math.MaxInt, m.rnd),
			edgeDec(t, "1.000000000000000000000000000000000"))
		edgeCheck(t, "pown MIN", "-1", m.name, d128_pown_rnd(edgeDec(t, "-1"), math.MinInt, m.rnd), edgeDec(t, "1"))
		edgeCheck(t, "pown MIN+1", "-1", m.name, d128_pown_rnd(edgeDec(t, "-1"), math.MinInt+1, m.rnd), edgeDec(t, "-1"))
		// |x| = 10: an exact power of ten out of range.
		edgeCheck(t, "pown MIN", "10", m.name, d128_pown_rnd(edgeDec(t, "10"), math.MinInt, m.rnd), edgeUnderflow(t, false, m.name))
		edgeCheck(t, "pown MAX", "10", m.name, d128_pown_rnd(edgeDec(t, "10"), math.MaxInt, m.rnd), edgeOverflow(t, false, m.name))
		// An exact negative power with a huge |n| is not exact (5^|n| is too wide).
		edgeCheck(t, "pown MIN", "5", m.name, d128_pown_rnd(edgeDec(t, "5"), math.MinInt, m.rnd), edgeUnderflow(t, false, m.name))
	}
}

func TestD48PownAtTheIntEdgesSignalsFlags(t *testing.T) {
	c1 := newContextRound(const_ROUND_TIES_TO_EVEN)
	r1 := d128_pown_ctx(edgeDec(t, "2"), math.MinInt, c1)
	if !d128_isZER(r1) || !c1.isSet(const_UNDERFLOW) || !c1.isSet(const_INEXACT) {
		t.Errorf("pown(2, MinInt): %s flags %s", d128_format(r1, const_FORMAT_AUTO), c1.exceptionLetters())
	}
	c2 := newContextRound(const_ROUND_TIES_TO_EVEN)
	r2 := d128_pown_ctx(edgeDec(t, "2"), math.MaxInt, c2)
	if !d128_isINF(r2) || !c2.isSet(const_OVERFLOW) || !c2.isSet(const_INEXACT) {
		t.Errorf("pown(2, MaxInt): %s flags %s", d128_format(r2, const_FORMAT_AUTO), c2.exceptionLetters())
	}
}

func TestD48RootnAtMinInt(t *testing.T) {
	// x^(−1/2^63): 63 square roots then 1/z, in D48, rounded once.
	want := []struct {
		x  string
		by [5]string
	}{
		{"2", [5]string{"0.9999999999999999999248488320984705", "0.9999999999999999999248488320984705", "0.9999999999999999999248488320984705", "0.9999999999999999999248488320984706", "0.9999999999999999999248488320984705"}},
		{"0.5", [5]string{"1.000000000000000000075151167901529", "1.000000000000000000075151167901529", "1.000000000000000000075151167901529", "1.000000000000000000075151167901530", "1.000000000000000000075151167901529"}},
		{"100", [5]string{"0.9999999999999999995007064479686237", "0.9999999999999999995007064479686237", "0.9999999999999999995007064479686236", "0.9999999999999999995007064479686237", "0.9999999999999999995007064479686236"}},
		{"3", [5]string{"0.9999999999999999998808882169906763", "0.9999999999999999998808882169906763", "0.9999999999999999998808882169906763", "0.9999999999999999998808882169906764", "0.9999999999999999998808882169906763"}},
	}
	for _, w := range want {
		for i, m := range edgeModes {
			edgeCheck(t, "rootn MIN", w.x, m.name, d128_rootn_rnd(edgeDec(t, w.x), math.MinInt, m.rnd), edgeDec(t, w.by[i]))
		}
	}
	one := edgeDec(t, "1")
	upper := edgeDec(t, "1.000000000000000001")
	for _, m := range edgeModes {
		// 1^(anything) = 1, through the exact rule (preferred exponent −0 = 0).
		edgeCheck(t, "rootn MIN", "1", m.name, d128_rootn_rnd(one, math.MinInt, m.rnd), one)
		// Even root of a negative: invalid (NaN), as for any even n.
		if !d128_isNAN(d128_rootn_rnd(edgeDec(t, "-2"), math.MinInt, m.rnd)) {
			t.Errorf("rootn(-2, MIN) %s: not NaN", m.name)
		}
		// An odd n of extreme size takes the pow path: |x|^(1/n) stays next to 1.
		r := d128_rootn_rnd(edgeDec(t, "2"), math.MaxInt, m.rnd)
		if !d128_isFNZ(r) || (cmpMagFnzFnz(r, one) < 0) || (cmpMagFnzFnz(r, upper) > 0) {
			t.Errorf("rootn(2, MAX) %s: %s out of range", m.name, d128_format(r, const_FORMAT_AUTO))
		}
	}
}
