// IEEE 754-2019 §9.2 lnp1, log2p1, log10p1, expm1, exp10m1
// (LogP1ExpM1WorkOrder.md), each in D48 with one final rounding. Correctly
// rounded except in the documented 1-ulp gap (CrossPlatformArchitecture.md,
// "Transcendental accuracy").
//
// Log family, L = ln(1 + x): |x| < 10^-5 runs the log1p series on x (exact in
// D48, accurate relative to x however small); otherwise t = 1 + x is formed in
// D48 -- exact for 10^-5 <= |x| < 10^48 -- and d48_logMagnitude(t) sees t − 1 = x
// exactly. log10p1 / log2p1 divide L by ln 10 / ln 2.
// Exp family: |x| < 10^-5 runs the expm1 series (on x·ln10 for exp10m1);
// otherwise the exp / exp10 kernel gives S = e^x / 10^x and r = S − 1 in D48,
// with the ambiguity bound scaled by the digits the subtraction cancels.
// A result near a rounding boundary goes, in order, to the exact rules, the tiny
// rules, then R2 / RN / RP (§3); one none of them decides is rounded as is.

package decimal128

import "math/bits"

// Ambiguity bounds, in units of the D48 result's 48th digit (§2; Phase 0 maxima:
// series 6, log kernel 570, exp kernel 45 units of S). The +1 is the divide by
// ln 10 / ln 2 (log family).
const (
	const_E_SERIES_D48 = 100
	const_E_LOG_D48    = 10_000

	// const_SERIES_SCI_EXP is the series / kernel switch: |x| < 10^-5, sciExp(x) < -5.
	const_SERIES_SCI_EXP = -5
	// const_R2_SCI_EXP: R2 holds for sciExp(x) < it (Phase 0: valid through −18, fails at −17).
	const_R2_SCI_EXP = -17
	// const_RN_SCI_EXP: RN holds for sciExp(δ) + k < it (valid through −36, fails at −35).
	const_RN_SCI_EXP = -35

	const_LOGP1_BASE_E  = 0
	const_LOGP1_BASE_2  = 2
	const_LOGP1_BASE_10 = 10
)

// MARK: - lnp1 / log2p1 / log10p1 ladders

func d128_lnp1_tte(x d128) d128               { return d128_lnp1_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_lnp1_rnd(x d128, rnd round) d128    { return d128_lnp1_rnd_ctxnull(x, rnd, nil) }
func d128_lnp1_ctx(x d128, ctx *context) d128 { return d128_lnp1_rnd_ctxnull(x, ctx.round, ctx) }
func d128_lnp1_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logP1Dispatch(x, const_LOGP1_BASE_E, rounding, ctxnull)
}

func d128_log2p1_tte(x d128) d128               { return d128_log2p1_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_log2p1_rnd(x d128, rnd round) d128    { return d128_log2p1_rnd_ctxnull(x, rnd, nil) }
func d128_log2p1_ctx(x d128, ctx *context) d128 { return d128_log2p1_rnd_ctxnull(x, ctx.round, ctx) }
func d128_log2p1_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logP1Dispatch(x, const_LOGP1_BASE_2, rounding, ctxnull)
}

func d128_log10p1_tte(x d128) d128               { return d128_log10p1_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_log10p1_rnd(x d128, rnd round) d128    { return d128_log10p1_rnd_ctxnull(x, rnd, nil) }
func d128_log10p1_ctx(x d128, ctx *context) d128 { return d128_log10p1_rnd_ctxnull(x, ctx.round, ctx) }
func d128_log10p1_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	return logP1Dispatch(x, const_LOGP1_BASE_10, rounding, ctxnull)
}

// MARK: - expm1 / exp10m1 ladders

func d128_expm1_tte(x d128) d128               { return d128_expm1_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_expm1_rnd(x d128, rnd round) d128    { return d128_expm1_rnd_ctxnull(x, rnd, nil) }
func d128_expm1_ctx(x d128, ctx *context) d128 { return d128_expm1_rnd_ctxnull(x, ctx.round, ctx) }
func d128_expm1_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return x // expm1(±0) = ±0
		}
		return expM1FNZ(x, rounding, ctxnull)
	}
	return expM1NonFinite(x, ctxnull)
}

func d128_exp10m1_tte(x d128) d128               { return d128_exp10m1_rnd_ctxnull(x, const_ROUND_TIES_TO_EVEN, nil) }
func d128_exp10m1_rnd(x d128, rnd round) d128    { return d128_exp10m1_rnd_ctxnull(x, rnd, nil) }
func d128_exp10m1_ctx(x d128, ctx *context) d128 { return d128_exp10m1_rnd_ctxnull(x, ctx.round, ctx) }
func d128_exp10m1_rnd_ctxnull(x d128, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return x // exp10m1(±0) = ±0
		}
		return exp10M1FNZ(x, rounding, ctxnull)
	}
	return expM1NonFinite(x, ctxnull)
}

