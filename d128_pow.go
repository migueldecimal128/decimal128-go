// IEEE 754-2019 Section 9.2 pow(x, y): x raised to a decimal power. Integer exponents
// in int32 delegate to d128_pown; a larger integer y keeps x's sign iff y is odd
// (powHugeIntegerY). The finite-nonzero transcendental core computes
// x^y = exp10(y * log10(x)) entirely in the 48-digit D48 intermediate (a single final
// round), with an integer exactness test for results that are exact decimals; an exact
// result takes the GDAS preferred exponent floor(y * Q(x)), an inexact one keeps all 34
// digits (IEEE 754-2019 Section 5.2). Correctly rounded except in the documented 1-ulp
// gap for results very close to 1 in directed modes (CrossPlatformArchitecture.md
// Section 12, "Transcendental Accuracy").

package decimal128

import (
	"math"
	"math/bits"
)

var const_D128_POW_ONE = d128_fromInt64(1)

const (
	const_Q_MAX  = 6111
	const_Q_TINY = -6176
)

// --- ladder ---

func d128_pow_tte(x, y d128) d128            { return d128_pow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil) }
func d128_pow_rnd(x, y d128, rnd round) d128 { return d128_pow_rnd_ctxnull(x, y, rnd, nil) }
func d128_pow_ctx(x, y d128, ctx *context) d128 {
	return d128_pow_rnd_ctxnull(x, y, ctx.round, ctx)
}

// IEEE 754-2019 Section 9.2 pow(x, y): x raised to a decimal power. An integer
// y delegates to pown (or, beyond 32 bits, to powHugeIntegerY); the
// finite-nonzero core computes x^y = 10^(y * log10(x)) in D48 with one final
// rounding, then applies the GDAS preferred exponent floor(y * Q(x)).
// Correctly rounded, except within the documented 1-ulp gap
// (CrossPlatformArchitecture.md, "Transcendental accuracy").
func d128_pow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Integer exponent → pown. Covers y = 0 (→ 1 for any x), every integer
	// power, and zero/inf/nan bases with integer y (pown owns those classes).
	if d128_isExactIntegral(y) {
		n := d128_toInt64TowardZero_ctxnull(y, nil)
		if n > math.MinInt32 && n <= math.MaxInt32 {
			return d128_pown_rnd_ctxnull(x, int(n), rounding, ctxnull)
		}
		return powHugeIntegerY(x, y, rounding, ctxnull)
	}

	// NaN operands (y is not an exact integer here).
	if d128_isNAN(x) || d128_isNAN(y) {
		// pow(1, qNaN) = 1 with no exception (§9.2).
		if d128_isFNZ(x) && !d128_signFlag(x) && d128_isQNAN(y) && cmpMagFnzFnz(x, const_D128_POW_ONE) == 0 {
			return d128_fromInt64(1)
		}
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}

	// x, y finite or infinite; y is non-integer (y may be ∞).
	if d128_isZER(x) {
		if d128_isINF(y) {
			// pow(±0, ±∞): y > 0 → +0, y < 0 → +∞ (no divideByZero, per §9.2).
			if d128_signFlag(y) {
				return const_D128_POS_INFINITY
			}
			return d128_zero(false, 0)
		}
		return powZerFnz(y, ctxnull)
	}
	if d128_isINF(x) {
		if d128_isINF(y) {
			return powFnzInf(x, y)
		}
		return powInfFnz(y)
	}
	// x finite non-zero.
	if d128_isINF(y) {
		return powFnzInf(x, y)
	}
	return powFnzFnz(x, y, rounding, ctxnull)
}

// pow(x, y) for an integer y outside pown's Int32 window (|y| >= 2^31): x's sign
// survives iff y is odd (IEEE 754-2019 Section 9.2.1); +/-0 and +/-1 take pown's cohorts.
// A finite non-zero |x| != 1 goes through powFromD48 with the result's sign; the
// result is never an exact decimal128 (PowHugeIntegerExponentWorkOrder.md HY-3).
func powHugeIntegerY(x, y d128, rounding round, ctxnull *context) d128 {
	if d128_isNAN(x) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	neg := d128_signFlag(x) && d128_isOddIntegral(y)
	yNeg := d128_signFlag(y)
	if d128_isZER(x) {
		if yNeg {
			return ctxSignalDivByZero_ctxnull(ctxnull, neg) // ±∞, divideByZero
		}
		return d128_zero(neg, d128_qExp(x)*9999) // pownZER's cohort (clamped)
	}
	if d128_isINF(x) {
		if yNeg {
			return d128_zero(neg, 0)
		}
		return d128_infinity(neg)
	}
	// |x| == 1: pown's cohort, clamp(y·Q(x), −33, 0); |y| >= 2^31 and Q(x) <= 0.
	if cmpMagFnzFnz(x, const_D128_POW_ONE) == 0 {
		q := 0
		if d128_qExp(x) < 0 && !yNeg {
			q = -33
		}
		return d128_oneWithQExp(neg, q)
	}
	return powFromD48(x, y, 0, rounding, ctxnull, neg, true)
}

