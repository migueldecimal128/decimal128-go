// D48 divide (round-half-even; Swift D48Div.swift). Mirrors d38_divide: the
// numerator is scaled so the quotient lands in [10^47, 10^49) -- 48 or 49
// digits -- and the full u320 quotient goes to d48_finalize_320, which scales
// the 49-digit case down. So the result always has exactly 48 digits, exact
// quotients included. The division is the 320÷160 MG kernel.

package decimal128

func d48_divide(x, y d48) d48 {
	verify("", func() bool { return !d48_isZero(y) })
	if d48_isZero(x) {
		return d48_ZERO
	}
	zSign := d48_signFlag(x) != d48_signFlag(y)
	digitLenDelta := d48_digitLen(x) - d48_digitLen(y)
	numeratorScale := 48 - digitLenDelta
	verify("", func() bool { return numeratorScale > 0 })

	// At most 48 + digitLen(y) <= 96 digits: fits u320.
	scaledNumerator := mulPow10_320(d48_u320(x), numeratorScale)
	qr := divRem_320_160_mg(scaledNumerator, d48_u160(y))
	res := residueFromRemainderDivisor_320(u320FromU192(qr.rem()), d48_u320(y))

	// x/y = quotient × 10^(qExp(x) − qExp(y) − numeratorScale).
	qExp := d48_qExp(x) - d48_qExp(y) - numeratorScale
	return d48_finalize_320(zSign, qExp, qr.quot(), res)
}
