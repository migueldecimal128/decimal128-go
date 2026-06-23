// The mulPow10 family (PrimitiveLayer.md D4): table-driven over the pow10 tables.
// The int exponent operand is implied by the op name.

package decimal128

import "math/bits"

// mulPow10_128_128 returns the low 128 bits of x * 10^pow10 (wrapping).
func mulPow10_128_128(xDw0, xDw1 int64, pow10 int) u128 {
	return mul_128_128x128(xDw0, xDw1, pow10_128_dw0(pow10), pow10_128_dw1(pow10))
}

// mulPow10_256_128 returns x * 10^pow10 as a u256, trapping where it must fit.
func mulPow10_256_128(xDw0, xDw1 int64, pow10 int) u256 {
	if pow10 <= 38 {
		return mul_256_128x128(xDw0, xDw1, pow10_128_dw0(pow10), pow10_128_dw1(pow10))
	}
	demand("mulPow10: pow10 exceeds maximum of 77", pow10 <= 77)
	b := pow10 - 39
	return mul_256_256x128(pow10_256_dw0(b), pow10_256_dw1(b), pow10_256_dw2(b), pow10_256_dw3(b), xDw0, xDw1)
}

// mulPow10_256_256 returns x * 10^pow10, trapping where it must fit.
func mulPow10_256_256(xDw0, xDw1, xDw2, xDw3 int64, pow10 int) u256 {
	if pow10 <= 0 {
		demand("", pow10 >= 0)
		return u256{xDw0, xDw1, xDw2, xDw3}
	}
	hiZero := xDw2 == 0 && xDw3 == 0
	if pow10 <= 19 {
		p64 := uint64(pow10_128_dw0(pow10))
		if hiZero {
			return mul_256_128x64(u128{xDw0, xDw1}, p64)
		}
		return mul_256_256x64(xDw0, xDw1, xDw2, xDw3, int64(p64))
	}
	if pow10 <= 38 {
		if hiZero {
			return mul_256_128x128(xDw0, xDw1, pow10_128_dw0(pow10), pow10_128_dw1(pow10))
		}
		return mul_256_256x128(xDw0, xDw1, xDw2, xDw3, pow10_128_dw0(pow10), pow10_128_dw1(pow10))
	}
	demand("mulPow10: pow10 exceeds maximum of 77", pow10 <= 77)
	b := pow10 - 39
	pd0, pd1 := pow10_256_dw0(b), pow10_256_dw1(b)
	pd2, pd3 := pow10_256_dw2(b), pow10_256_dw3(b)
	if hiZero {
		if xDw1 == 0 {
			return mul_256_256x64(pd0, pd1, pd2, pd3, xDw0)
		}
		return mul_256_256x128(pd0, pd1, pd2, pd3, xDw0, xDw1)
	}
	return mul_256_256x256(xDw0, xDw1, xDw2, xDw3, pd0, pd1, pd2, pd3)
}

// fusedMulPow10Add_256_256x64 returns x * 10^pow10 + addend, interleaving the
// limb products with the addend's carry chain (§3.4; sole consumer is the
// digit-accumulation path). Preconditions: pow10 in [0, 19], result fits 77
// digits.
func fusedMulPow10Add_256_256x64(x u256, pow10 int, addend uint64) u256 {
	demand("fusedMulPow10Add: pow10 outside [0, 19]", pow10 >= 0 && pow10 <= 19)
	demand("overflows 77 digits", x.digitLen()+pow10 <= 77)
	y := uint64(pow10_128_dw0(pow10)) // 10^19 < 2^64

	a0 := uint64(x.dw0)
	a1 := uint64(x.dw1)
	a2 := uint64(x.dw2)
	a3 := uint64(x.dw3)

	h0, l0 := bits.Mul64(a0, y)
	h1, l1 := bits.Mul64(a1, y)
	h2, l2 := bits.Mul64(a2, y)
	h3, l3 := bits.Mul64(a3, y)
	verify("", h3 == 0) // no high limb by the no-overflow precondition

	r0, c0 := bits.Add64(l0, addend, 0)

	r1t, c1a := bits.Add64(h0, l1, 0)
	r1, c1b := bits.Add64(r1t, c0, 0)
	c1 := c1a + c1b

	r2t, c2a := bits.Add64(h1, l2, 0)
	r2, c2b := bits.Add64(r2t, c1, 0)
	c2 := c2a + c2b

	r3t, c3a := bits.Add64(h2, l3, 0)
	r3, c3b := bits.Add64(r3t, c2, 0)
	verify("", c3a == 0 && c3b == 0)

	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}
