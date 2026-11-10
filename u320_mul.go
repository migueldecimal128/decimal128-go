// u320 multiplies for the D48 domain (D48GoWorkOrder.md G-2; Swift
// U320Mul.swift). Every product is written out as 64x64 -> 128 limb products
// (bits.Mul64) in comba (column) order into the three-word column accumulator
// of macCol512 (div_pow10_recipmul256.go), so a column sum never loses a carry.

package decimal128

import "math/bits"

// mul_320_192x192 is x · y for two D48 coefficients (each < 2^160, so the
// product is < 2^320): the 3x3 comba, nine limb products.
func mul_320_192x192(x, y u192) u320 {
	a0, a1, a2 := uint64(x.dw0), uint64(x.dw1), uint64(x.dw2)
	b0, b1, b2 := uint64(y.dw0), uint64(y.dw1), uint64(y.dw2)
	var acc0, acc1, acc2 uint64

	macCol512(a0, b0, &acc0, &acc1, &acc2)
	r0 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a0, b1, &acc0, &acc1, &acc2)
	macCol512(a1, b0, &acc0, &acc1, &acc2)
	r1 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a0, b2, &acc0, &acc1, &acc2)
	macCol512(a1, b1, &acc0, &acc1, &acc2)
	macCol512(a2, b0, &acc0, &acc1, &acc2)
	r2 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a1, b2, &acc0, &acc1, &acc2)
	macCol512(a2, b1, &acc0, &acc1, &acc2)
	r3 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a2, b2, &acc0, &acc1, &acc2)
	r4 := acc0
	r5 := acc1
	verify("mul_320_192x192 overflow", func() bool { return r5 == 0 })
	return u320{int64(r0), int64(r1), int64(r2), int64(r3), int64(r4)}
}

// mulWide_640_320x320 is the full 640-bit product x · y as (lo, hi) halves:
// the 5x5 comba, 25 limb products. The recipmul divPow10 kernel's reciprocal
// multiply.
func mulWide_640_320x320(x, y u320) (lo, hi u320) {
	a0, a1, a2, a3, a4 := uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), uint64(x.dw3), uint64(x.dw4)
	b0, b1, b2, b3, b4 := uint64(y.dw0), uint64(y.dw1), uint64(y.dw2), uint64(y.dw3), uint64(y.dw4)
	var acc0, acc1, acc2 uint64

	macCol512(a0, b0, &acc0, &acc1, &acc2)
	r0 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a0, b1, &acc0, &acc1, &acc2)
	macCol512(a1, b0, &acc0, &acc1, &acc2)
	r1 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a0, b2, &acc0, &acc1, &acc2)
	macCol512(a1, b1, &acc0, &acc1, &acc2)
	macCol512(a2, b0, &acc0, &acc1, &acc2)
	r2 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a0, b3, &acc0, &acc1, &acc2)
	macCol512(a1, b2, &acc0, &acc1, &acc2)
	macCol512(a2, b1, &acc0, &acc1, &acc2)
	macCol512(a3, b0, &acc0, &acc1, &acc2)
	r3 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a0, b4, &acc0, &acc1, &acc2)
	macCol512(a1, b3, &acc0, &acc1, &acc2)
	macCol512(a2, b2, &acc0, &acc1, &acc2)
	macCol512(a3, b1, &acc0, &acc1, &acc2)
	macCol512(a4, b0, &acc0, &acc1, &acc2)
	r4 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a1, b4, &acc0, &acc1, &acc2)
	macCol512(a2, b3, &acc0, &acc1, &acc2)
	macCol512(a3, b2, &acc0, &acc1, &acc2)
	macCol512(a4, b1, &acc0, &acc1, &acc2)
	r5 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a2, b4, &acc0, &acc1, &acc2)
	macCol512(a3, b3, &acc0, &acc1, &acc2)
	macCol512(a4, b2, &acc0, &acc1, &acc2)
	r6 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a3, b4, &acc0, &acc1, &acc2)
	macCol512(a4, b3, &acc0, &acc1, &acc2)
	r7 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a4, b4, &acc0, &acc1, &acc2)
	r8 := acc0
	r9 := acc1

	lo = u320{int64(r0), int64(r1), int64(r2), int64(r3), int64(r4)}
	hi = u320{int64(r5), int64(r6), int64(r7), int64(r8), int64(r9)}
	return lo, hi
}

