// The U128 core vocabulary (PrimitiveLayer.md §5, D5): the uniformly named
// operations through which core expresses all 128-bit arithmetic. The limb forms
// (xDw0, xDw1, … — operand-major, ascending per D23) are the uniform API.
//
// Swift reconstructs a native UInt128 and uses &+/&-/&* (wrapping). Go has no
// uint128, so the bodies synthesize from uint64 limbs via math/bits
// (Add64/Sub64/Mul64 carry chains) — the Java/Kotlin model. Arithmetic is
// wrapping (bit-bucket semantics).

package decimal128

import "math/bits"

func add_128_128x128(xDw0, xDw1, yDw0, yDw1 int64) u128 {
	lo, c := bits.Add64(uint64(xDw0), uint64(yDw0), 0)
	hi, _ := bits.Add64(uint64(xDw1), uint64(yDw1), c)
	return u128FromU64(lo, hi)
}

func sub_128_128x128(xDw0, xDw1, yDw0, yDw1 int64) u128 {
	lo, b := bits.Sub64(uint64(xDw0), uint64(yDw0), 0)
	hi, _ := bits.Sub64(uint64(xDw1), uint64(yDw1), b)
	return u128FromU64(lo, hi)
}

func mul_128_128x128(xDw0, xDw1, yDw0, yDw1 int64) u128 {
	// Low 128 bits of the full product: x0*y0 (128) plus the wrapped low halves
	// of the cross terms shifted into the high limb.
	hi, lo := bits.Mul64(uint64(xDw0), uint64(yDw0))
	hi += uint64(xDw0)*uint64(yDw1) + uint64(xDw1)*uint64(yDw0)
	return u128FromU64(lo, hi)
}

func shl_128_128(xDw0, xDw1 int64, n int) u128 {
	verify("", func() bool { return n >= 0 && n < 128 })
	lo := uint64(xDw0)
	hi := uint64(xDw1)
	if n == 0 {
		return u128FromU64(lo, hi)
	}
	if n < 64 {
		newHi := (hi << uint(n)) | (lo >> uint(64-n))
		newLo := lo << uint(n)
		return u128FromU64(newLo, newHi)
	}
	// 64 <= n < 128
	return u128FromU64(0, lo<<uint(n-64))
}

func shr_128_128(xDw0, xDw1 int64, n int) u128 {
	verify("", func() bool { return n >= 0 && n < 128 })
	lo := uint64(xDw0)
	hi := uint64(xDw1)
	if n == 0 {
		return u128FromU64(lo, hi)
	}
	if n < 64 {
		newLo := (lo >> uint(n)) | (hi << uint(64-n))
		newHi := hi >> uint(n)
		return u128FromU64(newLo, newHi)
	}
	// 64 <= n < 128
	return u128FromU64(hi>>uint(n-64), 0)
}

func and_128_128x128(xDw0, xDw1, yDw0, yDw1 int64) u128 {
	return u128{dw0: xDw0 & yDw0, dw1: xDw1 & yDw1}
}

func or_128_128x128(xDw0, xDw1, yDw0, yDw1 int64) u128 {
	return u128{dw0: xDw0 | yDw0, dw1: xDw1 | yDw1}
}

func not_128_128(xDw0, xDw1 int64) u128 {
	return u128{dw0: ^xDw0, dw1: ^xDw1}
}

func cmp_int_128x128(xDw0, xDw1, yDw0, yDw1 int64) int {
	if uint64(xDw1) != uint64(yDw1) {
		if uint64(xDw1) < uint64(yDw1) {
			return -1
		}
		return 1
	}
	if uint64(xDw0) != uint64(yDw0) {
		if uint64(xDw0) < uint64(yDw0) {
			return -1
		}
		return 1
	}
	return 0
}

func isZero_bool_128(xDw0, xDw1 int64) bool {
	return (xDw0 | xDw1) == 0
}

// mul_128_64x64 is the full-width 64×64→128 product (PrimitiveLayer.md §4.2).
func mul_128_64x64(x, y int64) u128 {
	hi, lo := bits.Mul64(uint64(x), uint64(y))
	return u128FromU64(lo, hi)
}

// --- primitive-interior typed spellings (§3.4); not uniform API ---

func shlU128(x u128, n int) u128 { return shl_128_128(x.dw0, x.dw1, n) }
func shrU128(x u128, n int) u128 { return shr_128_128(x.dw0, x.dw1, n) }
func orU128(a, b u128) u128      { return or_128_128x128(a.dw0, a.dw1, b.dw0, b.dw1) }

// sub128Borrow returns x-y and the borrow-out (1 if y>x), for the subAbs family.
func sub128Borrow(x, y u128) (u128, uint64) {
	lo, b := bits.Sub64(uint64(x.dw0), uint64(y.dw0), 0)
	hi, b := bits.Sub64(uint64(x.dw1), uint64(y.dw1), b)
	return u128FromU64(lo, hi), b
}