// --- special classes (non-integer y) ---

// pow(+/-0, y), y finite non-integer. Sign is always positive (y can't be odd).
func powZerFnz(y d128, ctxnull *context) d128 {
	if d128_signFlag(y) {
		return ctxSignalDivByZero_ctxnull(ctxnull, false) // y < 0 → +∞, divideByZero
	}
	return d128_zero(false, 0) // y > 0 → +0
}

// pow(+/-inf, y), y finite non-integer.
func powInfFnz(y d128) d128 {
	if d128_signFlag(y) {
		return d128_zero(false, 0) // y < 0 → +0
	}
	return const_D128_POS_INFINITY // y > 0 → +∞
}

// pow(x, +/-inf), x finite non-zero or +/-inf.
func powFnzInf(x, y d128) d128 {
	if d128_isFNZ(x) && cmpMagFnzFnz(x, const_D128_POW_ONE) == 0 {
		return d128_fromInt64(1) // pow(±1, ±∞) = 1
	}
	xLessThanOne := d128_isFNZ(x) && d128_sciExp(x) < 0
	yNeg := d128_signFlag(y)
	if xLessThanOne != yNeg {
		return d128_zero(false, 0)
	}
	return const_D128_POS_INFINITY
}

// --- finite non-zero core ---

func powFnzFnz(x, y d128, rounding round, ctxnull *context) d128 {
	// Negative base with non-integer exponent → invalid.
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causePowerOfNegBaseNonIntegerExponent, const_D128_QNAN0)
	}
	// GDAS preferred exponent ⌊y·Q(x)⌋, taken by exact results only.
	preferred := powPreferredQExp(x, y)
	return powFromD48(x, y, preferred, rounding, ctxnull, false, false)
}

// --- huge integer y ---

func powPreferredQExp(x, y d128) int {
	qX := d128_qExp(x)
	if qX == 0 {
		return 0
	}
	if d128_sciExp(y) >= 5 {
		if d128_signFlag(y) {
			return const_Q_TINY
		}
		return const_Q_MAX
	}
	// y·Q(x) is exact in D48 (34 + 5 digits).
	prod := d48_multiply(d48_fromD128(y, d128_signFlag(y)), d48_fromInt32(qX))
	floored := d48_roundToIntegralTowardNegative(prod)
	l := d48_toLongOrMin(floored)
	return int(max(min(l, const_Q_MAX), const_Q_TINY))
}

// pow(x, y) = 10^u, u = y*log10(x), over D48, rounded once. The D48 bound grows
// with |u|: log10(x)'s relative error ~E_LOG_D48 units becomes an absolute error
// |u|*1e-44 in u, a relative error ~= 2.3*|u|*1e-44 of 10^u: < 1000*|u| units.
// An ambiguous result goes to the exact rules (powExact, then the tiny-argument
// rule); one they don't decide is rounded as is (the documented 1-ulp gap).
// `sign` is the result's sign (true only for a negative base to an odd huge
// integer y). `integerY`: y is an integer with |y| >= 2^31, so x^y is never an
// exact decimal128 -- no exact rules, and a D48 u that lands on an integer is
// still Inexact (PowHugeIntegerExponentWorkOrder.md HY-3).
func powFromD48(x, y d128, preferred int, rounding round, ctxnull *context, sign, integerY bool) d128 {
	log10X48 := d48_logMagnitude(d48_fromD128(x, false), true)
	u := d48_multiply(d48_fromD128(y, d128_signFlag(y)), log10X48)
	if integerY {
		if t, ok := exp10Trivial(sign, u, rounding, ctxnull); ok {
			return d48_forceInexact(t, ctxnull)
		}
		s := exp10Scaled_d48(u)
		return finalizeScaled(sign, s.z, s.n, rounding, ctxnull)
	}
	// An integer u in range: x^y = 10^u exactly, at the preferred exponent
	// (pow(100, 0.5) = 10, not 1E+1). Non-integers read as math.MinInt64.
	if l := d48_toLongOrMin(u); (l >= -6176) && (l <= 6144) {
		return d48_exactWithPreferred(sign, u128{1, 0}, int(l), preferred, rounding, ctxnull)
	}
	if t, ok := exp10Trivial(sign, u, rounding, ctxnull); ok {
		return t
	}
	s := exp10Scaled_d48(u)
	uMag := mulPow10_320(u320FromDw0(1000), max(0, d48_sciExp(u)+1))
	if ambiguousScaled(s, add_320_320x320(const_E_EXP_D48, uMag)) {
		if r, ok := powExact(x, y, preferred, rounding, ctxnull); ok {
			return r
		}
		if !d48_isZero(u) && (d48_sciExp(u) < const_TINY_SCI_EXP-1) {
			return d48_forceInexact(d48_onePlusTiny(!d48_signFlag(u), rounding, ctxnull), ctxnull)
		}
	}
	return finalizeScaled(sign, s.z, s.n, rounding, ctxnull)
}