// --- Width-aware multiply by 10^k (Swift D-4) ---

// mulPow10_320 is x · 10^k, 0 <= k <= 96; the product must fit 320 bits.
//
// 10^k is read at its own width, never built as a five-limb value: 1 limb for
// k <= 19 (10^19 < 2^64), 2 for k <= 38 (region A), 3 for k <= 57 (10^57 <
// 2^192; region B's top dword is zero), 4 for k <= 77 (region B), 5 for
// 78..96 (region C). Only x's nonzero limbs take part, so a three-limb D48
// coefficient times 10^k (k <= 19) costs three limb products, not fifteen.
// Region B's zero low dword for k >= 64 is part of the value (2^64 divides
// 10^k), not padding, and is not special-cased.
func mulPow10_320(x u320, k int) u320 {
	verify("", func() bool { return (k >= 0) && (k <= 96) })
	if k == 0 {
		return x
	}
	// A three-limb x (every D48 coefficient) times a three-limb 10^k (k <= 57):
	// the straight-line 3x3 product (D48GoWorkOrder.md Phase 5, option A; Go
	// won't inline the mulTrunc_320 loop).
	if ((x.dw3 | x.dw4) == 0) && (k <= 57) {
		var p u192
		if k <= 38 {
			p = u192{pow10_128_dw0(k), pow10_128_dw1(k), 0}
		} else {
			p = u192{pow10_256_dw0(k - 39), pow10_256_dw1(k - 39), pow10_256_dw2(k - 39)}
		}
		return mul_320_192x192(u192{x.dw0, x.dw1, x.dw2}, p)
	}
	var m [5]uint64
	var mLen int
	if k <= 38 {
		m[0] = uint64(pow10_128_dw0(k))
		m[1] = uint64(pow10_128_dw1(k))
		mLen = 2
		if k <= 19 {
			mLen = 1
		}
	} else if k <= 77 {
		i := k - 39
		m[0] = uint64(pow10_256_dw0(i))
		m[1] = uint64(pow10_256_dw1(i))
		m[2] = uint64(pow10_256_dw2(i))
		m[3] = uint64(pow10_256_dw3(i))
		mLen = 4
		if k <= 57 {
			mLen = 3
		}
	} else {
		i := k - 78
		m[0] = uint64(pow10_320_dw0(i))
		m[1] = uint64(pow10_320_dw1(i))
		m[2] = uint64(pow10_320_dw2(i))
		m[3] = uint64(pow10_320_dw3(i))
		m[4] = uint64(pow10_320_dw4(i))
		mLen = 5
	}
	return mulTrunc_320(x, &m, mLen)
}

// mulTrunc_320 is x · m for an mLen-limb m, the product known to fit 320 bits:
// operand-scanning schoolbook over x's nonzero limbs and m's limbs only.
func mulTrunc_320(x u320, m *[5]uint64, mLen int) u320 {
	a := [5]uint64{uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), uint64(x.dw3), uint64(x.dw4)}
	xLen := limbLen_320(x)
	verify("mulPow10_320 overflow", func() bool { return (xLen == 0) || (xLen+mLen <= 6) })
	var r [5]uint64
	for i := 0; i < xLen; i++ {
		var carry uint64
		for j := 0; (j < mLen) && (i+j < 5); j++ {
			hi, lo := bits.Mul64(a[i], m[j])
			s1, o1 := bits.Add64(lo, r[i+j], 0)
			s2, o2 := bits.Add64(s1, carry, 0)
			r[i+j] = s2
			carry = hi + o1 + o2
		}
		if i+mLen < 5 {
			r[i+mLen] = carry
		} else {
			verify("mulPow10_320 overflow", func() bool { return carry == 0 })
		}
	}
	return u320{int64(r[0]), int64(r[1]), int64(r[2]), int64(r[3]), int64(r[4])}
}

// mul_320_320x64 is x · y for a one-limb y; the product must fit 320 bits.
// (exp's range reduction n · ln10.)
func mul_320_320x64(x u320, y int64) u320 {
	m := [5]uint64{uint64(y)}
	mLen := 1
	if y == 0 {
		mLen = 0
	}
	return mulTrunc_320(x, &m, mLen)
}
