// IEEE 754-2019 §9.2 rootn(x, n) = x^(1/n) for integer n ≠ 0. When |n| is a
// power of two, take √ repeatedly (then reciprocal for n < 0); otherwise
// |x|^(1/n) via pow, restoring the sign for odd n.

package decimal128

import "math/bits"

var d128RootnOne = d128_fromInt64(1)

func d128_rootn_tte(x d128, n int) d128            { return d128_rootn_rnd_ctxnull(x, n, ROUND_TIES_TO_EVEN, nil) }
func d128_rootn_rnd(x d128, n int, rnd round) d128 { return d128_rootn_rnd_ctxnull(x, n, rnd, nil) }
func d128_rootn_ctx(x d128, n int, ctx *context) d128 {
	return d128_rootn_rnd_ctxnull(x, n, ctx.round, ctx)
}

func d128_rootn_rnd_ctxnull(x d128, n int, rounding round, ctxnull *context) d128 {
	if n == 0 {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeRootnBadArgs)
	}
	xSign := d128_signFlag(x)
	nIsOdd := (n & 1) != 0

	if d128_isFIN(x) {
		if d128_isZER(x) {
			// rootn(±0, n): n < 0 → ±∞ (divideByZero); n > 0 → ±0. Sign carries
			// only for odd n.
			resultSign := nIsOdd && xSign
			if n < 0 {
				return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
			}
			return d128_zero(resultSign, 0)
		}
		// Finite non-zero.
		if xSign && !nIsOdd {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeRootnBadArgs) // even root of a negative
		}
		if useD48 { // D48GoWorkOrder.md G-14: Phase 4 keeps this arm only
			return rootnFNZ(x, n, rounding, ctxnull)
		}
		return d38_rootnFNZ(x, n, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	// x = ±∞.
	if xSign && nIsOdd {
		if n < 0 {
			return d128_zero(true, 0)
		}
		return d128_NEG_INFINITY() // -∞, odd n
	}
	if xSign {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeRootnBadArgs) // -∞, even n
	}
	if n < 0 {
		return d128_zero(false, 0)
	}
	return d128_POS_INFINITY() // +∞
}

// d38_rootnFNZ is the D38 magnitude (removed at the D48 switch, Phase 4).
func d38_rootnFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	// |n| as an unsigned magnitude (handles n == math.MinInt32 without overflow).
	var nAbs uint64
	if n < 0 {
		nAbs = -uint64(int64(n))
	} else {
		nAbs = uint64(int64(n))
	}

	if bits.OnesCount64(nAbs) == 1 {
		// |n| is a power of two: x^(1/n) = √ applied `shifts` times. Intermediate
		// roots are quiet/ties-to-even; the last honors the caller's rounding+flags.
		shifts := bits.TrailingZeros64(nAbs)
		z := x
		for i := 0; i < shifts; i++ {
			if i == shifts-1 {
				z = d128_sqrt_rnd_ctxnull(z, rounding, ctxnull)
			} else {
				z = d128_sqrt_tte(z)
			}
		}
		if n < 0 {
			z = d128_divide_rnd_ctxnull(d128RootnOne, z, rounding, ctxnull)
		}
		return z
	}

	// General case: |x|^(1/n), with the sign restored for odd n.
	absX := d128_abs(x)
	nReciprocal := d128_divide_tte(d128RootnOne, d128_fromInt64(n))
	z := d128_pow_rnd_ctxnull(absX, nReciprocal, forMagnitude(rounding, d128_signFlag(x)), ctxnull)
	if d128_signFlag(x) {
		return d128_negate(z)
	}
	return z
}

// --- finite non-zero magnitude, D48 (Swift D128Rootn.swift) ---

func rootnFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	// |n| as an unsigned magnitude (handles n == math.MinInt without overflow).
	nAbs := uint64(n)
	if n < 0 {
		nAbs = -nAbs
	}

	if bits.OnesCount64(nAbs) == 1 {
		// |n| is a power of two: x^(1/n) = √ applied `shifts` times. When that
		// would round more than once (two or more √ steps, or a √ followed by
		// 1/z), run the whole chain in D48 and round once. n = 2 and n = ±1 stay
		// single D128 operations (below). Intermediate roots there are quiet /
		// ties-to-even; the last honors the caller's rounding + flags.
		shifts := bits.TrailingZeros64(nAbs)
		steps := shifts
		if n < 0 {
			steps++
		}
		if steps >= 2 {
			return rootnChain_d48(x, shifts, n < 0, rounding, ctxnull)
		}
		z := x
		for i := 0; i < shifts; i++ {
			if i == shifts-1 {
				z = d128_sqrt_rnd_ctxnull(z, rounding, ctxnull)
			} else {
				z = d128_sqrt_tte(z)
			}
		}
		if n < 0 {
			z = d128_divide_rnd_ctxnull(d128RootnOne, z, rounding, ctxnull)
		}
		return z
	}

	// General case: |x|^(1/n), with the sign restored for odd n.
	absX := d128_abs(x)
	nReciprocal := d128_divide_tte(d128RootnOne, d128_fromInt64(n))
	z := d128_pow_rnd_ctxnull(absX, nReciprocal, forMagnitude(rounding, d128_signFlag(x)), ctxnull)
	if d128_signFlag(x) {
		return d128_negate(z)
	}
	return z
}

