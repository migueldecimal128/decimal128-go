// D48 decision helpers (Swift D48Decide.swift): the rounding-boundary test that
// gates the exact rules, the exact-result emitter, the tiny-argument rule, and
// the Inexact signal. There is no slow path: an ambiguous D48 result that no
// exact rule decides is rounded as is -- the documented 1-ulp gap
// (CrossPlatformArchitecture.md, "Transcendental accuracy").

package decimal128

// d48_ambiguous is true if a value R = coeff·10^qExp, known only to within ±e
// units of its last digit, could round differently into decimal128 than R
// itself: the digits that rounding drops (to 34 digits, or further in the
// subnormal range) lie within e of 0, of half a unit, or of a whole unit.
// e == 0 means R is exact.
//
// The unit 10^drop is never built as a value (Swift D-4): each comparison
// against it is cmp_int_320_pow10. tail ≥ unit − e ⇔ tail + e ≥ 10^drop, and
// (the unit is even) |tail − unit/2| ≤ e ⇔ 2·tail − 2e ≤ 10^drop ≤ 2·tail + 2e.
func d48_ambiguous(coeff u320, qExp int, e u320) bool {
	if coeff.isZero() || e.isZero() {
		return false
	}
	length := digitLen_320(coeff)
	drop := length - 34
	if qExp+drop < -6176 {
		drop = -6176 - qExp
	}
	if drop <= 0 {
		return true // R is representable, the true value is not
	}
	if drop > 96 {
		return cmp_int_320x320(coeff, e) <= 0 // far below half a unit unless ~0
	}
	q := divPow10_q320res_320_recipmul(coeff, drop).quot()
	tail := sub_320_320x320(coeff, mulPow10_320(q, drop))
	if cmp_int_320x320(tail, e) <= 0 {
		return true
	}
	if cmp_int_320_pow10(add_320_320x320(tail, e), drop) >= 0 {
		return true
	}
	twoTail := add_320_320x320(tail, tail)
	twoE := add_320_320x320(e, e)
	if cmp_int_320_pow10(add_320_320x320(twoTail, twoE), drop) < 0 {
		return false
	}
	if cmp_int_320x320(twoTail, twoE) <= 0 {
		return true
	}
	return cmp_int_320_pow10(sub_320_320x320(twoTail, twoE), drop) <= 0
}

// d48_exactWithPreferred emits an exact result c·10^q (c with trailing zeros
// stripped) at the exponent closest to preferred: pad zeros toward it, at most
// to 34 digits and not below Etiny (IEEE 754-2019 §5.2 / the D128 ops'
// preferred-exponent rule).
func d48_exactWithPreferred(sign bool, cIn u128, qIn int, preferred int, rounding round, ctxnull *context) d128 {
	c := cIn
	q := qIn
	if preferred < q {
		length := digitLen_int_128(c.dw0, c.dw1)
		pad := min(q-preferred, 34-length)
		if q > -6176 {
			pad = min(pad, q+6176)
		} else {
			pad = 0
		}
		if pad > 0 {
			c = mulPow10_128_128(c.dw0, c.dw1, pad)
			q -= pad
		}
	}
	return finalize_128_ctxnull(sign, q, c, RESIDUE_EXACT, rounding, ctxnull)
}

// d48_forceInexact signals Inexact on a result whose D48 value happened to land
// on a decimal128 point although the true value is not exact (its tail fell
// below 48 digits).
func d48_forceInexact(d d128, ctxnull *context) d128 {
	if ctxnull != nil {
		return ctxSignalInexact_ctxnull(ctxnull, d)
	}
	return d
}

// TINY_SCI_EXP is the tiny-argument rule's bound: e^u for sciExp(u) <
// TINY_SCI_EXP is decided by u's sign (d48_onePlusTiny). exp10 and pow test
// their base-10 argument against TINY_SCI_EXP − 1 (u·ln10, ln10 < 10).
//
// onePlusTiny is the correct rounding of e^u = 1 + u + u²/2 + … exactly when
// −5·10^−35 < u < 5·10^−34 (below one the grid step is 10^−34, above it
// 10^−33), so |u| < 10^−35 is the widest decade that holds for both signs
// (measured in D48SwiftWorkOrder.md Phase 0).
const TINY_SCI_EXP = -35

// d48_onePlusTiny is e^u for a tiny u (see TINY_SCI_EXP): 1 + u, decided by u's
// sign alone. Rounds 1⁺ (just above one) or 1⁻ (just below) into decimal128.
func d48_onePlusTiny(positive bool, rounding round, ctxnull *context) d128 {
	return d48_nudge_320(false, 0, u320FromDw0(1), positive, rounding, ctxnull)
}

// d48_nudge_320 is round(v ± tiny) for an exact value v = ±coeff·10^qExp (coeff
// != 0, below 10^77): v moved toward +∞ (up) or −∞ by less than any ulp, rounded
// once into decimal128, Inexact (LogP1ExpM1WorkOrder.md §3, the nudge helper;
// d48_onePlusTiny is v = 1). v may be a decimal128 point, a midpoint, or any
// other value. coeff is first scaled to at least 35 digits, so the final round
// always drops a digit; the tiny is then a residue below coeff's last digit:
// magnitude up is coeff + (0, ½), magnitude down (coeff − 1) + (½, 1).
func d48_nudge_320(sign bool, qExp int, coeffIn u320, up bool, rounding round, ctxnull *context) d128 {
	c := coeffIn
	q := qExp
	n := digitLen_320(c)
	if n < 35 {
		c = mulPow10_320(c, 35-n)
		q -= 35 - n
	}
	magnitudeUp := up != sign
	if !magnitudeUp {
		c = sub_320_320x320(c, u320FromDw0(1))
	}
	verify("", func() bool { return c.dw4 == 0 })
	res := RESIDUE_GT_HALF
	if magnitudeUp {
		res = RESIDUE_LT_HALF
	}
	return finalize_256_ctxnull(sign, q, u256{c.dw0, c.dw1, c.dw2, c.dw3}, res, rounding, ctxnull)
}

// d48_ambiguous48 is d48_ambiguous on a D48 result with e in units of its 48th
// significant digit: the coefficient is padded to 48 digits first, so the test
// does not depend on how many trailing zeros the D48 operation left.
func d48_ambiguous48(r d48, e u320) bool {
	if d48_isZero(r) {
		return false
	}
	c := d48_u320(r)
	q := d48_qExp(r)
	n := digitLen_320(c)
	if n < D48_DIGITS {
		c = mulPow10_320(c, D48_DIGITS-n)
		q -= D48_DIGITS - n
	}
	return d48_ambiguous(c, q, e)
}

// finalizeD48_ctxnull rounds a D48 result once into decimal128.
func finalizeD48_ctxnull(r d48, rounding round, ctxnull *context) d128 {
	return finalize_256_ctxnull(d48_signFlag(r), d48_qExp(r), u256{r.coeffDw0, r.coeffDw1, r.coeffDw2, 0},
		RESIDUE_EXACT, rounding, ctxnull)
}
