// IEEE 754-2019 §5.3.3 stripTrailingZeros: remove trailing decimal zeros from a
// finite coefficient, raising qExp correspondingly. The numeric value is
// unchanged; only the cohort representation differs.

package decimal128

func d128_stripTrailingZeros(x d128, maxToStrip int) d128 {
	return stripTrailingZerosImpl(x, maxToStrip, nil)
}

func d128_stripTrailingZeros_ctx(x d128, maxToStrip int, ctx *context) d128 {
	return stripTrailingZerosImpl(x, maxToStrip, ctx)
}

func stripTrailingZerosImpl(x d128, maxToStrip int, ctxnull *context) d128 {
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	if d128_isINF(x) {
		return x
	}
	if d128_isZER(x) {
		return d128_zero(d128_signFlag(x), 0)
	}
	if maxToStrip <= 0 {
		return x
	}
	xQ := d128_qExp(x)
	effectiveMax := min(maxToStrip, 6111-xQ)
	if effectiveMax <= 0 {
		return x
	}
	ss := strip_s128sc_128(x.ubdLo64, d128_hi49(x), effectiveMax)
	if ss.stripCount == 0 {
		return x
	}
	return decimalFNZ_64x64(d128_signFlag(x), xQ+int(ss.stripCount), ss.strippedDw0, ss.strippedDw1)
}