// rootnChain_d48 is x^(±1/2^shifts) for x > 0 over D48, rounded once. The
// boundary test gates the exactness check (rootnExact); a result it doesn't
// decide is rounded as is, and is inexact. Error bound: each √ halves the
// error so far and adds ½ ulp (< 1 ulp in total), 1/z adds ½: under 1.5 units;
// 40 leaves wide slack.
func rootnChain_d48(x d128, shifts int, invert bool, rounding round, ctxnull *context) d128 {
	z := d48_fromD128(x, false)
	for j := 0; j < shifts; j++ {
		z = d48_sqrt(z)
	}
	if invert {
		z = d48_divide(d48_fromInt32(1), z)
	}
	if d48_ambiguous(d48_u320(z), d48_qExp(z), u320FromDw0(40)) {
		// Near a boundary. An exact root lands on a 34-digit point: check it exactly.
		candidate := finalizeD48_ctxnull(z, ROUND_TIES_TO_EVEN, nil)
		if exact, ok := rootnExact(x, shifts, invert, candidate, rounding, ctxnull); ok {
			return exact
		}
	}
	return d48_forceInexact(finalizeD48_ctxnull(z, rounding, ctxnull), ctxnull)
}

// twoFive is c = 2^a · 5^b exactly, or ok = false.
func twoFive(cIn u128) (a, b int, ok bool) {
	a = u128TrailingZeros(cIn)
	c := shrU128(cIn, a)
	for {
		q, r := u128DivRemSmall(c, 5)
		if r != 0 {
			break
		}
		c = q
		b++
	}
	return a, b, c == (u128{1, 0})
}

// shlExp is v·2^s for the exponent arithmetic of rootnExact, |v| < 2^14 and
// 0 <= s <= 63: exact while it fits, else ±2^50, which matches no exponent the
// test compares against (|q| < 2^14, k <= 231) (G-9; Swift computes q << s).
func shlExp(v, s int) int {
	if (v == 0) || (s <= 31) {
		return v << s
	}
	if v < 0 {
		return -(1 << 50)
	}
	return 1 << 50
}

// rootnExact: if y = candidate is exactly x^(±1/2^shifts), return it at the
// exponent closest to the preferred one the D128 chain would give (√: ⌊Q/2⌋
// per step; 1/z: −Q(z)). Exact test in integers: y^(2^s) == x, or
// y^(2^s)·x == 1. y^(2^s) == x needs the power to equal x's coefficient
// (< 2^113), so it is computed in 128 bits and abandoned on overflow;
// y^(2^s)·x = 10^k needs both stripped coefficients to be 2^a·5^b, which
// reduces it to exponent arithmetic.
func rootnExact(x d128, shifts int, invert bool, candidate d128, rounding round, ctxnull *context) (d128, bool) {
	if !d128_isFNZ(candidate) {
		return d128{}, false
	}
	ysC, ysQ := stripTens(d128_u128(candidate), d128_qExp(candidate))
	xsC, xsQ := stripTens(d128_u128(x), d128_qExp(x))
	qPow := shlExp(ysQ, shifts)
	if !invert {
		p := ysC
		for j := 0; j < shifts; j++ {
			t, ok := u128MulChecked(p, p)
			if !ok {
				return d128{}, false
			}
			p = t
		}
		if (p != xsC) || (qPow != xsQ) {
			return d128{}, false
		}
	} else {
		// y^(2^s) · x = P·c_x · 10^(qPow + q_x) must equal 1.
		k := -(qPow + xsQ)
		if (k < 0) || (k > 231) {
			return d128{}, false
		}
		fyA, fyB, okY := twoFive(ysC)
		fxA, fxB, okX := twoFive(xsC)
		if !okY || !okX {
			return d128{}, false
		}
		if (shlExp(fyA, shifts)+fxA != k) || (shlExp(fyB, shifts)+fxB != k) {
			return d128{}, false
		}
	}
	preferred := d128_qExp(x)
	for j := 0; j < shifts; j++ {
		preferred >>= 1 // ⌊Q/2⌋ (arithmetic shift)
	}
	if invert {
		preferred = -preferred
	}
	return d48_exactWithPreferred(false, ysC, ysQ, preferred, rounding, ctxnull), true
}