// MARK: - Special values (IEEE 754-2019 §9.2, LP-6)

// logP1Dispatch: −1 → −∞ with divideByZero; below −1 (and −∞) → invalid; +∞ → +∞.
func logP1Dispatch(x d128, base int, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return x // ±0, sign and exponent kept
		}
		if d128_signFlag(x) {
			cmp := cmpMagFnzFnz(x, d128_fromInt64(1))
			if cmp == 0 {
				return ctxSignalDivByZero_ctxnull(ctxnull, true)
			}
			if cmp > 0 {
				return ctxSignalInvalidOperation_ctxnull(ctxnull, causeLogOfNegNumber)
			}
		}
		return logP1FNZ(x, base, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeLogOfNegNumber)
	}
	return const_D128_POS_INFINITY
}

// expM1NonFinite: +∞ → +∞, −∞ → −1 exactly, NaN propagates.
func expM1NonFinite(x d128, ctxnull *context) d128 {
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	if d128_signFlag(x) {
		return d128_fromInt64(-1)
	}
	return const_D128_POS_INFINITY
}

// MARK: - Log family

// logP1FNZ is ln(1 + x) / ln(base) for a finite non-zero x > −1.
func logP1FNZ(x d128, base int, rounding round, ctxnull *context) d128 {
	xd := d48_fromD128(x, d128_signFlag(x))
	series := d128_sciExp(x) < const_SERIES_SCI_EXP
	var t, l d48
	if series {
		l = d48_log1pSeries(xd)
	} else {
		t = d48_add(const_D48_ONE, xd) // 1 + x, kernel path only
		l = d48_logMagnitude(t, false)
	}
	if base == const_LOGP1_BASE_E {
		// The kernel path has no rule after it, so (as ln) it skips the test.
		if series && d48_ambiguous48(l, u320FromDw0(const_E_SERIES_D48)) {
			if d128_sciExp(x) < const_TINY_SCI_EXP {
				return nudgeD128(x, false, rounding, ctxnull) // x − x²/2: x nudged down
			}
			if r, ok := secondOrder(x, false, rounding, ctxnull); ok {
				return r
			}
		}
		return d48_forceInexact(finalizeD48_ctxnull(l, rounding, ctxnull), ctxnull)
	}
	div := const_D48_LN_TABLE[2]
	if base == const_LOGP1_BASE_10 {
		div = const_D48_LN10
	}
	r := d48_divide(l, div)
	// The series path has no rule (x/ln2, x/ln10 are irrational): only the kernel
	// path runs the test.
	if !series && d48_ambiguous48(r, u320FromDw0(const_E_LOG_D48+1)) {
		if d128_sciExp(x) < 34 {
			// 1 + x is exact (t); every exact case is below 10^34.
			if k, ok := logP1ExactPower(t, base); ok {
				return d128_fromInt64(k)
			}
		} else if base == const_LOGP1_BASE_10 && isExactPowerOf10(x) {
			// RP: log10(10^k + 1) = k + 1/(10^k·ln10) − …: k nudged up.
			return d48_nudge_320(false, 0, u320FromDw0(int64(d128_sciExp(x))), true, rounding, ctxnull)
		}
	}
	return d48_forceInexact(finalizeD48_ctxnull(r, rounding, ctxnull), ctxnull)
}

// logP1ExactPower is k if the exact t = 1 + x is base^k (base 2 or 10, k != 0).
func logP1ExactPower(t d48, base int) (int, bool) {
	c := d48_u320(t)
	q := d48_qExp(t)
	for {
		qr := divPow10_q320res_320_recipmul_1(c)
		if qr.residue != const_RESIDUE_EXACT {
			break
		}
		c = qr.quot()
		q++
	}
	if (c.dw2 | c.dw3 | c.dw4) != 0 {
		return 0, false // c' >= 2^128: neither 1, 2^k (k <= 112) nor 5^k (k <= 34)
	}
	lo, hi := uint64(c.dw0), uint64(c.dw1)
	if base == const_LOGP1_BASE_10 {
		return q, lo == 1 && hi == 0 && q != 0
	}
	if q == 0 {
		// t = c': a power of two other than 1
		if bits.OnesCount64(lo)+bits.OnesCount64(hi) != 1 || (lo == 1 && hi == 0) {
			return 0, false
		}
		if hi != 0 {
			return 64 + bits.TrailingZeros64(hi), true
		}
		return bits.TrailingZeros64(lo), true
	}
	if q > 0 || q < -34 {
		return 0, false // c'·10^q has a factor 5; 2^k needs k >= −34
	}
	p := fromLoHi_128_64x64(1, 0) // t = 2^q = 5^−q · 10^q; 5^34 < 2^80
	for i := 0; i < -q; i++ {
		p = mul_128_128x128(p.dw0, p.dw1, 5, 0)
	}
	return q, p.dw0 == c.dw0 && p.dw1 == c.dw1
}

