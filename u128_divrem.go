// The 128-bit divRem family (PrimitiveLayer.md §4.2).
//
// Swift uses native UInt128 division; Go synthesizes. divRem_q128r64 uses the
// hardware 128÷64 (bits.Div64). divRem_q128r128 follows the JVM strategy named in
// the Swift comment: a sub-2^64 divisor delegates to the 128÷64 kernel; otherwise
// the quotient fits in 64 bits, reached by one normalized DIV3BY2 step
// (Möller–Granlund) instead of native division.

package decimal128

import "math/bits"

// divRem_q128r64_128x64 returns (x / y, x % y) for a 64-bit divisor; y nonzero.
func divRem_q128r64_128x64(xDw0, xDw1, y int64) quot128Rem64 {
	demand("division by zero", y != 0)
	x0 := uint64(xDw0)
	x1 := uint64(xDw1)
	d := uint64(y)
	q1 := x1 / d
	r1 := x1 % d
	q0, r0 := bits.Div64(r1, x0, d) // r1 < d guaranteed
	return quot128Rem64{int64(q0), int64(q1), int64(r0)}
}

// divRem_q128r128_128x128 returns (x / y, x % y); y nonzero (D15).
func divRem_q128r128_128x128(xDw0, xDw1, yDw0, yDw1 int64) quot128Rem128 {
	demand("division by zero", !isZero_bool_128(yDw0, yDw1))

	if yDw1 == 0 {
		// Divisor < 2^64: delegate to the 128÷64 kernel.
		qr := divRem_q128r64_128x64(xDw0, xDw1, yDw0)
		return quot128Rem128{qr.quotDw0, qr.quotDw1, qr.remDw0, 0}
	}
	// Divisor >= 2^64, so the quotient fits in 64 bits.
	if cmp_int_128x128(xDw0, xDw1, yDw0, yDw1) < 0 {
		return quot128Rem128{0, 0, xDw0, xDw1} // x < y: q = 0, r = x
	}

	x0 := uint64(xDw0)
	x1 := uint64(xDw1)
	y0 := uint64(yDw0)
	y1 := uint64(yDw1)
	s := clz_int_64(yDw1) // 0..63

	var xn0, xn1, xn2, dn0, dn1 uint64
	if s == 0 {
		xn0, xn1, xn2 = x0, x1, 0
		dn0, dn1 = y0, y1
	} else {
		rs := uint(64 - s)
		xn0 = x0 << uint(s)
		xn1 = (x1 << uint(s)) | (x0 >> rs)
		xn2 = x1 >> rs
		dn0 = y0 << uint(s)
		dn1 = (y1 << uint(s)) | (y0 >> rs)
	}

	v := vReciprocal_64_128_mg(int64(dn1), int64(dn0))
	res := divRem_q64r128_192x128x64_v_mg(int64(xn2), int64(xn1), int64(xn0), int64(dn1), int64(dn0), v)

	// The normalized remainder is the true remainder shifted left by s.
	remLo := uint64(res.remDw0)
	remHi := uint64(res.remDw1)
	if s != 0 {
		rs := uint(64 - s)
		remLo = (remLo >> uint(s)) | (remHi << rs)
		remHi = remHi >> uint(s)
	}
	return quot128Rem128{res.quotDw0, 0, int64(remLo), int64(remHi)}
}
