// u192 and u320: the D48 domain's integer types (D48GoWorkOrder.md G-1; Swift
// U320.swift). u192 holds a D48 coefficient as a value (< 10^48 < 2^160, three
// limbs); u320 holds every D48 intermediate (< 10^96 < 2^320, five limbs). Both
// mirror u256: signed int64 limb fields in ascending order (D23, D-G2),
// arithmetic on uint64 limbs through math/bits, every multiply written out
// explicitly (u320_mul.go).

package decimal128

import "math/bits"

type u192 struct {
	dw0 int64
	dw1 int64
	dw2 int64
}

type u320 struct {
	dw0 int64
	dw1 int64
	dw2 int64
	dw3 int64
	dw4 int64
}

// u320FromDw0 is the one-limb u320 (Swift U320(dw0:)).
func u320FromDw0(dw0 int64) u320 { return u320{dw0: dw0} }

// u320FromU192 widens a u192 (zero-extended).
func u320FromU192(x u192) u320 { return u320{x.dw0, x.dw1, x.dw2, 0, 0} }

func (x u192) isZero() bool { return (x.dw0 | x.dw1 | x.dw2) == 0 }

func (x u320) isZero() bool { return (x.dw0 | x.dw1 | x.dw2 | x.dw3 | x.dw4) == 0 }

// u192 narrows to a u192; the value must fit (dw3 == dw4 == 0).
func (x u320) u192() u192 {
	verify("u320 does not fit 192 bits", func() bool { return (x.dw3 | x.dw4) == 0 })
	return u192{x.dw0, x.dw1, x.dw2}
}

// --- Compare, bit length ---

// cmp_int_320x320 is -1, 0 or 1 as x <, ==, > y (unsigned).
func cmp_int_320x320(x, y u320) int {
	if x.dw4 != y.dw4 {
		return cmpLimb(x.dw4, y.dw4)
	}
	if x.dw3 != y.dw3 {
		return cmpLimb(x.dw3, y.dw3)
	}
	if x.dw2 != y.dw2 {
		return cmpLimb(x.dw2, y.dw2)
	}
	if x.dw1 != y.dw1 {
		return cmpLimb(x.dw1, y.dw1)
	}
	if x.dw0 != y.dw0 {
		return cmpLimb(x.dw0, y.dw0)
	}
	return 0
}

// cmpLimb is -1 or 1 as x < or > y (unsigned), for x != y.
func cmpLimb(x, y int64) int {
	if uint64(x) < uint64(y) {
		return -1
	}
	return 1
}

// lt_320 is x < y (unsigned); Swift's U320 `<`. A borrow chain rather than
// cmp_int_320x320(x, y) < 0, so it fits Go's inline budget (G-7).
func lt_320(x, y u320) bool {
	_, b := bits.Sub64(uint64(x.dw0), uint64(y.dw0), 0)
	_, b = bits.Sub64(uint64(x.dw1), uint64(y.dw1), b)
	_, b = bits.Sub64(uint64(x.dw2), uint64(y.dw2), b)
	_, b = bits.Sub64(uint64(x.dw3), uint64(y.dw3), b)
	_, b = bits.Sub64(uint64(x.dw4), uint64(y.dw4), b)
	return b != 0
}

// bitLen_int_320 is the number of significant bits; 0 for x == 0. Spelled
// with bits.Len64 (= 64 - LeadingZeros64) to fit Go's inline budget (G-7).
func bitLen_int_320(x u320) int {
	if x.dw4 != 0 {
		return 256 + bits.Len64(uint64(x.dw4))
	}
	if x.dw3 != 0 {
		return 192 + bits.Len64(uint64(x.dw3))
	}
	if x.dw2 != 0 {
		return 128 + bits.Len64(uint64(x.dw2))
	}
	if x.dw1 != 0 {
		return 64 + bits.Len64(uint64(x.dw1))
	}
	return bits.Len64(uint64(x.dw0))
}

// limbLen_320 is 0 for x == 0, else 1 + the index of the top nonzero limb.
func limbLen_320(x u320) int {
	if x.dw4 != 0 {
		return 5
	}
	if x.dw3 != 0 {
		return 4
	}
	if x.dw2 != 0 {
		return 3
	}
	if x.dw1 != 0 {
		return 2
	}
	if x.dw0 != 0 {
		return 1
	}
	return 0
}

// --- Shifts ---
//
// Straight-line: pick the limbs by whole-limb shift q (a switch, no loop or
// array), then shift by s = n & 63 with the double-shift trick
// (v >> s | (w << 1) << (63 - s)), which needs no s == 0 special case. (Go
// defines a shift by 64 as 0, so the plain form would be safe too; the trick
// keeps it branch-free, as Swift.)

// limbsDown_320 is the five limbs of x moved down by q whole limbs (zero-filled).
func limbsDown_320(x u320, q int) (uint64, uint64, uint64, uint64, uint64) {
	x0, x1, x2, x3, x4 := uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), uint64(x.dw3), uint64(x.dw4)
	switch q {
	case 0:
		return x0, x1, x2, x3, x4
	case 1:
		return x1, x2, x3, x4, 0
	case 2:
		return x2, x3, x4, 0, 0
	case 3:
		return x3, x4, 0, 0, 0
	default:
		return x4, 0, 0, 0, 0
	}
}

