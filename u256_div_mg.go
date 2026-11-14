// Möller–Granlund 256/128 division (PrimitiveLayer.md; MG Alg 5, DIV3BY2).

package decimal128

import "math/bits"

// div_256_128_q128r128_mg: x > 2^128, y > 2^64, quotient <= 2^128. One reciprocal
// build + two DIV3BY2 steps.
func div_256_128_q128r128_mg(x u256, y u128) quot128Rem128 {
	verify("", func() bool { return y.dw1 != 0 })
	shift := bits.LeadingZeros64(uint64(y.dw1)) // 0..63
	d := y
	if shift != 0 {
		d = shlU128(y, shift)
	}
	d1 := uint64(d.dw1)
	d0 := uint64(d.dw0)
	v := vReciprocal_64_128_mg(int64(d1), int64(d0))

	u0, u1, u2, u3, u4 := normalizeDividend(x, shift)
	verify("quotient exceeds 128 bits", func() bool { return u4 == 0 })

	remHi, remLo := u3, u2 // top 128 bits; < d (qh == 0)

	s1 := divRem_q64r128_192x128x64_v_mg(int64(remHi), int64(remLo), int64(u1), int64(d1), int64(d0), v)
	qHi := uint64(s1.quotDw0)
	remHi, remLo = uint64(s1.remDw1), uint64(s1.remDw0)

	s2 := divRem_q64r128_192x128x64_v_mg(int64(remHi), int64(remLo), int64(u0), int64(d1), int64(d0), v)
	qLo := uint64(s2.quotDw0)
	remHi, remLo = uint64(s2.remDw1), uint64(s2.remDw0)

	remLo, remHi = denormalizeRemainder(remLo, remHi, shift)
	return quot128Rem128{int64(qLo), int64(qHi), int64(remLo), int64(remHi)}
}

// div_256_128_q256r128_mg: x > 2^128, y > 2^64, full quotient (sqrt).
func div_256_128_q256r128_mg(x u256, y u128) quot256Rem128 {
	verify("", func() bool { return y.dw1 != 0 })
	shift := bits.LeadingZeros64(uint64(y.dw1))
	d := y
	if shift != 0 {
		d = shlU128(y, shift)
	}
	d1 := uint64(d.dw1)
	d0 := uint64(d.dw0)
	v := vReciprocal_64_128_mg(int64(d1), int64(d0))

	u0, u1, u2, u3, u4 := normalizeDividend(x, shift)

	remHi, remLo := u4, u3
	var qh uint64 = 0
	if cmp_int_128x128(int64(remLo), int64(remHi), int64(d0), int64(d1)) >= 0 {
		dd := sub_128_128x128(int64(remLo), int64(remHi), int64(d0), int64(d1))
		remLo, remHi = uint64(dd.dw0), uint64(dd.dw1)
		qh = 1
	}

	s2 := divRem_q64r128_192x128x64_v_mg(int64(remHi), int64(remLo), int64(u2), int64(d1), int64(d0), v)
	q2 := uint64(s2.quotDw0)
	remHi, remLo = uint64(s2.remDw1), uint64(s2.remDw0)
	s1 := divRem_q64r128_192x128x64_v_mg(int64(remHi), int64(remLo), int64(u1), int64(d1), int64(d0), v)
	q1 := uint64(s1.quotDw0)
	remHi, remLo = uint64(s1.remDw1), uint64(s1.remDw0)
	s0 := divRem_q64r128_192x128x64_v_mg(int64(remHi), int64(remLo), int64(u0), int64(d1), int64(d0), v)
	q0 := uint64(s0.quotDw0)
	remHi, remLo = uint64(s0.remDw1), uint64(s0.remDw0)

	remLo, remHi = denormalizeRemainder(remLo, remHi, shift)
	return quot256Rem128{int64(q0), int64(q1), int64(q2), int64(qh), int64(remLo), int64(remHi)}
}
