// The add/sub/subAbs family (PrimitiveLayer.md §4.2), D9 segment grammar
// op_result_operands. Limb forms (operand-major, ascending per D23) are the
// uniform API. Swift reconstructs UInt128 halves; Go runs uint64 carry/borrow
// chains via math/bits.

package decimal128

import "math/bits"

// add_256_128x128 returns x + y as a u256; cannot overflow.
func add_256_128x128(xDw0, xDw1, yDw0, yDw1 int64) u256 {
	r0, c := bits.Add64(uint64(xDw0), uint64(yDw0), 0)
	r1, c := bits.Add64(uint64(xDw1), uint64(yDw1), c)
	return u256{int64(r0), int64(r1), int64(c), 0}
}

// add_256_256x64 returns x + y, trapping on 256-bit overflow.
func add_256_256x64(xDw0, xDw1, xDw2, xDw3, y int64) u256 {
	r0, c := bits.Add64(uint64(xDw0), uint64(y), 0)
	r1, c := bits.Add64(uint64(xDw1), 0, c)
	r2, c := bits.Add64(uint64(xDw2), 0, c)
	r3, c := bits.Add64(uint64(xDw3), 0, c)
	if c != 0 {
		impossible("add_256_256x64 overflow")
	}
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}

// add_256_256x128 returns x + y, trapping on 256-bit overflow.
func add_256_256x128(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1 int64) u256 {
	r0, c := bits.Add64(uint64(xDw0), uint64(yDw0), 0)
	r1, c := bits.Add64(uint64(xDw1), uint64(yDw1), c)
	r2, c := bits.Add64(uint64(xDw2), 0, c)
	r3, c := bits.Add64(uint64(xDw3), 0, c)
	if c != 0 {
		impossible("add_256_256x128 overflow")
	}
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}

// add_256_256x256 returns x + y, trapping on 256-bit overflow.
func add_256_256x256(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1, yDw2, yDw3 int64) u256 {
	r0, c := bits.Add64(uint64(xDw0), uint64(yDw0), 0)
	r1, c := bits.Add64(uint64(xDw1), uint64(yDw1), c)
	r2, c := bits.Add64(uint64(xDw2), uint64(yDw2), c)
	r3, c := bits.Add64(uint64(xDw3), uint64(yDw3), c)
	if c != 0 {
		impossible("add_256_256x256 overflow")
	}
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}

// sub_256_256x256 returns x - y, trapping on underflow.
func sub_256_256x256(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1, yDw2, yDw3 int64) u256 {
	d, b := sub256Borrow(
		u256{xDw0, xDw1, xDw2, xDw3},
		u256{yDw0, yDw1, yDw2, yDw3},
	)
	if b != 0 {
		impossible("sub_256_256x256 underflow")
	}
	return d
}

// subAbs_d128swap_128x128 returns |x - y| and whether the operands were swapped
// (y > x). The borrow flag is the swap flag (D14).
func subAbs_d128swap_128x128(xDw0, xDw1, yDw0, yDw1 int64) diff128Swap {
	diff, borrow := sub128Borrow(u128{xDw0, xDw1}, u128{yDw0, yDw1})
	swap := borrow != 0
	if swap {
		diff, _ = sub128Borrow(u128{0, 0}, diff) // negate: |x-y| = 0 - (x-y)
	}
	return diff128Swap{diff.dw0, diff.dw1, swap}
}

// subAbs_d256swap_256x128 returns |x - y| and the swap flag (y > x).
func subAbs_d256swap_256x128(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1 int64) diff256Swap {
	diff, borrow := sub256Borrow(
		u256{xDw0, xDw1, xDw2, xDw3},
		u256{yDw0, yDw1, 0, 0},
	)
	return absDiff256(diff, borrow)
}

// subAbs_d256swap_256x256 returns |x - y| and the swap flag (y > x).
func subAbs_d256swap_256x256(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1, yDw2, yDw3 int64) diff256Swap {
	diff, borrow := sub256Borrow(
		u256{xDw0, xDw1, xDw2, xDw3},
		u256{yDw0, yDw1, yDw2, yDw3},
	)
	return absDiff256(diff, borrow)
}

// --- primitive-interior helpers (§3.4) ---

// sub256Borrow returns x-y and the borrow-out (1 if y>x).
func sub256Borrow(x, y u256) (u256, uint64) {
	r0, b := bits.Sub64(uint64(x.dw0), uint64(y.dw0), 0)
	r1, b := bits.Sub64(uint64(x.dw1), uint64(y.dw1), b)
	r2, b := bits.Sub64(uint64(x.dw2), uint64(y.dw2), b)
	r3, b := bits.Sub64(uint64(x.dw3), uint64(y.dw3), b)
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}, b
}

// absDiff256 packages a 256-bit difference and its borrow into a diff256Swap,
// conditionally negating so the magnitude is |x-y|.
func absDiff256(diff u256, borrow uint64) diff256Swap {
	swap := borrow != 0
	if swap {
		diff, _ = sub256Borrow(u256{0, 0, 0, 0}, diff)
	}
	return diff256Swap{diff.dw0, diff.dw1, diff.dw2, diff.dw3, swap}
}

// cmp_int_256x256 compares two 256-bit values as unsigned magnitudes, returning
// -1, 0, or 1. Sibling of cmp_int_128x128, one tier up.
func cmp_int_256x256(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1, yDw2, yDw3 int64) int {
	if uint64(xDw3) != uint64(yDw3) {
		if uint64(xDw3) < uint64(yDw3) {
			return -1
		}
		return 1
	}
	if uint64(xDw2) != uint64(yDw2) {
		if uint64(xDw2) < uint64(yDw2) {
			return -1
		}
		return 1
	}
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
