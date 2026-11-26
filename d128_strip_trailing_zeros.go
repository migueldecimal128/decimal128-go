// stripTrailingZeros, an extended operation: removes up to maxToStrip trailing decimal
// zeros from a finite value's coefficient and raises its exponent by the same count,
// capped by the upward exponent room. The numeric value is unchanged; only the cohort
// representation differs. sqrt uses it for its GDAS ideal-exponent reduction.

package decimal128

// d128_stripTrailingZeros returns this value with up to `maxToStrip` trailing decimal zeros
// removed from the coefficient, increasing `qExp` by the same count.
// The numeric value is unchanged.
//
// Non-finite values: infinities are returned unchanged; NaNs propagate
// (with sNaN quieted -- no flag raised under Quiet).
// Zeros collapse to the canonical zero with `qExp = 0`.
//
//   - Parameter maxToStrip: Upper bound on zeros to remove. Defaults
//     to 99 which effectively means "strip all available".
func d128_stripTrailingZeros(x d128, maxToStrip int) d128 {
	return stripTrailingZerosImpl(x, maxToStrip, nil)
}

func d128_stripTrailingZeros_ctx(x d128, maxToStrip int, ctx *context) d128 {
	return stripTrailingZerosImpl(x, maxToStrip, ctx)
}

// Shared implementation. `ctx == nil` for Quiet, non-nil for ctx-aware.
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
