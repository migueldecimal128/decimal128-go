// The 256÷64 simple-division unit (PrimitiveLayer.md D3, §4.4): full 256÷64
// quotient and 64-bit remainder. The dispatcher specializes by divisor shape —
// power-of-two (shift+mask), 16- and 32-bit divisors (hardware divides on 48/32-
// bit chunks), and 33–64-bit divisors (one Möller–Granlund reciprocal build plus
// four DIV2BY1 steps). Ported from Swift; uint64 limb bodies.

package decimal128

import "math/bits"

func divRem_q256r64_256x64(xDw0, xDw1, xDw2, xDw3, y0 int64) quot256Rem64 {
	d := uint64(y0)
	x := u256{xDw0, xDw1, xDw2, xDw3}

	if d <= 1 {
		if d == 0 {
			impossible("division by zero")
		}
		return quot256Rem64{x.dw0, x.dw1, x.dw2, x.dw3, 0} // d == 1
	}

	dw0 := uint64(x.dw0)
	if x.bitLen() <= 64 {
		return quot256Rem64{int64(dw0 / d), 0, 0, 0, int64(dw0 % d)}
	}

	// Power-of-two divisor: shift + mask.
	if d&(d-1) == 0 {
		ntz := bits.TrailingZeros64(d) // >= 1 since d >= 2
		mask := (uint64(1) << uint(ntz)) - 1
		r := dw0 & mask
		s := shrU256(x, ntz)
		return quot256Rem64{s.dw0, s.dw1, s.dw2, s.dw3, int64(r)}
	}

	// Specialize by divisor width.
	if d>>32 == 0 {
		if d>>16 == 0 {
			return divRem16(x, d)
		}
		return divRem32(x, d)
	}
	// 33–64-bit divisor.
	return divRem256by64(x.dw0, x.dw1, x.dw2, x.dw3, y0)
}

// divRem256by64 is the Möller–Granlund Alg. 4 path: one reciprocal build plus
// four DIV2BY1 steps. Precondition: 33–64-bit divisor (y >> 32 != 0), so the
// normalization shift <= 31 and the top normalized limb u4 < dn.
func divRem256by64(xDw0, xDw1, xDw2, xDw3, y int64) quot256Rem64 {
	x0, x1 := uint64(xDw0), uint64(xDw1)
	x2, x3 := uint64(xDw2), uint64(xDw3)
	d0 := uint64(y)
	verify("", (d0>>32) != 0)

	shift := bits.LeadingZeros64(d0) // 0..31
	dn := d0
	if shift != 0 {
		dn = d0 << uint(shift)
	}
	v := vReciprocal_64_64_mg(int64(dn))

	var u0, u1, u2, u3, u4 uint64
	if shift == 0 {
		u0, u1, u2, u3, u4 = x0, x1, x2, x3, 0
	} else {
		rs := uint(64 - shift)
		u0 = x0 << uint(shift)
		u1 = (x1 << uint(shift)) | (x0 >> rs)
		u2 = (x2 << uint(shift)) | (x1 >> rs)
		u3 = (x3 << uint(shift)) | (x2 >> rs)
		u4 = x3 >> rs
	}

	r := u4
	s3 := divRem_q64r64_128x64x64_v_mg(int64(r), int64(u3), int64(dn), v)
	q3 := uint64(s3.quotDw0)
	r = uint64(s3.remDw0)
	s2 := divRem_q64r64_128x64x64_v_mg(int64(r), int64(u2), int64(dn), v)
	q2 := uint64(s2.quotDw0)
	r = uint64(s2.remDw0)
	s1 := divRem_q64r64_128x64x64_v_mg(int64(r), int64(u1), int64(dn), v)
	q1 := uint64(s1.quotDw0)
	r = uint64(s1.remDw0)
	s0 := divRem_q64r64_128x64x64_v_mg(int64(r), int64(u0), int64(dn), v)
	q0 := uint64(s0.quotDw0)
	r = uint64(s0.remDw0)

	remainder := r
	if shift != 0 {
		remainder = r >> uint(shift)
	}
	return quot256Rem64{int64(q0), int64(q1), int64(q2), int64(q3), int64(remainder)}
}

func divRem32(x u256, y0 uint64) quot256Rem64 {
	if y0>>16 == 0 {
		return divRem16(x, y0)
	}
	verify("", y0>>32 == 0)
	bitLen := x.bitLen()
	if bitLen <= 64 {
		return divMod64x32(uint64(x.dw0), y0)
	}
	if bitLen <= 128 {
		return divMod128x32(uint64(x.dw1), uint64(x.dw0), y0)
	}
	if bitLen <= 192 {
		return divMod192x32(uint64(x.dw2), uint64(x.dw1), uint64(x.dw0), y0)
	}
	return divMod256x32(uint64(x.dw3), uint64(x.dw2), uint64(x.dw1), uint64(x.dw0), y0)
}

func divRem16(x u256, y0 uint64) quot256Rem64 {
	demand("", y0 > 1 && y0>>16 == 0)
	bitLen := x.bitLen()
	if bitLen <= 64 {
		return divMod64x32(uint64(x.dw0), y0) // intentionally x32
	}
	if bitLen <= 128 {
		return divMod128x16(uint64(x.dw1), uint64(x.dw0), y0)
	}
	if bitLen <= 192 {
		return divMod192x16(uint64(x.dw2), uint64(x.dw1), uint64(x.dw0), y0)
	}
	return divMod256x16(uint64(x.dw3), uint64(x.dw2), uint64(x.dw1), uint64(x.dw0), y0)
}

