package decimal128

// Correctness sweep for the Möller–Granlund reciprocal divides that replaced
// Knuth Algorithm D. The oracle is the independent round-trip identity
// q*y + r == x with 0 <= r < y, which uniquely determines (q, r) — so these
// checks need no reference divider. Covers all three production entry points:
//   - div_256_128_q128r128_mg   (≤128-bit-quotient divide hot path, Alg. 5 ×2)
//   - div_256_128_q256r128_mg   (full-quotient sqrt/D38 path, Alg. 5 loop)
//   - divRem_q256r64_256x64     (33–64-bit divisor arm, Alg. 4 ×4)

import (
	"math/big"
	"testing"
)

// u256DivMGRoundTrip checks q*y + r == x and r < y.
func u256DivMGRoundTrip(t *testing.T, x u256, y, q, r *big.Int, label string) {
	t.Helper()
	if r.Cmp(y) >= 0 {
		t.Errorf("remainder >= divisor (%s)", label)
	}
	if new(big.Int).Add(new(big.Int).Mul(q, y), r).Cmp(big256v(x)) != 0 {
		t.Errorf("round-trip q*y+r != x (%s)", label)
	}
}

func u256DivMGRandomDivisor(rng *splitMix64) *big.Int {
	// Random, forced > 2^64, varied top-limb bit-length.
	shift := uint(rng.rangeIn(0, 63))
	n := rng.next()
	if n == 0 {
		n = 1
	}
	yHi := (n >> shift) | 1
	return big128(int64(rng.next()), int64(yHi))
}

// MARK: - 2-step divide hot path (q ≤ 128 bits)

func TestU256DivMGDivHotPath_classN_randomized(t *testing.T) {
	// class-N: scaledNumerator = cx * 10^34, divisor = cy, cx, cy ∈ [10^33, 10^34)
	// → ~34-digit quotient (< 2^128) and cy > 2^96 (the common D128 divide shape).
	rng := splitMix64{0x0D1F_0001}
	lo33, p34 := ten(33), ten(34)
	span := new(big.Int).Sub(p34, lo33)
	coeff := func() *big.Int {
		return new(big.Int).Add(lo33, new(big.Int).Mod(big128(int64(rng.next()), int64(rng.next())), span))
	}
	for i := 0; i < 50_000; i++ {
		cx, cy := coeff(), coeff()
		x := u256Of(new(big.Int).Mul(cx, p34))
		r := div_256_128_q128r128_mg(x, u128Of(cy))
		u256DivMGRoundTrip(t, x, cy, big128(r.quotDw0, r.quotDw1), big128(r.remDw0, r.remDw1), "classN")
	}
}

func TestU256DivMGDivHotPath_broadSweep(t *testing.T) {
	rng := splitMix64{0x0D1F_0002}
	checked := 0
	for i := 0; i < 200_000; i++ {
		y := u256DivMGRandomDivisor(&rng)
		// Dividend: random 256-bit, varied size.
		xShift := uint(rng.rangeIn(0, 127))
		xLo := big128(int64(rng.next()), int64(rng.next()))
		xHi := new(big.Int).Rsh(big128(int64(rng.next()), int64(rng.next())), xShift)
		// In-contract for the 2-step wrapper: x > 2^128 and quotient ≤ 128 bits (xHi < y).
		if xHi.Sign() == 0 || xHi.Cmp(y) >= 0 {
			continue
		}
		x := u256Of(new(big.Int).Or(new(big.Int).Lsh(xHi, 128), xLo))
		r := div_256_128_q128r128_mg(x, u128Of(y))
		u256DivMGRoundTrip(t, x, y, big128(r.quotDw0, r.quotDw1), big128(r.remDw0, r.remDw1), "broad2step")
		checked++
	}
	if checked <= 50_000 {
		t.Errorf("too few in-contract cases (%d)", checked)
	}
}

// MARK: - Full-quotient path (q up to ~192 bits; sqrt / D38)

func TestU256DivMGDivFullQuotient_broadSweep(t *testing.T) {
	rng := splitMix64{0x0D1F_0003}
	checked := 0
	for i := 0; i < 200_000; i++ {
		y := u256DivMGRandomDivisor(&rng)
		x := u256{int64(rng.next()), int64(rng.next()), int64(rng.next()), int64(rng.next())}
		// General path contract: x > 2^128 and y > 2^64. Quotient may exceed 128 bits.
		if x.dw2 == 0 && x.dw3 == 0 {
			continue
		}
		r := div_256_128_q256r128_mg(x, u128Of(y))
		u256DivMGRoundTrip(t, x, y, big256(r.quotDw0, r.quotDw1, r.quotDw2, r.quotDw3), big128(r.remDw0, r.remDw1), "fullQuot")
		checked++
	}
	if checked <= 50_000 {
		t.Errorf("too few in-contract cases (%d)", checked)
	}
}

// MARK: - 33–64-bit divisor arm (Alg. 4 ×4, via the canonical dispatcher)

func TestU256DivMGDiv256by64_arm_broadSweep(t *testing.T) {
	rng := splitMix64{0x0D1F_0004}
	for i := 0; i < 200_000; i++ {
		// 33–64-bit divisor, not a power of two (so it reaches the M&G arm).
		d0 := rng.next() | 1<<32
		if d0&(d0-1) == 0 {
			d0 |= 3
		}
		x := u256{int64(rng.next()), int64(rng.next()), int64(rng.next()), int64(rng.next())}
		r := divRem_q256r64_256x64(x.dw0, x.dw1, x.dw2, x.dw3, int64(d0))
		u256DivMGRoundTrip(t, x, new(big.Int).SetUint64(d0), big256(r.quotDw0, r.quotDw1, r.quotDw2, r.quotDw3), ubig(r.remDw0), "arm256by64")
	}
}
