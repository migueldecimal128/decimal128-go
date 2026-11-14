// Core 256÷128 division with residue (the divide hot path). Delegates to the
// primitive 256÷64 / 128÷128 kernels where the operands fit, and to the
// Möller–Granlund reciprocal divide for the wide case (x > 2^128, y > 2^64).
// Two MG wrappers: a branch-light 2-step form for the ≤128-bit quotient the D128
// divide contract guarantees, and a schoolbook loop for the full U256 quotient
// (sqrt). 128-bit rem intermediates are synthesized from (remHi, remLo).

package decimal128

func u256DivRem128(x u256, y u128) quot256Rem128 {
	if y.dw1 == 0 { // y < 2^64
		qr := divRem_q256r64_256x64(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0)
		return quot256Rem128{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3, qr.remDw0, 0}
	}
	if isZero_bool_128(x.dw2, x.dw3) { // x < 2^128
		if cmp_int_128x128(x.dw0, x.dw1, y.dw0, y.dw1) < 0 {
			return quot256Rem128{0, 0, 0, 0, x.dw0, x.dw1}
		}
		qr := divRem_q128r128_128x128(x.dw0, x.dw1, y.dw0, y.dw1)
		return quot256Rem128{qr.quotDw0, qr.quotDw1, 0, 0, qr.remDw0, qr.remDw1}
	}
	return div_256_128_q256r128_mg(x, y)
}

func u256DivRem_128(x u256, y u128) quot128Rem128 {
	// The D128 divide path (quotient <= 128 bits by contract) routes a one-word
	// divisor through the 192/64 primitive (D54) and its class-N case
	// (x > 2^128, y > 2^64) through the branch-light 2-step M&G divide;
	// sqrt reaches the wide-quotient loop via u256DivRem128 directly.
	if y.dw1 == 0 {
		// The 128-bit quotient bound means x.dw3 == 0 and x.dw2 <u y, so two
		// chained 128/64 steps suffice.
		verify("quotient exceeds 128 bits", func() bool { return x.dw3 == 0 && uint64(x.dw2) < uint64(y.dw0) })
		qr := divRem_q128r64_192x64(x.dw0, x.dw1, x.dw2, y.dw0)
		return quot128Rem128{qr.quotDw0, qr.quotDw1, qr.remDw0, 0}
	}
	if !isZero_bool_128(x.dw2, x.dw3) {
		return div_256_128_q128r128_mg(x, y)
	}
	qr := u256DivRem128(x, y)
	verify("", func() bool { return (qr.quotDw2 | qr.quotDw3) == 0 })
	return quot128Rem128{qr.quotDw0, qr.quotDw1, qr.remDw0, qr.remDw1}
}

func u256DivResidue_128(x u256, y u128) quot128Residue {
	qr := u256DivRem_128(x, y)
	res := residueFromRemainderDivisor_128(qr.remDw0, qr.remDw1, y.dw0, y.dw1)
	return quot128Residue{qr.quotDw0, qr.quotDw1, res}
}

// normalizeDividend shifts x left by shift (0..63) into 5 limbs u0..u4.
func normalizeDividend(x u256, shift int) (u0, u1, u2, u3, u4 uint64) {
	x0, x1 := uint64(x.dw0), uint64(x.dw1)
	x2, x3 := uint64(x.dw2), uint64(x.dw3)
	if shift == 0 {
		return x0, x1, x2, x3, 0
	}
	rs := uint(64 - shift)
	u0 = x0 << uint(shift)
	u1 = (x1 << uint(shift)) | (x0 >> rs)
	u2 = (x2 << uint(shift)) | (x1 >> rs)
	u3 = (x3 << uint(shift)) | (x2 >> rs)
	u4 = x3 >> rs
	return
}

// denormalizeRemainder undoes the normalization shift on the final remainder.
func denormalizeRemainder(remLo, remHi uint64, shift int) (uint64, uint64) {
	if shift == 0 {
		return remLo, remHi
	}
	r := shrU128(u128{int64(remLo), int64(remHi)}, shift)
	return uint64(r.dw0), uint64(r.dw1)
}
