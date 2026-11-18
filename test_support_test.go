package decimal128

// Shared test helpers: the math/big views of int64 limbs that Swift's
// BigIntHelpers.swift gives, a SplitMix64 RNG, and expectTrap, which requires a
// call to abort (impossible panics), as Swift's #expect(processExitsWith:) does.

import (
	"math/big"
	"testing"
)

// ubig is the unsigned value of a 64-bit limb.
func ubig(x int64) *big.Int { return new(big.Int).SetUint64(uint64(x)) }

// big128 is the unsigned value of (dw1:dw0).
func big128(dw0, dw1 int64) *big.Int {
	r := new(big.Int).Lsh(ubig(dw1), 64)
	return r.Or(r, ubig(dw0))
}

// big256 is the unsigned value of (dw3:dw2:dw1:dw0).
func big256(dw0, dw1, dw2, dw3 int64) *big.Int {
	r := new(big.Int).Lsh(big128(dw2, dw3), 128)
	return r.Or(r, big128(dw0, dw1))
}

func big256v(x u256) *big.Int { return big256(x.dw0, x.dw1, x.dw2, x.dw3) }

// limbOf is limb i (0 = lowest) of a non-negative x, as a signed dword.
func limbOf(x *big.Int, i uint) int64 {
	m := new(big.Int).Rsh(x, 64*i)
	return int64(m.And(m, new(big.Int).SetUint64(^uint64(0))).Uint64())
}

func u256Of(x *big.Int) u256 {
	return u256{dw0: limbOf(x, 0), dw1: limbOf(x, 1), dw2: limbOf(x, 2), dw3: limbOf(x, 3)}
}

func ten(k int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil) }

func pow2(n uint) *big.Int { return new(big.Int).Lsh(big.NewInt(1), n) }

func bigStr(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("bad decimal " + s)
	}
	return v
}

// splitMix64 is a deterministic test RNG, so a failure reproduces.
type splitMix64 struct{ state uint64 }

func (r *splitMix64) next() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// rangeIn is uniform in lo..=hi (modulo bias is irrelevant for tests).
func (r *splitMix64) rangeIn(lo, hi int) int { return lo + int(r.next()%uint64(hi-lo+1)) }

// below is a value in [0, n), n > 0.
func (r *splitMix64) below(n *big.Int) *big.Int {
	x := new(big.Int)
	for i := 0; i <= n.BitLen()/64+1; i++ {
		x.Lsh(x, 64).Or(x, new(big.Int).SetUint64(r.next()))
	}
	return x.Mod(x, n)
}

// expectTrap requires f to abort.
func expectTrap(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("expected a trap: %s", what)
		}
	}()
	f()
}

// mustDec parses a decimal literal the test knows is valid.
func mustDec(t *testing.T, s string) d128 {
	t.Helper()
	x, ok := d128_parse(s)
	if !ok {
		t.Fatalf("bad literal %q", s)
	}
	return x
}

// s64 reinterprets an unsigned 64-bit pattern as a signed dword.
func s64(u uint64) int64 { return int64(u) }

type bigIntT = big.Int

func u128Of(x *big.Int) u128 { return u128{dw0: limbOf(x, 0), dw1: limbOf(x, 1)} }
