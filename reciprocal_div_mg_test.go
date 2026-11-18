package decimal128

// The Möller–Granlund DIV3BY2 step (divRem_q64r128_192x128x64_v_mg) against
// math/big, with half the inputs drawn just below the divisor, where the
// quotient estimate reaches 2^64 - 1. The step once computed its remainder as
// (n1:n0) - (q+1)*d, and q + 1 wrapped to 0 there (wrong D128 quotients
// whenever a quotient limb was all ones).

import (
	"math"
	"math/big"
	"math/bits"
	"testing"
)

func reciprocalDivMGCheck(t *testing.T, n2, n1, n0, d1, d0 uint64) {
	t.Helper()
	v := vReciprocal_64_128_mg(int64(d1), int64(d0))
	s := divRem_q64r128_192x128x64_v_mg(int64(n2), int64(n1), int64(n0), int64(d1), int64(d0), v)
	num := new(big.Int).Or(new(big.Int).Lsh(big128(int64(n1), int64(n2)), 64), new(big.Int).SetUint64(n0))
	d := big128(int64(d0), int64(d1))
	q, r := new(big.Int).QuoRem(num, d, new(big.Int))
	if ubig(s.quotDw0).Cmp(q) != 0 || big128(s.remDw0, s.remDw1).Cmp(r) != 0 {
		t.Errorf("n=%d:%d:%d d=%d:%d", n2, n1, n0, d1, d0)
	}
}

func TestReciprocalDivMGDiv3by2NearTheTopAndRandom(t *testing.T) {
	rng := splitMix64{64}
	for i := 0; i < 200_000; i++ {
		d1 := rng.next() | 1<<63
		d0 := rng.next()
		var n2, n1 uint64
		if i%2 == 0 {
			// (n2:n1) = d - 1 - small: the estimate lands on 2^64 - 1.
			a := rng.next()
			small := a >> (rng.next() % 64)
			var b uint64
			n1, b = bits.Sub64(d0, small, 0)
			n2 = d1 - b
			n1, b = bits.Sub64(n1, 1, 0)
			n2 -= b
		} else {
			n2 = rng.next() % d1
			n1 = rng.next()
		}
		reciprocalDivMGCheck(t, n2, n1, rng.next(), d1, d0)
	}
	// The extreme: (n2:n1) = d - 1 and n0 = 2^64 - 1, quotient 2^64 - 1.
	reciprocalDivMGCheck(t, math.MaxUint64, math.MaxUint64-1, math.MaxUint64, math.MaxUint64, math.MaxUint64)
	reciprocalDivMGCheck(t, 1<<63-1, math.MaxUint64, math.MaxUint64, 1<<63, 0) // (n2:n1) = d - 1
}