// limbsUp_320 is the five limbs of x moved up by q whole limbs (zero-filled).
func limbsUp_320(x u320, q int) (uint64, uint64, uint64, uint64, uint64) {
	x0, x1, x2, x3, x4 := uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), uint64(x.dw3), uint64(x.dw4)
	switch q {
	case 0:
		return x0, x1, x2, x3, x4
	case 1:
		return 0, x0, x1, x2, x3
	case 2:
		return 0, 0, x0, x1, x2
	case 3:
		return 0, 0, 0, x0, x1
	default:
		return 0, 0, 0, 0, x0
	}
}

// shl_320_320 is x << n, 0 <= n < 320; bits shifted past 2^320 must be zero.
func shl_320_320(x u320, n int) u320 {
	verify("", func() bool { return (n >= 0) && (n < 320) })
	verify("shl_320_320 overflow", func() bool { return (n == 0) || (bitLen_int_320(x)+n <= 320) })
	a0, a1, a2, a3, a4 := limbsUp_320(x, n>>6)
	s := uint(n & 63)
	t := 63 - s
	return u320{
		int64(a0 << s),
		int64((a1 << s) | ((a0 >> 1) >> t)),
		int64((a2 << s) | ((a1 >> 1) >> t)),
		int64((a3 << s) | ((a2 >> 1) >> t)),
		int64((a4 << s) | ((a3 >> 1) >> t)),
	}
}

// shr_320_320 is x >> n, 0 <= n < 320.
func shr_320_320(x u320, n int) u320 {
	verify("", func() bool { return (n >= 0) && (n < 320) })
	a0, a1, a2, a3, a4 := limbsDown_320(x, n>>6)
	s := uint(n & 63)
	t := 63 - s
	return u320{
		int64((a0 >> s) | ((a1 << 1) << t)),
		int64((a1 >> s) | ((a2 << 1) << t)),
		int64((a2 >> s) | ((a3 << 1) << t)),
		int64((a3 >> s) | ((a4 << 1) << t)),
		int64(a4 >> s),
	}
}

// lowMaskLimb is the low-n-bits mask for limb i (bits 64i .. 64i+63): all
// ones, partial or none.
func lowMaskLimb(n, i int) uint64 {
	m := n - 64*i // bits of this limb below n
	if m >= 64 {
		return ^uint64(0)
	}
	if m <= 0 {
		return 0
	}
	return (uint64(1) << uint(m)) - 1
}

// lowBitsNonZero_320 is true iff any of the low n bits (0 <= n <= 320) of x
// are set.
func lowBitsNonZero_320(x u320, n int) bool {
	verify("", func() bool { return (n >= 0) && (n <= 320) })
	return ((uint64(x.dw0) & lowMaskLimb(n, 0)) | (uint64(x.dw1) & lowMaskLimb(n, 1)) |
		(uint64(x.dw2) & lowMaskLimb(n, 2)) | (uint64(x.dw3) & lowMaskLimb(n, 3)) |
		(uint64(x.dw4) & lowMaskLimb(n, 4))) != 0
}

// --- Add / sub ---

// add_320_320x320 is x + y; a 320-bit overflow is impossible.
func add_320_320x320(x, y u320) u320 {
	r0, c := bits.Add64(uint64(x.dw0), uint64(y.dw0), 0)
	r1, c := bits.Add64(uint64(x.dw1), uint64(y.dw1), c)
	r2, c := bits.Add64(uint64(x.dw2), uint64(y.dw2), c)
	r3, c := bits.Add64(uint64(x.dw3), uint64(y.dw3), c)
	r4, c := bits.Add64(uint64(x.dw4), uint64(y.dw4), c)
	if c != 0 {
		impossible("add_320_320x320 overflow")
	}
	return u320{int64(r0), int64(r1), int64(r2), int64(r3), int64(r4)}
}

// add_320_320x64 is x + y for a one-limb y; a 320-bit overflow is impossible.
func add_320_320x64(x u320, y int64) u320 {
	return add_320_320x320(x, u320FromDw0(y))
}

// sub_320_320x320 is x - y; an underflow (x < y) is impossible.
func sub_320_320x320(x, y u320) u320 {
	r0, b := bits.Sub64(uint64(x.dw0), uint64(y.dw0), 0)
	r1, b := bits.Sub64(uint64(x.dw1), uint64(y.dw1), b)
	r2, b := bits.Sub64(uint64(x.dw2), uint64(y.dw2), b)
	r3, b := bits.Sub64(uint64(x.dw3), uint64(y.dw3), b)
	r4, b := bits.Sub64(uint64(x.dw4), uint64(y.dw4), b)
	if b != 0 {
		impossible("sub_320_320x320 underflow")
	}
	return u320{int64(r0), int64(r1), int64(r2), int64(r3), int64(r4)}
}