// x^y when it is exactly a decimal128 value (D48ZigPrototypeWorkOrder.md Section 6.7),
// for x > 0 and y not an integer; ok = false otherwise. Write y = m/n in lowest terms
// (n = 2^a2*5^a5 >= 2) and x = 2^A*5^B*w with w coprime to 10. x^y is a finite
// decimal iff n | A, n | B and w = r^n; then x^y = 2^(A*m/n)*5^(B*m/n)*r^m. For
// w > 1, r >= 3 and r^m < 10^34 bound n and m by 71, and m must be positive (a
// negative power of r doesn't terminate). Integer operations only; the result
// takes the exponent closest to `preferred` (pow's preferred exponent).
func powExact(x, y d128, preferred int, rounding round, ctxnull *context) (d128, bool) {
	// y = ±cy·10^−k with cy free of trailing zeros, then reduced to m/n.
	cy, qy := stripTens(d128_u128(y), d128_qExp(y))
	if qy >= 0 {
		return d128{}, false // an integer y never reaches this path
	}
	k := -qy
	t2 := 0
	for (t2 < k) && ((cy.dw0 & 1) == 0) {
		cy = shrU128(cy, 1)
		t2 += 1
	}
	t5 := 0
	for t5 < k {
		q, r := divSmall(cy, 5)
		if r != 0 {
			break
		}
		cy = q
		t5 += 1
	}
	m := cy
	a2 := k - t2
	a5 := k - t5
	// |A|, |B| < 2^14: a larger n divides them only when x = 1 (decided elsewhere).
	if (a2 > 20) || (a5 > 10) {
		return d128{}, false
	}
	n := 1 << a2
	i := 0
	for i < a5 {
		n *= 5
		i += 1
	}

	// x = 2^A·5^B·w.
	cx := d128_u128(x)
	p2 := ctz128(cx)
	cx = shrU128(cx, p2)
	p5 := 0
	for {
		q, r := divSmall(cx, 5)
		if r != 0 {
			break
		}
		cx = q
		p5 += 1
	}
	bigA := p2 + d128_qExp(x)
	bigB := p5 + d128_qExp(x)
	if ((bigA % n) != 0) || ((bigB % n) != 0) {
		return d128{}, false
	}

	// r^m, with w = r^n checked one root at a time (each step must be exact).
	rm := u128{1, 0}
	if cx != (u128{1, 0}) {
		if (n > 71) || d128_signFlag(y) || u128GtSmall(m, 71) {
			return d128{}, false
		}
		r := cx
		i = 0
		for i < a5 {
			root, ok := iroot5(r)
			if !ok {
				return d128{}, false
			}
			r = root
			i += 1
		}
		i = 0
		for i < a2 {
			sq := isqrt_128(uint64(r.dw0), uint64(r.dw1))
			if mul_128_64x64(int64(sq), int64(sq)) != r {
				return d128{}, false
			}
			r = u128{int64(sq), 0}
			i += 1
		}
		j := int64(0)
		for j < m.dw0 { // m <= 71
			t, ok := u128MulChecked(rm, r)
			if !ok || !isLt_pow10_34_bool_128(t.dw0, t.dw1) {
				return d128{}, false
			}
			rm = t
			j += 1
		}
	}

	// x^y = 2^ap·5^bp·r^m = coeff·10^min(ap, bp), ap = (A/n)·±m, bp = (B/n)·±m.
	// Swift computes these in Int128. Go has none: m can be up to 10^34 only
	// when w = 1, and then a nonzero A/n or B/n with m >= 2^40 puts ap or bp
	// (or both, equal, with e = ap) beyond the dd / e bounds below, so that
	// case returns false without the product; otherwise |A/n|·m < 2^54.
	var ap, bp int64
	if (bigA/n != 0) || (bigB/n != 0) {
		if u128GtSmall(m, (1<<40)-1) {
			return d128{}, false
		}
		mS := m.dw0
		if d128_signFlag(y) {
			mS = -mS
		}
		ap = int64(bigA/n) * mS
		bp = int64(bigB/n) * mS
	}
	dd := ap - bp
	if (dd > 112) || (dd < -48) {
		return d128{}, false // 2^dd or 5^−dd alone is over 34 digits
	}
	e := min(ap, bp)
	if (e < -6176-40) || (e > 6144+40) {
		return d128{}, false // fully out of range: the D48 path decides
	}
	f := u128{1, 0}
	if dd >= 0 {
		f = shlU128(f, int(dd))
	} else {
		i = 0
		for i < int(-dd) {
			f = mul_128_128x128(f.dw0, f.dw1, 5, 0)
			i += 1
		}
	}
	coeff, ok := u128MulChecked(rm, f)
	if !ok || !isLt_pow10_34_bool_128(coeff.dw0, coeff.dw1) {
		return d128{}, false
	}
	return d48_exactWithPreferred(false, coeff, int(e), preferred, rounding, ctxnull), true
}

