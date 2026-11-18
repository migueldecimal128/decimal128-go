// IEEE 754-2019 §5.3.1 nextUp / nextDown.
//
// Returns the immediately-larger (nextUp) or immediately-smaller (nextDown)
// representable value. The two are duals: nextDown(x) == -nextUp(-x).
//
// For finite values, the operation walks one ulp in the chosen direction. To
// handle decade boundaries cleanly (so that nextUp(0.99999...) → 1.00000... in
// the natural cohort), the coefficient is first promoted to full precision (34
// digits) by absorbing headroom from the representation, then the ±1 step is
// applied. Carry across a decade boundary increments the exponent; underflow
// below the smallest subnormal collapses to zero or saturates by direction.

package decimal128

// The largest finite magnitude, (10^34 - 1) * 10^6111, with the given sign
// (also finalize's overflow saturation value).
func maxFiniteMagnitude(sign bool) d128 {
	c := sub_128_128x128(pow10_128_dw0(34), pow10_128_dw1(34), 1, 0)
	return decimalFNZ_128(sign, 6111, c.dw0, c.dw1)
}

// The smallest finite (subnormal) magnitude, 1 * 10^-6176, with the given
// sign.
func minFiniteMagnitude(sign bool) d128 {
	return decimalFNZ_64x64(sign, -6176, 1, 0)
}

// d128_nextUp_ctxnull returns the smallest representable value greater than `x`.
// IEEE 754-2019 Section 5.3.1 `nextUp`.
//
//   - For finite values, walks one ulp toward +inf.
//   - `+inf` returns `+inf`; `-inf` returns the most-negative finite.
//   - `+0` and `-0` return the smallest positive subnormal.
//   - A quiet NaN is returned unchanged.
//   - A signaling NaN is quieted; under a context, `invalidOperation`
//     is signaled. Under Quiet (`ctx == nil`), no flag is raised.
func d128_nextUp_ctxnull(x d128, ctxnull *context) d128 {
	return nextUpOrDown_ctxnull(x, true, ctxnull)
}

// d128_nextDown_ctxnull returns the largest representable value smaller than `x`.
// IEEE 754-2019 Section 5.3.1 `nextDown`.
//
//   - For finite values, walks one ulp toward -inf.
//   - `-inf` returns `-inf`; `+inf` returns the most-positive finite.
//   - `+0` and `-0` return the smallest negative subnormal magnitude.
//   - A quiet NaN is returned unchanged.
//   - A signaling NaN is quieted; under a context, `invalidOperation`
//     is signaled. Under Quiet (`ctx == nil`), no flag is raised.
func d128_nextDown_ctxnull(x d128, ctxnull *context) d128 {
	return nextUpOrDown_ctxnull(x, false, ctxnull)
}

func nextUpOrDown_ctxnull(x d128, isUp bool, ctxnull *context) d128 {
	signX := d128_signFlag(x)
	if d128_isFIN(x) {
		if d128_isZER(x) {
			// From signed zero, the next step is the smallest finite whose sign
			// is opposite to the direction:
			//   nextUp(±0)  → +minFinite
			//   nextDown(±0) → -minFinite
			return minFiniteMagnitude(!isUp)
		}
		// FNZ: walking "toward zero" when the sign matches the direction
		// (positive nextDown, negative nextUp), else "away from zero".
		if signX == isUp {
			return nextFnzTowardZero(x)
		}
		return nextFnzAwayFromZero(x)
	}
	if d128_isINF(x) {
		// signX==true, isUp==true:  -∞ nextUp   → maxFinite(-).
		// signX==false, isUp==false: +∞ nextDown → maxFinite(+).
		if signX == isUp {
			return maxFiniteMagnitude(signX)
		}
		return x
	}
	// NaN: quiet returned as-is; signaling quieted (and signaled under a ctx).
	verify("", func() bool { return d128_isNAN(x) })
	if d128_isQNAN(x) {
		return x
	}
	if ctxnull != nil {
		return ctxSignalSNanOperandFound_ctxnull(ctxnull, x)
	}
	return signalSNanOperandFound_quiet(x)
}

// Walk one ulp away from zero (increase magnitude).
func nextFnzAwayFromZero(x d128) d128 {
	verify("", func() bool { return d128_isFNZ(x) })
	signX := d128_signFlag(x)
	qExp := d128_qExp(x)
	coeff := d128_u128(x)

	// Promote to full precision so the ±1 step lands at the smallest ulp the
	// value's exponent permits.
	headroom := min(34-d128_digitLen(x), qExp-(-6176))
	if headroom > 0 {
		coeff = mulPow10_128_128(coeff.dw0, coeff.dw1, headroom)
		qExp -= headroom
	}

	coeff = add_128_128x128(coeff.dw0, coeff.dw1, 1, 0)

	// Carry across the decade boundary: 10^34 → 10^33 with qExp+1.
	if cmp_int_128x128_pow10(coeff.dw0, coeff.dw1, 34) == 0 {
		coeff = fromLoHi_128_64x64(pow10_128_dw0(33), pow10_128_dw1(33))
		qExp += 1
		if qExp > 6111 {
			return d128_infinity(signX)
		}
	}
	return decimalFNZ_128(signX, qExp, coeff.dw0, coeff.dw1)
}

// Walk one ulp toward zero (decrease magnitude).
func nextFnzTowardZero(x d128) d128 {
	verify("", func() bool { return d128_isFNZ(x) })
	signX := d128_signFlag(x)
	qExp := d128_qExp(x)
	coeff := d128_u128(x)

	// Promote to full precision when room exists. This makes the decade-boundary
	// check below well-defined: full precision means the coefficient is in
	// [10^33, 10^34), so a decrement may cross 10^33.
	headroom := min(34-d128_digitLen(x), qExp-(-6176))
	if headroom > 0 {
		coeff = mulPow10_128_128(coeff.dw0, coeff.dw1, headroom)
		qExp -= headroom
	}

	// If the coefficient is exactly 10^33 and there is exponent room below, drop
	// a decade so we get the full 34-digit representation at the smaller exponent
	// before decrementing.
	if cmp_int_128x128_pow10(coeff.dw0, coeff.dw1, 33) == 0 && qExp > -6176 {
		coeff = fromLoHi_128_64x64(pow10_128_dw0(34), pow10_128_dw1(34))
		qExp -= 1
	}
	coeff = sub_128_128x128(coeff.dw0, coeff.dw1, 1, 0)
	// Stepping from the smallest subnormal lands on zero.
	if isZero_bool_128(coeff.dw0, coeff.dw1) {
		return d128_zero(signX, qExp)
	}
	return decimalFNZ_128(signX, qExp, coeff.dw0, coeff.dw1)
}