// MARK: - Exp family

// d48_expm1Series is expm1(x) = x·Σ x^(k−1)/k!, Horner in D48 (const_D48_EXPM1_WEIGHTS,
// 9 terms): 48 digits relative to x for |x| < 10^-5.
func d48_expm1Series(x d48) d48 {
	w := const_D48_EXPM1_WEIGHTS
	acc := w[len(w)-1]
	for k := len(w) - 2; k >= 0; k-- {
		acc = d48_fma(acc, x, w[k])
	}
	return d48_multiply(acc, x)
}

// expM1FNZ is e^x − 1 for a finite non-zero x.
func expM1FNZ(x d128, rounding round, ctxnull *context) d128 {
	neg := d128_signFlag(x)
	sci := d128_sciExp(x)
	if !neg && sci >= 6 {
		return finalizeOverflow_ctxnull(false, rounding, ctxnull)
	}
	if neg && sci >= 2 {
		return minusOnePlusTiny(rounding, ctxnull) // x <= −100: e^x < 10^-43
	}
	if sci < const_SERIES_SCI_EXP {
		r := d48_expm1Series(d48_fromD128(x, neg))
		if d48_ambiguous48(r, u320FromDw0(const_E_SERIES_D48)) {
			if sci < const_TINY_SCI_EXP {
				return nudgeD128(x, true, rounding, ctxnull) // x + x²/2: x nudged up
			}
			if v, ok := secondOrder(x, true, rounding, ctxnull); ok {
				return v
			}
		}
		return d48_forceInexact(finalizeD48_ctxnull(r, rounding, ctxnull), ctxnull)
	}
	return expM1Kernel(expScaled_d48(x), false, d48{}, rounding, ctxnull)
}

// exp10M1FNZ is 10^x − 1 for a finite non-zero x.
func exp10M1FNZ(x d128, rounding round, ctxnull *context) d128 {
	if d128_isExactIntegral(x) {
		return exp10M1Integer(x, rounding, ctxnull)
	}
	neg := d128_signFlag(x)
	xd := d48_fromD128(x, neg)
	if !neg && d48_cmpMag(xd, d48_fromInt32(6200)) > 0 {
		return finalizeOverflow_ctxnull(false, rounding, ctxnull) // x > 6200
	}
	if neg && d48_cmpMag(xd, d48_fromInt32(50)) > 0 {
		return minusOnePlusTiny(rounding, ctxnull) // x < −50
	}
	if d128_sciExp(x) < const_SERIES_SCI_EXP {
		// u = x·ln10 is irrational: no rule applies, so no ambiguity test.
		r := d48_expm1Series(d48_multiply(xd, const_D48_LN10))
		return d48_forceInexact(finalizeD48_ctxnull(r, rounding, ctxnull), ctxnull)
	}
	return expM1Kernel(exp10Scaled_d48(xd), true, xd, rounding, ctxnull)
}

// expM1Kernel rounds r = S − 1 for the kernel's S = z·10^n once. The bound scales
// with the digits the subtraction cancels: const_E_EXP_D48·10^max(0, sciExp(S) −
// sciExp(r)) + 1 units of r's 48th digit. isExp10m1 enables RN on xd.
func expM1Kernel(s scaled, isExp10m1 bool, xd d48, rounding round, ctxnull *context) d128 {
	bigS := d48_fromSignQExp_64x64x64(false, d48_qExp(s.z)+s.n, s.z.coeffDw0, s.z.coeffDw1, s.z.coeffDw2)
	r := d48_subtract(bigS, const_D48_ONE)
	shift := d48_sciExp(bigS) - d48_sciExp(r)
	if shift < 0 {
		shift = 0
	}
	e := add_320_320x320(mulPow10_320(const_E_EXP_D48, shift), u320FromDw0(1))
	if d48_ambiguous48(r, e) {
		if d48_sciExp(bigS) < const_TINY_SCI_EXP {
			return minusOnePlusTiny(rounding, ctxnull)
		}
		if isExp10m1 {
			if v, ok := nearNegativeInteger(xd, rounding, ctxnull); ok {
				return v
			}
		}
	}
	return d48_forceInexact(finalizeD48_ctxnull(r, rounding, ctxnull), ctxnull)
}

