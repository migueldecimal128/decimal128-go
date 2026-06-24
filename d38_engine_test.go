package decimal128

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// The D38 engine (add/sub/div/sqrt/round) has no conformance corpus of its own —
// it is the internal substrate of the §9 transcendental functions. This test
// validates it directly against exact big.Rat arithmetic with 38-significant-
// digit half-even rounding, the contract finalize38 promises.

func bigPow10(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

func ratPow10(n int) *big.Rat { return new(big.Rat).SetInt(bigPow10(n)) }

// ratPow10e returns 10^e as a big.Rat for any (possibly negative) e.
func ratPow10e(e int) *big.Rat {
	if e >= 0 {
		return ratPow10(e)
	}
	return new(big.Rat).Inv(ratPow10(-e))
}

// decExp returns floor(log10(a)) for a > 0 — the true decimal exponent, which
// digit-counting overestimates by one just below a power of ten.
func decExp(a *big.Rat) int {
	n := len(a.Num().Text(10)) - len(a.Denom().Text(10))
	for a.Cmp(ratPow10e(n)) < 0 {
		n--
	}
	for a.Cmp(ratPow10e(n+1)) >= 0 {
		n++
	}
	return n
}

func u128ToBig(lo, hi int64) *big.Int {
	b := new(big.Int).SetUint64(uint64(hi))
	b.Lsh(b, 64)
	return b.Or(b, new(big.Int).SetUint64(uint64(lo)))
}

func d38ToRat(x d38) *big.Rat {
	r := new(big.Rat).SetInt(u128ToBig(x.coeffLo64, x.coeffHi64))
	if q := d38_qExp(x); q >= 0 {
		r.Mul(r, ratPow10(q))
	} else {
		r.Quo(r, ratPow10(-q))
	}
	if d38_signFlag(x) {
		r.Neg(r)
	}
	return r
}

func d38FromBig(neg bool, coeff *big.Int, qExp int) d38 {
	lo := new(big.Int).And(coeff, new(big.Int).SetUint64(math.MaxUint64))
	hi := new(big.Int).Rsh(coeff, 64)
	return d38Pack(neg, qExp, int64(lo.Uint64()), int64(hi.Uint64()))
}

// divRoundHalfEven rounds N/D (N≥0, D>0) to the nearest integer, ties to even.
func divRoundHalfEven(n, d *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(n, d, new(big.Int))
	twoR := new(big.Int).Lsh(r, 1)
	c := twoR.Cmp(d)
	if c > 0 || (c == 0 && q.Bit(0) == 1) {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// round38 returns the value of R rounded to 38 significant digits, half-even,
// as an exact big.Rat.
func round38(r *big.Rat) *big.Rat {
	if r.Sign() == 0 {
		return new(big.Rat)
	}
	neg := r.Sign() < 0
	a := new(big.Rat).Abs(r)
	p, q := a.Num(), a.Denom()
	// e places the result on the 38-significant-digit grid: C ∈ [10^37, 10^38).
	e := decExp(a) - 37
	var num, den *big.Int
	if e >= 0 {
		num, den = p, new(big.Int).Mul(q, bigPow10(e))
	} else {
		num, den = new(big.Int).Mul(p, bigPow10(-e)), q
	}
	c := divRoundHalfEven(num, den)
	if c.Cmp(bigPow10(38)) >= 0 {
		// Rounded up across the decade (C == 10^38): renormalise to 10^37.
		c = bigPow10(37)
		e++
	}
	val := new(big.Rat).Mul(new(big.Rat).SetInt(c), ratPow10e(e))
	if neg {
		val.Neg(val)
	}
	return val
}

// randD38 builds a random finite non-zero D38 value (1..38 digits).
func randD38(rng *rand.Rand) d38 {
	digits := 1 + rng.Intn(38)
	lo, hi := bigPow10(digits-1), bigPow10(digits)
	span := new(big.Int).Sub(hi, lo)
	coeff := new(big.Int).Add(lo, new(big.Int).Rand(rng, span))
	qExp := rng.Intn(121) - 60
	return d38FromBig(rng.Intn(2) == 1, coeff, qExp)
}

func TestD38EngineAddSubDiv(t *testing.T) {
	rng := rand.New(rand.NewSource(0xD38ADD))
	for i := 0; i < 20000; i++ {
		x, y := randD38(rng), randD38(rng)
		xr, yr := d38ToRat(x), d38ToRat(y)

		add := d38ToRat(d38_add(x, y))
		if want := round38(new(big.Rat).Add(xr, yr)); add.Cmp(want) != 0 {
			t.Fatalf("add: %s + %s = %s want %s", xr, yr, add, want)
		}
		sub := d38ToRat(d38_subtract(x, y))
		if want := round38(new(big.Rat).Sub(xr, yr)); sub.Cmp(want) != 0 {
			t.Fatalf("sub: %s - %s = %s want %s", xr, yr, sub, want)
		}
		div := d38ToRat(d38_divide(x, y))
		if want := round38(new(big.Rat).Quo(xr, yr)); div.Cmp(want) != 0 {
			t.Fatalf("div: %s / %s = %s want %s", xr, yr, div, want)
		}
	}
}

func TestD38EngineSqrt(t *testing.T) {
	rng := rand.New(rand.NewSource(0xD38597))
	for i := 0; i < 20000; i++ {
		x := randD38(rng)
		x.qExpAndSignBit &^= 1 // force non-negative
		v := d38ToRat(x)
		r := d38_sqrt(x)
		rr := d38ToRat(r)
		// Correctly rounded iff (rr − ½ulp)² ≤ v ≤ (rr + ½ulp)².
		half := new(big.Rat).SetInt(bigPow10(0))
		if q := d38_qExp(r); q >= 0 {
			half.Mul(half, ratPow10(q))
		} else {
			half.Quo(half, ratPow10(-q))
		}
		half.Quo(half, big.NewRat(2, 1))
		lo := new(big.Rat).Sub(rr, half)
		hi := new(big.Rat).Add(rr, half)
		lo2 := new(big.Rat).Mul(lo, lo)
		hi2 := new(big.Rat).Mul(hi, hi)
		if lo2.Cmp(v) > 0 || hi2.Cmp(v) < 0 {
			t.Fatalf("sqrt: √(%s) = %s not within ½ulp", v, rr)
		}
	}
}

func TestD38EngineRoundTiesToAway(t *testing.T) {
	rng := rand.New(rand.NewSource(0xD38127))
	for i := 0; i < 20000; i++ {
		x := randD38(rng)
		got := d38ToRat(d38_roundToIntegralTiesToAway(x))
		// Oracle: round to nearest integer, ties away from zero.
		r := d38ToRat(x)
		neg := r.Sign() < 0
		a := new(big.Rat).Abs(r)
		quo, rem := new(big.Int).QuoRem(a.Num(), a.Denom(), new(big.Int))
		if new(big.Int).Lsh(rem, 1).Cmp(a.Denom()) >= 0 {
			quo.Add(quo, big.NewInt(1))
		}
		if neg {
			quo.Neg(quo)
		}
		if want := new(big.Rat).SetInt(quo); got.Cmp(want) != 0 {
			t.Fatalf("round: round(%s) = %s want %s", r, got, want)
		}
	}
}

func TestD38ToLongOrMin(t *testing.T) {
	cases := []struct {
		x    d38
		want int64
	}{
		{d38Pack(false, 0, 0, 0), 0},
		{d38Pack(false, 0, 12345, 0), 12345},
		{d38Pack(true, 0, 12345, 0), -12345},
		{d38Pack(false, 3, 12, 0), 12000},            // 12·10^3
		{d38Pack(false, -2, 4200, 0), 42},            // 4200·10^-2 = 42 exact
		{d38Pack(false, -2, 4201, 0), math.MinInt64}, // 42.01 not integral
		{d38Pack(false, -1, 5, 0), math.MinInt64},    // 0.5 < 1
	}
	for _, c := range cases {
		if got := d38_toLongOrMin(c.x); got != c.want {
			t.Fatalf("toLong(%s) = %d want %d", d38_format(c.x), got, c.want)
		}
	}
}