// The exact integer fifth root of v, or ok = false: binary search over r < 7*10^6
// (r^5 < 10^34), so no floating point and at most 23 steps.
func iroot5(v u128) (u128, bool) {
	if !isLt_pow10_34_bool_128(v.dw0, v.dw1) {
		return u128{}, false
	}
	var lo, hi int64 = 0, 7_000_000
	for lo < hi {
		mid := (lo + hi + 1) >> 1
		m4 := mul_128_64x64(mid*mid, mid*mid) // mid < 2^23: mid^5 < 2^115
		p := mul_128_128x128(m4.dw0, m4.dw1, mid, 0)
		if cmp_int_128x128(p.dw0, p.dw1, v.dw0, v.dw1) <= 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	l4 := mul_128_64x64(lo*lo, lo*lo)
	return u128{lo, 0}, mul_128_128x128(l4.dw0, l4.dw1, lo, 0) == v
}

// --- 128-bit integer helpers for the exact rules (Swift uses UInt128) ---

// divSmall is c / d and c mod d for a one-word d > 0.
func divSmall(c u128, d int64) (u128, int64) {
	qr := divRem_q128r64_128x64(c.dw0, c.dw1, d)
	return u128{qr.quotDw0, qr.quotDw1}, qr.remDw0
}

// u128MulChecked is a·b; ok = false on 128-bit overflow.
func u128MulChecked(a, b u128) (u128, bool) {
	p := mul_256_128x128(a.dw0, a.dw1, b.dw0, b.dw1)
	return u128{p.dw0, p.dw1}, (p.dw2 | p.dw3) == 0
}

// ctz128 is the number of trailing zero bits of a nonzero c.
func ctz128(c u128) int {
	if c.dw0 != 0 {
		return bits.TrailingZeros64(uint64(c.dw0))
	}
	return 64 + bits.TrailingZeros64(uint64(c.dw1))
}

// u128GtSmall is c > v for a one-word v.
func u128GtSmall(c u128, v uint64) bool {
	return (c.dw1 != 0) || (uint64(c.dw0) > v)
}

// --- integrality predicates ---

// True iff x is an exact integer (a finite value with no fractional part).
func d128_isExactIntegral(x d128) bool {
	if !d128_isFIN(x) {
		return false
	}
	if d128_isZER(x) {
		return true
	}
	q := d128_qExp(x)
	if q >= 0 {
		return true
	}
	drop := -q
	digitLen := d128_digitLen(x)
	if drop >= digitLen {
		return false // |x| < 1, nonzero
	}
	qr := div_128_pow10_q128res(d128_u128(x), drop)
	return qr.residue == const_RESIDUE_EXACT
}

// True iff x is an odd integer. The caller guarantees x is an exact integer.
func d128_isOddIntegral(x d128) bool {
	q := d128_qExp(x)
	if q > 0 {
		return false // a multiple of 10
	}
	c := d128_u128(x)
	if q == 0 {
		return (c.dw0 & 1) != 0
	}
	qr := div_128_pow10_q128res(c, -q)
	return (qr.quotDw0 & 1) != 0
}
