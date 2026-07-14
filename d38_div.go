// D38 division — the 38-digit internal extended-precision domain (round-half-
// even, no flags, no rounding mode), used by the Padé transcendental kernels.
// Mirrors the D128 divide pattern but targets 38 digits.
//
// Capacity note: a 38-digit-precision quotient lands in [10^37, 10^39) — i.e.
// 38 OR 39 digits — so it can exceed U128. We take u256DivRem128's full U256
// quotient and let d38_finalize_256 scale the 39-digit case back down (merging
// residues).

package decimal128

func d38_divide(x, y d38) d38 {
	verify("", func() bool { return !d38_isZero(y) })
	if d38_isZero(x) {
		return D38_ZERO
	}
	zSign := d38_signFlag(x) != d38_signFlag(y)
	digitLenDelta := d38_digitLen(x) - d38_digitLen(y)
	numeratorScale := 38 - digitLenDelta
	verify("", func() bool { return numeratorScale > 0 })

	// Scale x's coefficient up so the quotient carries a full 38–39 digits.
	scaledNumerator := mulPow10_256_128(x.coeffLo64, x.coeffHi64, numeratorScale)
	yU128 := d38_u128(y)
	qr := u256DivRem128(scaledNumerator, yU128)
	quot := u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
	res := residueFromRemainderDivisor_128(qr.remDw0, qr.remDw1, yU128.dw0, yU128.dw1)

	// x/y = quotient × 10^(qExp(x) − qExp(y) − numeratorScale).
	qExp := d38_qExp(x) - d38_qExp(y) - numeratorScale
	return d38_finalize_256(zSign, qExp, quot, res)
}
