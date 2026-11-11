// IEEE 754-2019 §9.2 compound(x, n) = (1+x)^n for x ≥ -1 and integer n. Form
// t = 1+x, reject x < -1, then raise t to the n -- never from a t rounded to 34
// digits, whose error the power multiplies by |n| (CompoundD48OnePlusXWorkOrder.md).
// Three cases by the digit span of 1 + x:
//   - at most 34 digits: t is an exact D128, so the D128 add and pown are already
//     correctly rounded (the fast path, and the common case for rates);
//   - 35..48 digits: t is exact in D48;
//   - more than 48: t is 1 + x rounded to odd at 48 digits, which keeps the side
//     of the dropped part for the final rounding in the directed modes.
// Past the fast path, t^n runs on pown's D48 bodies (n = 2 and n = −1 included,
// CX-3) and is rounded once; n = 1 is the single D128 add. x = -1 (t = 0) and the
// special classes are handled directly; a zero result carries the preferred
// exponent ⌊|n|·min(0, Q(x))⌋.

package decimal128

var const_D128_COMPOUND_ONE = d128_fromInt64(1)

func d128_compound_tte(x d128, n int) d128 {
	return d128_compound_rnd_ctxnull(x, n, const_ROUND_TIES_TO_EVEN, nil)
}
func d128_compound_rnd(x d128, n int, rnd round) d128 {
	return d128_compound_rnd_ctxnull(x, n, rnd, nil)
}
func d128_compound_ctx(x d128, n int, ctx *context) d128 {
	return d128_compound_rnd_ctxnull(x, n, ctx.round, ctx)
}

func d128_compound_rnd_ctxnull(x d128, n int, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return d128_fromInt64(1) // (1 ± 0)^n = 1
		}
		return compoundFNZ(x, n, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		// compound(x, 0) is 1 for a quiet NaN (§9.2).
		if n == 0 && d128_isQNAN(x) {
			return d128_fromInt64(1)
		}
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		// -∞ is below -1 → invalid (even for n == 0).
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeCompoundXLtNegOne)
	}
	if n > 0 {
		return const_D128_POS_INFINITY // (1 + ∞)^(n>0) = +∞
	}
	if n == 0 {
		return d128_fromInt64(1) // compound(+∞, 0) = 1
	}
	return d128_zero(false, 0) // n < 0 → +0
}

// compoundMinusOne is x = -1 exactly (n != 0): +∞ with divideByZero for n < 0,
// else +0 with the preferred exponent ⌊n·min(0, Q(x))⌋.
func compoundMinusOne(x d128, n int, ctxnull *context) d128 {
	if n < 0 {
		return ctxSignalDivByZero_ctxnull(ctxnull, false) // +∞
	}
	nCapped := min(n, 10_000) // keep the product inside int
	return d128_zero(false, nCapped*min(0, d128_qExp(x)))
}

// compoundZeroResult gives a zero power the preferred exponent ⌊|n|·min(0, Q(x))⌋.
func compoundZeroResult(x d128, n int, result d128) d128 {
	if d128_isZER(result) {
		qPreferred := max(min(-satMul(-min(0, d128_qExp(x)), absU64(n)), 6111), -6176)
		return d128_zero(false, qPreferred)
	}
	return result
}

// absU64 is |n| as uint64 (exact for math.MinInt).
func absU64(n int) uint64 {
	if n < 0 {
		return -uint64(n)
	}
	return uint64(n)
}

// compoundOnePlusXSpan is the digit span of 1 + x: from 10^max(sciExp(x), 0)
// down to 10^min(qExp(x), 0). A span of at most k means 1 + x is exact in k
// digits. Over 48 it is not exact: that needs |x| < 10^−14 or x >= 10^48, where
// nothing cancels. (Cancellation only shortens span-35 sums, x in (−1, −0.1]
// with qExp −34; they take the D48 path.)
func compoundOnePlusXSpan(x d128) int {
	return max(d128_sciExp(x), 0) - min(d128_qExp(x), 0) + 1
}

// compoundRoundToOdd is 1 + x rounded to odd at 48 digits, from t = 1 + x
// rounded ties-to-even (span > 48). Round-to-odd truncates and, if anything was
// dropped, makes the last digit odd: of the two 48-digit neighbours of 1 + x
// exactly one is odd, and t is one of them. If t is even, step one unit toward
// 1 + x. The side comes from the residual r = (1 + x) − t, whose inner
// difference is exact in D48: span > 48 means x >= 10^48 (t − x is a few units
// of t's last digit) or |x| < 10^−14 (t − 1 has at most 34 digits); the outer
// one only needs its sign, which rounding keeps.
func compoundRoundToOdd(x d128, t48 d48) d48 {
	one := d48_fromInt32(1)
	x48 := d48_fromD128(x, d128_signFlag(x))
	var r d48
	if d128_sciExp(x) >= const_D48_DIGITS {
		r = d48_subtract(one, d48_subtract(t48, x48))
	} else {
		r = d48_subtract(x48, d48_subtract(t48, one))
	}
	if d48_isZero(r) {
		return t48 // 1 + x was exact after all
	}
	// Put t on the 48-digit grid so a unit is its last digit.
	c := d48_u320(t48)
	q := d48_qExp(t48)
	if n := digitLen_320(c); n < const_D48_DIGITS {
		c = mulPow10_320(c, const_D48_DIGITS-n)
		q -= const_D48_DIGITS - n
	}
	if (c.dw0 & 1) == 0 {
		// t is even: 999…9 is odd, so a step up never reaches 49 digits.
		if d48_signFlag(r) {
			c = sub_320_320x320(c, u320FromDw0(1))
		} else {
			c = add_320_320x320(c, u320FromDw0(1))
		}
	}
	return d48_fromSignQExp_64x64x64(false, q, c.dw0, c.dw1, c.dw2)
}