// exp10M1Integer is exp10m1's integer exit (LP-4): 10^k − 1 exact for |k| <= 34
// (99…9 at exponent 0, −0.99…9 at exponent k); 10^k nudged down for k >= 35; −1
// nudged up for k <= −35. Never through D48: for |k| >= 48 the D48 10^k − 1 is
// rounded.
func exp10M1Integer(x d128, rounding round, ctxnull *context) d128 {
	neg := d128_signFlag(x)
	var k int64
	switch {
	case d128_sciExp(x) >= 10 && neg:
		k = -1_000_000_000
	case d128_sciExp(x) >= 10:
		k = 1_000_000_000
	default:
		k = d48_toLongOrMin(d48_fromD128(x, neg))
	}
	if k > 6145 {
		return finalizeOverflow_ctxnull(false, rounding, ctxnull)
	}
	if k > 34 {
		return d48_nudge_320(false, int(k), u320FromDw0(1), false, rounding, ctxnull)
	}
	if k < -34 {
		return minusOnePlusTiny(rounding, ctxnull)
	}
	absK := int(k)
	if absK < 0 {
		absK = -absK
	}
	p := pow10U128(absK)
	nines := sub_128_128x128(p.dw0, p.dw1, 1, 0)
	if k > 0 {
		return d128Pack(false, 0, nines.dw0, nines.dw1)
	}
	return d128Pack(true, int(k), nines.dw0, nines.dw1)
}

// nearNegativeInteger is RN (§3): exp10m1 at x = k + δ, k = round(x) in [−34, −1],
// sciExp(δ) + k < const_RN_SCI_EXP: 10^x − 1 = (10^k − 1) + 10^k·(10^δ − 1), whose
// second term is below D48's resolution; the result is 10^k − 1 nudged by δ's sign.
func nearNegativeInteger(xd d48, rounding round, ctxnull *context) (d128, bool) {
	if !d48_signFlag(xd) {
		return d128{}, false
	}
	kd := d48_roundToIntegralTiesToAway(xd)
	k := int(d48_toLongOrMin(kd))
	if k < -34 || k > -1 {
		return d128{}, false
	}
	delta := d48_subtract(xd, kd) // exact: x has at most 34 digits
	if d48_isZero(delta) || d48_sciExp(delta)+k >= const_RN_SCI_EXP {
		return d128{}, false
	}
	nines := sub_320_320x320(mulPow10_320(u320FromDw0(1), -k), u320FromDw0(1))
	return d48_nudge_320(true, k, nines, !d48_signFlag(delta), rounding, ctxnull), true
}

// MARK: - Shared rules

// minusOnePlusTiny is −1 + tiny (Inexact): e^x or 10^x below every rounding step of −1.
func minusOnePlusTiny(rounding round, ctxnull *context) d128 {
	return d48_nudge_320(true, 0, u320FromDw0(1), true, rounding, ctxnull)
}

// nudgeD128 is the decimal128 x nudged toward +∞ (up) or −∞: the tiny rules.
func nudgeD128(x d128, up bool, rounding round, ctxnull *context) d128 {
	c := d128_u128(x)
	return d48_nudge_320(d128_signFlag(x), d128_qExp(x), u320{c.dw0, c.dw1, 0, 0, 0}, up, rounding, ctxnull)
}

// secondOrder is R2 (§3), lnp1 / expm1 series path, const_TINY_SCI_EXP <= sciExp(x) <
// const_R2_SCI_EXP, x of at most 8 significant digits: v2 = x ∓ x²/2 (lnp1 −, expm1
// +), exact, nudged toward x's sign (the x³ term has x's sign in both). Only such
// x can put v2 on a decimal128 point or midpoint. ok is false when the gate fails.
func secondOrder(x d128, isExpm1 bool, rounding round, ctxnull *context) (d128, bool) {
	if d128_sciExp(x) >= const_R2_SCI_EXP {
		return d128{}, false
	}
	cw := d128_u128(x)
	q := d128_qExp(x)
	for {
		qr := divRem_q128r64_128x64(cw.dw0, cw.dw1, 10)
		if qr.remDw0 != 0 {
			break
		}
		cw = fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		q++
	}
	if cw.dw1 != 0 || uint64(cw.dw0) >= 100_000_000 {
		return d128{}, false
	}
	c := cw.dw0
	// x = c·10^q, x²/2 = 5c²·10^(2q−1): v2 = ±(c·10^(1−q) ± 5c²)·10^(2q−1), with
	// 1 − q = digitLen(c) − sciExp(x) <= 8 + 35, so the lead stays below 10^51.
	lead := mulPow10_320(u320FromDw0(c), 1-q)
	half := u320FromDw0(5 * c * c) // 5c² < 5·10^16
	neg := d128_signFlag(x)
	mag := sub_320_320x320(lead, half)
	if isExpm1 != neg {
		mag = add_320_320x320(lead, half)
	}
	return d48_nudge_320(neg, 2*q-1, mag, !neg, rounding, ctxnull), true
}
