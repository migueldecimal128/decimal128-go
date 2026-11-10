// 320 ÷ 160 division for the D48 divide and isqrt (D48GoWorkOrder.md Phase 1;
// Swift U320DivMG.swift). Schoolbook over 64-bit limbs in the shape of GMP's
// mpn_sbpi1_div_qr: normalize, then per quotient limb one Möller–Granlund
// DIV3BY2 estimate on the top three remainder limbs (reciprocal_div_mg.go), a
// multiply-subtract of the divisor's remaining low limb, and at most one
// add-back. A one-limb divisor takes the DIV2BY1 loop instead. When the top two
// remainder limbs equal the divisor's, DIV3BY2's precondition fails and the
// quotient limb is 2^64 - 1.

package decimal128

import "math/bits"

// divRem_320_160_mg is x / y and x mod y for a nonzero y < 2^160 (as a u192)
// and any x < 2^320.
func divRem_320_160_mg(x u320, y u192) quot320Rem160 {
	y3 := u320FromU192(y)
	demand("division by zero", !y.isZero())
	verify("divisor wider than 160 bits", func() bool { return bitLen_int_320(y3) <= 160 })
	yBits := bitLen_int_320(y3)
	yLimbs := (yBits + 63) >> 6      // 1..3
	shift := uint(64*yLimbs - yBits) // leading zeros of the top limb
	ya := [3]uint64{uint64(y.dw0), uint64(y.dw1), uint64(y.dw2)}
	xa := [5]uint64{uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), uint64(x.dw3), uint64(x.dw4)}

	// Normalized divisor d[0..2] (top bit of d[yLimbs-1] set) and dividend
	// u[0..5]; u[6] = 0 guards the first step.
	var d [3]uint64
	var u [7]uint64
	if shift == 0 {
		d = ya
		copy(u[:5], xa[:])
	} else {
		rs := 64 - shift
		d[0] = ya[0] << shift
		d[1] = (ya[1] << shift) | (ya[0] >> rs)
		d[2] = (ya[2] << shift) | (ya[1] >> rs)
		u[0] = xa[0] << shift
		for i := 1; i < 5; i++ {
			u[i] = (xa[i] << shift) | (xa[i-1] >> rs)
		}
		u[5] = xa[4] >> rs
	}
	var q [6]uint64

	if yLimbs == 1 {
		dd := d[0]
		v := vReciprocal_64_64_mg(int64(dd))
		var r uint64
		for i := 5; i >= 0; i-- {
			s := divRem_q64r64_128x64x64_v_mg(int64(r), int64(u[i]), int64(dd), v)
			q[i] = uint64(s.quotDw0)
			r = uint64(s.remDw0)
		}
		qq := assemble_320(&q)
		return quot320Rem160{qq.dw0, qq.dw1, qq.dw2, qq.dw3, qq.dw4, int64(r >> shift), 0, 0}
	}

	n := yLimbs // 2 or 3
	d1 := d[n-1]
	d0 := d[n-2]
	v := vReciprocal_64_128_mg(int64(d1), int64(d0))

	for j := 6 - n; j >= 0; j-- { // quotient limbs q[0 .. 6-n]
		n2 := u[j+n]
		n1 := u[j+n-1]
		var qj uint64
		if (n2 == d1) && (n1 == d0) {
			// DIV3BY2's precondition fails; the quotient limb is 2^64 - 1.
			qj = ^uint64(0)
			var cy uint64
			for i := 0; i < n; i++ {
				hi, lo := bits.Mul64(qj, d[i])
				lo, o := bits.Add64(lo, cy, 0)
				cy = hi + o
				if u[j+i] < lo {
					cy++
				}
				u[j+i] -= lo
			}
			u[j+n] -= cy
			verify("", func() bool { return u[j+n] == 0 })
		} else {
			s := divRem_q64r128_192x128x64_v_mg(int64(n2), int64(n1), int64(u[j+n-2]), int64(d1), int64(d0), v)
			qj = uint64(s.quotDw0)
			r0 := uint64(s.remDw0)
			r1 := uint64(s.remDw1)
			// Subtract qj · d[0 .. n-3] (one limb when n == 3) below the estimate.
			var cy uint64
			for i := 0; i < n-2; i++ {
				hi, lo := bits.Mul64(qj, d[i])
				lo, o := bits.Add64(lo, cy, 0)
				cy = hi + o
				if u[j+i] < lo {
					cy++
				}
				u[j+i] -= lo
			}
			var b0, b1 uint64
			r0, b0 = bits.Sub64(r0, cy, 0)
			r1, b1 = bits.Sub64(r1, b0, 0)
			u[j+n-2] = r0
			if b1 != 0 {
				// qj was one too large: add the divisor back.
				var c uint64
				for i := 0; i < n-1; i++ {
					u[j+i], c = bits.Add64(u[j+i], d[i], c)
				}
				r1 = r1 + d1 + c
				qj--
			}
			u[j+n-1] = r1
			u[j+n] = 0
		}
		q[j] = qj
	}

	// Remainder: u[0 .. n-1] >> shift.
	var rem [3]uint64
	for i := 0; i < n; i++ {
		w := u[i] >> shift
		if (shift != 0) && (i+1 < n) {
			w |= u[i+1] << (64 - shift)
		}
		rem[i] = w
	}
	qq := assemble_320(&q)
	return quot320Rem160{qq.dw0, qq.dw1, qq.dw2, qq.dw3, qq.dw4, int64(rem[0]), int64(rem[1]), int64(rem[2])}
}

func assemble_320(q *[6]uint64) u320 {
	verify("quotient wider than 320 bits", func() bool { return q[5] == 0 })
	return u320{int64(q[0]), int64(q[1]), int64(q[2]), int64(q[3]), int64(q[4])}
}