// --- x32 kernels (32-bit chunking) ---

func divMod64x32(x0, y0 uint64) quot256Rem64 {
	verify("", y0>>32 == 0)
	q0 := x0 / y0
	r := x0 - q0*y0
	return quot256Rem64{int64(q0), 0, 0, 0, int64(r)}
}

func divMod128x32(x1, x0, y0 uint64) quot256Rem64 {
	verify("", y0>>32 == 0)
	q1 := x1 / y0
	rem := x1 - q1*y0
	d1 := (rem << 32) | (x0 >> 32)
	w1 := d1 / y0
	rem = d1 - w1*y0
	d0 := (rem << 32) | (x0 & mask32L)
	w0 := d0 / y0
	rem = d0 - w0*y0
	q0 := (w1 << 32) | w0
	return quot256Rem64{int64(q0), int64(q1), 0, 0, int64(rem)}
}

func divMod192x32(x2, x1, x0, y0 uint64) quot256Rem64 {
	verify("", y0>>32 == 0)
	q2 := x2 / y0
	rem := x2 - q2*y0
	d3 := (rem << 32) | (x1 >> 32)
	w3 := d3 / y0
	rem = d3 - w3*y0
	d2 := (rem << 32) | (x1 & mask32L)
	w2 := d2 / y0
	rem = d2 - w2*y0
	d1 := (rem << 32) | (x0 >> 32)
	w1 := d1 / y0
	rem = d1 - w1*y0
	d0 := (rem << 32) | (x0 & mask32L)
	w0 := d0 / y0
	rem = d0 - w0*y0
	q1 := (w3 << 32) | w2
	q0 := (w1 << 32) | w0
	return quot256Rem64{int64(q0), int64(q1), int64(q2), 0, int64(rem)}
}

func divMod256x32(x3, x2, x1, x0, y0 uint64) quot256Rem64 {
	verify("", y0>>32 == 0)
	q3 := x3 / y0
	rem := x3 - q3*y0
	d5 := (rem << 32) | (x2 >> 32)
	w5 := d5 / y0
	rem = d5 - w5*y0
	d4 := (rem << 32) | (x2 & mask32L)
	w4 := d4 / y0
	rem = d4 - w4*y0
	d3 := (rem << 32) | (x1 >> 32)
	w3 := d3 / y0
	rem = d3 - w3*y0
	d2 := (rem << 32) | (x1 & mask32L)
	w2 := d2 / y0
	rem = d2 - w2*y0
	d1 := (rem << 32) | (x0 >> 32)
	w1 := d1 / y0
	rem = d1 - w1*y0
	d0 := (rem << 32) | (x0 & mask32L)
	w0 := d0 / y0
	rem = d0 - w0*y0
	q2 := (w5 << 32) | w4
	q1 := (w3 << 32) | w2
	q0 := (w1 << 32) | w0
	return quot256Rem64{int64(q0), int64(q1), int64(q2), int64(q3), int64(rem)}
}

// --- x16 kernels (48-bit chunking) ---

func divMod128x16(x1, x0, y0 uint64) quot256Rem64 {
	verify("", y0>>16 == 0)
	q1 := x1 / y0
	rem := x1 - q1*y0
	d1 := (rem << 32) | (x0 >> 32)
	w1 := d1 / y0
	rem = d1 - w1*y0
	d0 := (rem << 32) | (x0 & mask32L)
	w0 := d0 / y0
	rem = d0 - w0*y0
	q0 := (w1 << 32) | w0
	return quot256Rem64{int64(q0), int64(q1), 0, 0, int64(rem)}
}

func divMod192x16(x2, x1, x0, y0 uint64) quot256Rem64 {
	verify("", y0>>16 == 0)
	q2 := x2 / y0
	rem := x2 - q2*y0
	d2 := (rem << 48) | (x1 >> 16)
	w2 := d2 / y0
	rem = d2 - w2*y0
	d1 := (rem << 48) | ((x1 & mask16L) << 32) | (x0 >> 32)
	w1 := d1 / y0
	rem = d1 - w1*y0
	d0 := (rem << 32) | (x0 & mask32L)
	w0 := d0 / y0
	rem = d0 - w0*y0
	q1 := (w2 << 16) | (w1 >> 32)
	q0 := (w1 << 32) | w0
	return quot256Rem64{int64(q0), int64(q1), int64(q2), 0, int64(rem)}
}

func divMod256x16(x3, x2, x1, x0, y0 uint64) quot256Rem64 {
	verify("", y0>>16 == 0)
	q3 := x3 / y0
	rem := x3 - q3*y0
	d3 := (rem << 48) | (x2 >> 16)
	w3 := d3 / y0
	rem = d3 - w3*y0
	d2 := (rem << 48) | ((x2 & mask16L) << 32) | (x1 >> 32)
	w2 := d2 / y0
	rem = d2 - w2*y0
	d1 := (rem << 48) | ((x1 & mask32L) << 16) | (x0 >> 48)
	w1 := d1 / y0
	rem = d1 - w1*y0
	d0 := (rem << 48) | (x0 & mask48L)
	w0 := d0 / y0
	rem = d0 - w0*y0
	q2 := (w3 << 16) | (w2 >> 32)
	q1 := (w2 << 32) | (w1 >> 16)
	q0 := (w1 << 48) | w0
	return quot256Rem64{int64(q0), int64(q1), int64(q2), int64(q3), int64(rem)}
}
