package decimal128

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
)

// The native logexp corpus pins exp/exp10 to a handful of exact 34-digit golden
// values. This adds breadth: a float64 cross-check over thousands of random
// arguments catches any kernel / constant / range-reduction error. (The D38
// engine's own precision is validated exactly in d38_engine_test.go; here we
// only need ~float64 agreement, well within the 34-digit result's accuracy.)

func d128FromFloat(f float64) d128 {
	return d128_parseOrNaN(strconv.FormatFloat(f, 'g', -1, 64))
}

func d128ToFloat(t *testing.T, x d128) float64 {
	f, err := strconv.ParseFloat(d128_format(x, FORMAT_AUTO), 64)
	if err != nil {
		t.Fatalf("parse result %q: %v", d128_format(x, FORMAT_AUTO), err)
	}
	return f
}

func checkRelErr(t *testing.T, name string, arg, got, want float64) {
	checkRelErrTol(t, name, arg, got, want, 1e-13)
}

func checkRelErrTol(t *testing.T, name string, arg, got, want, tol float64) {
	if want == 0 {
		if got != 0 {
			t.Fatalf("%s(%v) = %v want 0", name, arg, got)
		}
		return
	}
	if rel := math.Abs(got-want) / math.Abs(want); rel > tol {
		t.Fatalf("%s(%v) = %v want %v (rel err %g)", name, arg, got, want, rel)
	}
}

func TestExpVsFloat(t *testing.T) {
	ctx := newContextRound(ROUND_TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0xE7))
	for i := 0; i < 50000; i++ {
		f := (rng.Float64()*2 - 1) * 200 // [-200, 200]
		got := d128ToFloat(t, d128_exp_ctx(d128FromFloat(f), ctx))
		checkRelErr(t, "exp", f, got, math.Exp(f))
	}
}

func TestExp10VsFloat(t *testing.T) {
	ctx := newContextRound(ROUND_TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0xE710))
	for i := 0; i < 50000; i++ {
		f := (rng.Float64()*2 - 1) * 290 // [-290, 290], inside float64 range
		got := d128ToFloat(t, d128_exp10_ctx(d128FromFloat(f), ctx))
		checkRelErr(t, "exp10", f, got, math.Pow(10, f))
	}
}

// randPositiveFloat returns a positive float64 spanning ~10^-150 .. 10^150 so
// the log argument reduction (leading-digit + √-reduction) is well exercised.
func randPositiveFloat(rng *rand.Rand) float64 {
	return math.Pow(10, (rng.Float64()*2-1)*150) * (1 + rng.Float64()*9)
}

func TestLnVsFloat(t *testing.T) {
	ctx := newContextRound(ROUND_TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0x1A))
	for i := 0; i < 50000; i++ {
		f := randPositiveFloat(rng)
		got := d128ToFloat(t, d128_ln_ctx(d128FromFloat(f), ctx))
		checkRelErr(t, "ln", f, got, math.Log(f))
	}
}

func TestLog10VsFloat(t *testing.T) {
	ctx := newContextRound(ROUND_TIES_TO_EVEN)
	rng := rand.New(rand.NewSource(0x1A10))
	for i := 0; i < 50000; i++ {
		f := randPositiveFloat(rng)
		got := d128ToFloat(t, d128_log10_ctx(d128FromFloat(f), ctx))
		checkRelErr(t, "log10", f, got, math.Log10(f))
	}
}
