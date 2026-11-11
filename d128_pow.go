// IEEE 754-2019 §9.2 pow(x, y): x raised to a decimal power. Integer exponents
// delegate to pown; the finite-nonzero transcendental core computes
// x^y = exp10(y · log10(x)) entirely in the 48-digit D48 intermediate (a single
// final round), with an integer exactness test for results that are exact
// decimals, then applies the GDAS preferred exponent ⌊y·Q(x)⌋. Correctly
// rounded except in the documented 1-ulp gap for results very close to 1 in
// directed modes (CrossPlatformArchitecture.md §12, "Transcendental accuracy").

package decimal128

import (
	"math"
	"math/bits"
)

var d128PowOne = d128_fromInt64(1)

const (
	qMaxPow  = 6111
	qTinyPow = -6176
)

// --- ladder ---

func d128_pow_tte(x, y d128) d128            { return d128_pow_rnd_ctxnull(x, y, ROUND_TIES_TO_EVEN, nil) }
func d128_pow_rnd(x, y d128, rnd round) d128 { return d128_pow_rnd_ctxnull(x, y, rnd, nil) }
func d128_pow_ctx(x, y d128, ctx *context) d128 {
	return d128_pow_rnd_ctxnull(x, y, ctx.round, ctx)
}

func d128_pow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Integer exponent → pown. Covers y = 0 (→ 1 for any x), every integer
	// power, and zero/inf/nan bases with integer y (pown owns those classes).
	if d128_isExactIntegral(y) {
		n := d128_toInt64TowardZero_ctxnull(y, nil)
		if n > math.MinInt32 && n <= math.MaxInt32 {
			return d128_pown_rnd_ctxnull(x, int(n), rounding, ctxnull)
		}
		// |y| too large for pown: only a magnitude-1 base stays finite.
		if d128_isFNZ(x) && cmpMagFnzFnz(x, d128PowOne) == 0 {
			if d128_signFlag(x) && d128_isOddIntegral(y) {
				return d128_fromInt64(-1)
			}
			return d128_fromInt64(1)
		}
		// else fall through: a huge integer y overflows/underflows in the matrix.
	}

	// NaN operands (y is not an exact integer here).
	if d128_isNAN(x) || d128_isNAN(y) {
		// pow(1, qNaN) = 1 with no exception (§9.2).
		if d128_isFNZ(x) && !d128_signFlag(x) && d128_isQNAN(y) && cmpMagFnzFnz(x, d128PowOne) == 0 {
			return d128_fromInt64(1)
		}
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}

	// x, y finite or infinite; y is non-integer (y may be ∞).
	if d128_isZER(x) {
		if d128_isINF(y) {
			// pow(±0, ±∞): y > 0 → +0, y < 0 → +∞ (no divideByZero, per §9.2).
			if d128_signFlag(y) {
				return d128_POS_INFINITY()
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

// --- special classes (non-integer y) ---

// powZerFnz: pow(±0, y), y finite non-integer. Sign is always positive.
func powZerFnz(y d128, ctxnull *context) d128 {
	if d128_signFlag(y) {
		return ctxSignalDivByZero_ctxnull(ctxnull, false) // y < 0 → +∞, divideByZero
	}
	return d128_zero(false, 0) // y > 0 → +0
}

// powInfFnz: pow(±∞, y), y finite non-integer.
func powInfFnz(y d128) d128 {
	if d128_signFlag(y) {
		return d128_zero(false, 0) // y < 0 → +0
	}
	return d128_POS_INFINITY() // y > 0 → +∞
}

// powFnzInf: pow(x, ±∞), x finite non-zero or ±∞.
func powFnzInf(x, y d128) d128 {
	if d128_isFNZ(x) && cmpMagFnzFnz(x, d128PowOne) == 0 {
		return d128_fromInt64(1) // pow(±1, ±∞) = 1
	}
	xLessThanOne := d128_isFNZ(x) && d128_sciExp(x) < 0
	yNeg := d128_signFlag(y)
	if xLessThanOne != yNeg {
		return d128_zero(false, 0)
	}
	return d128_POS_INFINITY()
}

// --- finite non-zero core ---

func powFnzFnz(x, y d128, rounding round, ctxnull *context) d128 {
	// Negative base with non-integer exponent → invalid.
	if d128_signFlag(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causePowerOfNegBaseNonIntegerExponent)
	}
	// GDAS preferred exponent ⌊y·Q(x)⌋ (applied to a finite-nonzero result).
	preferred := powPreferredQExp(x, y)
	result := powFromD48(x, y, preferred, rounding, ctxnull)
	if d128_isFNZ(result) {
		return applyPreferredScale(result, preferred)
	}
	return result
}

func powPreferredQExp(x, y d128) int {
	qX := d128_qExp(x)
	if qX == 0 {
		return 0
	}
	if d128_sciExp(y) >= 5 {
		if d128_signFlag(y) {
			return qTinyPow
		}
		return qMaxPow
	}
	// y·Q(x) is exact in D48 (34 + 5 digits).
	prod := d48_multiply(d48_fromD128(y, d128_signFlag(y)), d48_fromInt32(qX))
	floored := d48_roundToIntegralTowardNegative(prod)
	l := d48_toLongOrMin(floored)
	return int(maxI64(minI64(l, qMaxPow), qTinyPow))
}

// powFromD48 is pow(x, y) = 10^u, u = y·log10(x), over D48, rounded once. The
// D48 bound grows with |u|: log10(x)'s relative error ~E_LOG_D48 units becomes
// an absolute error |u|·1e−44 in u, a relative error ≈ 2.3·|u|·1e−44 of 10^u:
// < 1000·|u| units. An ambiguous result goes to the exact rules (powExact, then
// the tiny-argument rule); one they don't decide is rounded as is (the
// documented 1-ulp gap).
func powFromD48(x, y d128, preferred int, rounding round, ctxnull *context) d128 {
	log10X48 := d48_logMagnitude(d48_fromD128(x, false), true)
	u := d48_multiply(d48_fromD128(y, d128_signFlag(y)), log10X48)
	if t, ok := exp10Trivial(false, u, rounding, ctxnull); ok {
		return t
	}
	s := exp10Scaled_d48(u)
	uMag := mulPow10_320(u320FromDw0(1000), max(0, d48_sciExp(u)+1))
	if ambiguousScaled(s, add_320_320x320(e_EXP_D48, uMag)) {
		if r, ok := powExact(x, y, preferred, rounding, ctxnull); ok {
			return r
		}
		if !d48_isZero(u) && (d48_sciExp(u) < TINY_SCI_EXP-1) {
			return d48_forceInexact(d48_onePlusTiny(!d48_signFlag(u), rounding, ctxnull), ctxnull)
		}
	}
	return finalizeScaled(false, s.z, s.n, rounding, ctxnull)
}

// powExact is x^y when it is exactly a decimal128 value
// (D48ZigPrototypeWorkOrder.md §6.7), for x > 0 and y not an integer; ok =
// false otherwise. Write y = m/n in lowest terms (n = 2^a2·5^a5 >= 2) and x =
// 2^A·5^B·w with w coprime to 10. x^y is a finite decimal iff n | A, n | B and
// w = r^n; then x^y = 2^(A·m/n)·5^(B·m/n)·r^m. For w > 1, r >= 3 and r^m <
// 10^34 bound n and m by 71, and m must be positive (a negative power of r
// doesn't terminate). Integer operations only; the result takes the exponent
// closest to `preferred` (pow's preferred exponent).
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
		t2++
	}
	t5 := 0
	for t5 < k {
		q, r := u128DivRemSmall(cy, 5)
		if r != 0 {
			break
		}
		cy = q
		t5++
	}
	m := cy
	a2 := k - t2
	a5 := k - t5
	// |A|, |B| < 2^14: a larger n divides them only when x = 1 (decided elsewhere).
	if (a2 > 20) || (a5 > 10) {
		return d128{}, false
	}
	n := 1 << a2
	for i := 0; i < a5; i++ {
		n *= 5
	}

	// x = 2^A·5^B·w.
	cx := d128_u128(x)
	p2 := u128TrailingZeros(cx)
	cx = shrU128(cx, p2)
	p5 := 0
	for {
		q, r := u128DivRemSmall(cx, 5)
		if r != 0 {
			break
		}
		cx = q
		p5++
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
		for i := 0; i < a5; i++ {
			root, ok := iroot5(r)
			if !ok {
				return d128{}, false
			}
			r = root
		}
		for i := 0; i < a2; i++ {
			sq := isqrt_128(uint64(r.dw0), uint64(r.dw1))
			if mul_128_64x64(int64(sq), int64(sq)) != r {
				return d128{}, false
			}
			r = u128{int64(sq), 0}
		}
		for j := int64(0); j < m.dw0; j++ { // m <= 71
			t, ok := u128MulChecked(rm, r)
			if !ok || !isLt_pow10_34_bool_128(t.dw0, t.dw1) {
				return d128{}, false
			}
			rm = t
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
		for i := 0; i < int(-dd); i++ {
			f = mul_128_128x128(f.dw0, f.dw1, 5, 0)
		}
	}
	coeff, ok := u128MulChecked(rm, f)
	if !ok || !isLt_pow10_34_bool_128(coeff.dw0, coeff.dw1) {
		return d128{}, false
	}
	return d48_exactWithPreferred(false, coeff, int(e), preferred, rounding, ctxnull), true
}

// iroot5 is the exact integer fifth root of v, or ok = false: binary search
// over r < 7·10^6 (r^5 < 10^34), so no floating point and at most 23 steps.
func iroot5(v u128) (u128, bool) {
	if !isLt_pow10_34_bool_128(v.dw0, v.dw1) {
		return u128{}, false
	}
	pow5 := func(r int64) u128 { // r < 2^23: r^5 < 2^115
		r2 := r * r
		r4 := mul_128_64x64(r2, r2)
		return mul_128_128x128(r4.dw0, r4.dw1, r, 0)
	}
	var lo, hi int64 = 0, 7_000_000
	for lo < hi {
		mid := (lo + hi + 1) >> 1
		p := pow5(mid)
		if cmp_int_128x128(p.dw0, p.dw1, v.dw0, v.dw1) <= 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if pow5(lo) == v {
		return u128{lo, 0}, true
	}
	return u128{}, false
}

// --- 128-bit integer helpers for the exact rules (Swift uses UInt128) ---

// u128DivRemSmall is c / d and c mod d for a one-word d > 0.
func u128DivRemSmall(c u128, d int64) (u128, int64) {
	qr := divRem_q128r64_128x64(c.dw0, c.dw1, d)
	return u128{qr.quotDw0, qr.quotDw1}, qr.remDw0
}

// u128MulChecked is a·b; ok = false on 128-bit overflow.
func u128MulChecked(a, b u128) (u128, bool) {
	p := mul_256_128x128(a.dw0, a.dw1, b.dw0, b.dw1)
	return u128{p.dw0, p.dw1}, (p.dw2 | p.dw3) == 0
}

// u128TrailingZeros is the number of trailing zero bits of a nonzero c.
func u128TrailingZeros(c u128) int {
	if c.dw0 != 0 {
		return bits.TrailingZeros64(uint64(c.dw0))
	}
	return 64 + bits.TrailingZeros64(uint64(c.dw1))
}

// u128GtSmall is c > v for a one-word v.
func u128GtSmall(c u128, v uint64) bool {
	return (c.dw1 != 0) || (uint64(c.dw0) > v)
}

// stripTens is (c, q) with c free of trailing zeros: the same value c·10^q.
func stripTens(cIn u128, qIn int) (u128, int) {
	c := cIn
	q := qIn
	for !isZero_bool_128(c.dw0, c.dw1) {
		quot, r := u128DivRemSmall(c, 10)
		if r != 0 {
			break
		}
		c = quot
		q++
	}
	return c, q
}

// applyPreferredScale re-expresses a finite-nonzero result with quantum exponent
// preferredQExp when the cohort allows — pad trailing zeros (if there is
// precision headroom) or strip them. A no-op on a full-precision inexact result.
// Value-preserving, raises no flags.
func applyPreferredScale(x d128, preferredQExp int) d128 {
	xQ := d128_qExp(x)
	qDelta := xQ - preferredQExp
	if qDelta == 0 {
		return x
	}
	if qDelta > 0 {
		// Lower the exponent toward preferred by padding trailing zeros.
		digitLen := d128_digitLen(x)
		headroom := min(34-digitLen, xQ-qTinyPow)
		if qDelta <= headroom {
			c := d128_u128(x)
			scaled := mulPow10_128_128(c.dw0, c.dw1, qDelta)
			return decimalFNZ_64x64(d128_signFlag(x), xQ-qDelta, scaled.dw0, scaled.dw1)
		}
		return x
	}
	// qDelta < 0: raise the exponent toward preferred by stripping trailing zeros.
	return d128_stripTrailingZeros(x, -qDelta)
}

// --- integrality predicates ---

// d128_isExactIntegral reports whether x is an exact integer (finite, no
// fractional part).
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
	return qr.residue == RESIDUE_EXACT
}

// d128_isOddIntegral reports whether x is an odd integer. The caller guarantees
// x is an exact integer.
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