// compoundStripToU128 is t with trailing zeros stripped while its coefficient
// needs the third limb. A stripped coefficient >= 2^128 satisfies neither exact
// rule for t = 1 + x with a 34-digit x: c'^n < 10^34 fails outright, and
// c' = 5^b >= 2^128 (b = 56, m = 2) would need x to span at least 39 digits.
func compoundStripToU128(t d48) d48 {
	c := d48_u320(t)
	q := d48_qExp(t)
	for c.dw2 != 0 {
		qr := divPow10_q320res_320_recipmul_1(c)
		if qr.residue != const_RESIDUE_EXACT {
			break
		}
		c = qr.quot()
		q++
	}
	return d48_fromSignQExp_64x64x64(false, q, c.dw0, c.dw1, c.dw2)
}

// compoundPowerD48 is t^n (n not in {0, 1}) for t = 1 + x > 0 held in D48,
// rounded once into decimal128. tExact: t is 1 + x exactly (else it is rounded
// to odd and the result inexact).
func compoundPowerD48(t d48, tExact bool, n int, rounding round, ctxnull *context) d128 {
	// The exact rules need t exact and its stripped coefficient in a u128.
	s := t
	rulesApply := false
	if tExact {
		s = compoundStripToU128(t)
		rulesApply = s.coeffDw2 == 0
	}
	sc := u128{s.coeffDw0, s.coeffDw1}
	if n > 0 {
		exact := rulesApply && pown_positiveExactCoeff(sc, uint64(n))
		return pown_d48Positive(t, uint64(n), exact, rounding, ctxnull)
	}
	absPow := absU64(n)
	if rulesApply {
		// The preferred exponent is −absPow·Q(t), from t before stripping.
		if exact, ok := pown_negativeExactCoeff(false, sc, d48_qExp(s), d48_qExp(t), absPow, rounding, ctxnull); ok {
			return exact
		}
	}
	return pown_d48Negative(t, absPow, rounding, ctxnull)
}

func compoundFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	// |n| > 2^31: pown's log route on ε = x itself, with the tiny rule; neither
	// t = 1 + x nor round-to-odd (PownLargeNWorkOrder.md LN-4).
	if absU64(n) > const_POWN_LARGE_N {
		if d128_signFlag(x) {
			cmp := cmpMagFnzFnz(x, const_D128_COMPOUND_ONE)
			if cmp > 0 {
				return ctxSignalInvalidOperation_ctxnull(ctxnull, causeCompoundXLtNegOne) // x < -1
			}
			if cmp == 0 {
				return compoundMinusOne(x, n, ctxnull)
			}
		}
		eps := d48_fromD128(x, d128_signFlag(x))
		return compoundZeroResult(x, n, pown_largeN(eps, n, false, true, rounding, ctxnull))
	}
	span := compoundOnePlusXSpan(x)
	if span <= 34 {
		// 1 + x is an exact D128: the add below never rounds.
		t := d128_add_rnd_ctxnull(const_D128_COMPOUND_ONE, x, rounding, ctxnull)
		if d128_signFlag(t) && !d128_isZER(t) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeCompoundXLtNegOne) // x < -1
		}
		if n == 0 {
			return d128_fromInt64(1)
		}
		if d128_isZER(t) {
			return compoundMinusOne(x, n, ctxnull)
		}
		return compoundZeroResult(x, n, d128_pown_rnd_ctxnull(t, n, rounding, ctxnull))
	}
	// t = 1 + x in D48. Rounding to 48 digits never changes the sign or makes a
	// non-zero sum zero, so the sign and zero tests below are exact.
	t := d48_add(d48_fromInt32(1), d48_fromD128(x, d128_signFlag(x)))
	if d48_signFlag(t) && !d48_isZero(t) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeCompoundXLtNegOne) // x < -1
	}
	if n == 0 {
		return d128_fromInt64(1)
	}
	if d48_isZero(t) {
		return compoundMinusOne(x, n, ctxnull)
	}
	if n == 1 {
		return d128_add_rnd_ctxnull(const_D128_COMPOUND_ONE, x, rounding, ctxnull) // one rounding already
	}
	if span > const_D48_DIGITS {
		t = compoundRoundToOdd(x, t)
	}
	return compoundZeroResult(x, n, compoundPowerD48(t, span <= const_D48_DIGITS, n, rounding, ctxnull))
}
