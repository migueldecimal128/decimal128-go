package decimal128

import (
	"math/big"
	"testing"
)

// divRem_q128r64_192x64 (PrimitiveLayer.md D54) against a math/big oracle, and
// the one-word-divisor branch of u256DivRem_128 / u256DivResidue_128 against
// the old full-quotient path (u256DivRem128 + 128-bit residue) that sqrt/D38 keep.

var div192Rng uint64 = 0x0D54D54D54D54D54

func div192NextRand() uint64 {
	div192Rng ^= div192Rng << 13
	div192Rng ^= div192Rng >> 7
	div192Rng ^= div192Rng << 17
	return div192Rng
}

func div192Big(x0, x1, x2 uint64) *big.Int {
	v := new(big.Int).SetUint64(x2)
	v.Lsh(v, 64).Or(v, new(big.Int).SetUint64(x1))
	v.Lsh(v, 64).Or(v, new(big.Int).SetUint64(x0))
	return v
}

// Round-trip oracle: q*y + r == x and 0 <= r < y uniquely determine (q, r).
func div192CheckPrimitive(t *testing.T, x0, x1, x2, y uint64) {
	t.Helper()
	qr := divRem_q128r64_192x64(int64(x0), int64(x1), int64(x2), int64(y))
	q := div192Big(uint64(qr.quotDw0), uint64(qr.quotDw1), 0)
	r := new(big.Int).SetUint64(uint64(qr.remDw0))
	d := new(big.Int).SetUint64(y)
	back := new(big.Int).Mul(q, d)
	back.Add(back, r)
	if back.Cmp(div192Big(x0, x1, x2)) != 0 || r.Cmp(d) >= 0 {
		t.Fatalf("divRem_q128r64_192x64(%016x:%016x:%016x / %016x) round-trip fails", x2, x1, x0, y)
	}
}

// New one-word branch vs the old full-quotient path: identical q, r, residue.
func div192CheckDifferential(t *testing.T, x0, x1, x2, y uint64) {
	t.Helper()
	x := u256{int64(x0), int64(x1), int64(x2), 0}
	d := u128{int64(y), 0}
	old := u256DivRem128(x, d)
	oldRes := residueFromRemainderDivisor_128(old.remDw0, old.remDw1, d.dw0, d.dw1)
	got := u256DivRem_128(x, d)
	if got.quotDw0 != old.quotDw0 || got.quotDw1 != old.quotDw1 || old.quotDw2 != 0 ||
		old.quotDw3 != 0 || got.remDw0 != old.remDw0 || got.remDw1 != old.remDw1 {
		t.Fatalf("u256DivRem_128 differs from old path (y=%016x)", y)
	}
	gr := u256DivResidue_128(x, d)
	if gr.quotDw0 != old.quotDw0 || gr.quotDw1 != old.quotDw1 || gr.residue != oldRes {
		t.Fatalf("u256DivResidue_128 differs from old path (y=%016x)", y)
	}
}

// Random (x2, y) with x2 <u y, the primitive's precondition.
func div192RandCase() (x0, x1, x2, y uint64) {
	bits := 1 + int(div192NextRand()%64)
	uy := div192NextRand() >> (64 - bits)
	if uy < 2 {
		uy = 2
	}
	x0 = div192NextRand()
	x1 = div192NextRand()
	x2 = div192NextRand() % uy
	return x0, x1, x2, uy
}

// Decimal digit count of the 128-bit value (c1:c0); 0 for zero.
func div192DigitLen(c0, c1 uint64) int {
	n := 0
	for c0 != 0 || c1 != 0 {
		qr := divRem_q128r64_128x64(int64(c0), int64(c1), 10)
		c0, c1 = uint64(qr.quotDw0), uint64(qr.quotDw1)
		n++
	}
	return n
}

func TestDiv192(t *testing.T) {
	const m = ^uint64(0)
	ys := []uint64{1, 2, 3, 10, 0xFFFF, 0x10000, 0xFFFFFFFF, 0x100000000,
		1 << 62, 1 << 63, m, 999999999999999999, 1000000000000000000}
	lows := [][2]uint64{{0, 0}, {m, m}, {1, 0}, {m, 0}, {0, m}}
	for _, y := range ys {
		for _, x2 := range []uint64{0, 1, y / 2, y - 1} {
			if x2 >= y {
				continue
			}
			for _, lo := range lows {
				div192CheckPrimitive(t, lo[0], lo[1], x2, y)
				div192CheckDifferential(t, lo[0], lo[1], x2, y)
			}
		}
	}
	for i := 0; i < 500000; i++ {
		x0, x1, x2, y := div192RandCase()
		div192CheckPrimitive(t, x0, x1, x2, y)
		div192CheckDifferential(t, x0, x1, x2, y)
	}
	// The d128 divide's own inputs: a coefficient of dx digits scaled by 10^k
	// over every one-word divisor width dy (1..19 digits), k making the
	// quotient 34-35 digits (the WD scaled path).
	for i := 0; i < 200000; i++ {
		dy := 1 + int(div192NextRand()%19)
		dx := 1 + int(div192NextRand()%34)
		lo := uint64(pow10_128_dw0(dy - 1))
		y := lo + div192NextRand()%(uint64(pow10_128_dw0(dy))-lo)
		c0, c1 := div192NextRand(), div192NextRand()>>15
		for div192DigitLen(c0, c1) > dx {
			c0, c1 = c0>>3|c1<<61, c1>>3
		}
		if c0 == 0 && c1 == 0 {
			continue
		}
		k := 34 - (div192DigitLen(c0, c1) - dy)
		x := mulPow10_256_128(int64(c0), int64(c1), k)
		if x.dw3 != 0 {
			t.Fatalf("scaled dividend exceeds 192 bits (dx=%d dy=%d k=%d)", dx, dy, k)
		}
		div192CheckDifferential(t, uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), y)
	}
}
