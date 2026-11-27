// IEEE 754 divide, divideInteger, and remainderTrunc / remainderNear.
// `d128_divide_rnd_ctxnull` is the general divide body; the `_tte`, `_rnd` and `_ctx`
// forms fold in rounding/context. divideInteger and the remainders never round: the
// result is exact or NaN.

package decimal128

import "math/bits"

// d128_divide_tte is the ties-to-even divide without a context, in one frame for a
// one-word divisor (GoUbdKernelGapWorkOrder.md L2); everything else goes to
// divideSlow_tte, which takes a two-word divisor in one more frame. Every other case (a special or zero operand, an exponent that
// leaves the format, a quotient that rounds up to 10^34) goes to
// d128_divide_rnd_ctxnull, which gives the same value there.
//
//   - A power-of-ten divisor moves the point: the dividend, stripped of up to
//     digits(y) - 1 trailing zeros toward the preferred exponent.
//   - Otherwise the dividend is scaled by 10^k so that the quotient has exactly 34
//     digits (k from the digit counts and one branch-free compare), half of the
//     divisor is added, and the division is two steps of two words by one. The
//     rounded quotient is then the truncated one; a remainder equal to the half
//     means the division was exact (the quotient is stripped back toward the
//     preferred exponent), and a zero remainder with an even divisor is a tie
//     (made even).
//   - The two steps divide by 32-bit halves on the hardware divider for a
//     divisor below 2^32, and otherwise by the Moller-Granlund reciprocal
//     (vReciprocal_64_64_mgNewton, no divide): bits.Div64 is a software routine
//     on arm64.
func d128_divide_tte(x, y d128) d128 {
	a, xhi, b := uint64(x.ubdLo64), uint64(d128_hi49(x)), uint64(y.ubdLo64)
	if !bothFinite(x, y) || d128_hi49(y) != 0 || b == 0 || a|xhi == 0 {
		return divideSlow_tte(x, y)
	}
	signBit := (x.ubdHi64 ^ y.ubdHi64) & (1 << 49) // the quotient's sign, at bit 49
	exp := d128_qExp(x) - d128_qExp(y)             // the preferred exponent
	db := digitLen_word(b)

	if b == uint64(pow10_128_dw0(db-1)) {
		// A power of ten.
		q0, q1, e := a, xhi, exp-(db-1)
		if db > 1 {
			ss := strip_s128sc_128(int64(a), int64(xhi), db-1)
			q0, q1, e = uint64(ss.strippedDw0), uint64(ss.strippedDw1), e+int(ss.stripCount)
		}
		if e < -6176 || e > 6111 {
			return d128_divide_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
		}
		return d128{ubdLo64: int64(q0), ubdHi64: int64(e)<<50 | signBit | int64(q1)}
	}

	// An exact quotient of one-word operands, without the long division
	// (GoUbdKernelGapWorkOrder.md L3). With b = 2^tb * 5^fb * g, b divides
	// a * 10^j for some j exactly when g divides a; the least such j is then
	// max(tb - twos(a), fb - fives(a), 0), and a * 10^j / b is the result at j
	// places below the preferred exponent, already stripped. An inexact division
	// leaves at the test of g.
	if xhi == 0 {
		tb := bits.TrailingZeros64(b)
		fb, g := fives_64(b >> uint(tb))
		if a%g == 0 {
			fa := fivesUpTo_128(a, 0, fb)
			j := max(tb-bits.TrailingZeros64(a), fb-fa, 0)
			if j <= 19 {
				if hi, lo := bits.Mul64(a, uint64(pow10_128_dw0(j))); hi == 0 {
					if e := exp - j; e >= -6176 && e <= 6111 {
						return d128{ubdLo64: int64(lo / b), ubdHi64: int64(e)<<50 | signBit}
					}
				}
			}
		}
	}

	// The dividend scaled by 10^k, three words, with k such that the quotient has
	// 34 digits: lt is 1 when the dividend's leading digits are below the
	// divisor's once both are aligned.
	var n0, n1, n2 uint64
	var k int
	if xhi == 0 {
		// One-word dividend: whichever coefficient is shorter is scaled by the
		// difference in digits, the other by 10^0.
		e := db - digitLen_word(a)
		ea, eb := e, -e
		if ea < 0 {
			ea = 0
		}
		if eb < 0 {
			eb = 0
		}
		ah, al := bits.Mul64(a, uint64(pow10_128_dw0(ea)))
		bh, bl := bits.Mul64(b, uint64(pow10_128_dw0(eb)))
		_, lt := bits.Sub64(al, bl, 0)
		_, lt = bits.Sub64(ah, bh, lt)
		k = 33 + e + int(lt)
		m, p := a, k
		if k > 38 {
			// Past the last 128-bit power: a has at most digits(b) - 4 digits
			// then, and a * 10^(k - 38) fits one word.
			m, p = a*uint64(pow10_128_dw0(k-38)), 38
		}
		t0, t1 := uint64(pow10_128_dw0(p)), uint64(pow10_128_dw1(p))
		var h0, l1, c uint64
		h0, n0 = bits.Mul64(m, t0)
		n2, l1 = bits.Mul64(m, t1)
		n1, c = bits.Add64(h0, l1, 0)
		n2 += c
	} else {
		// Two-word dividend: it is the longer by e >= 0 digits, and the divisor
		// scaled by 10^e still fits two words.
		e := digitLen_int_128(int64(a), int64(xhi)) - db
		t0, t1 := uint64(pow10_128_dw0(e)), uint64(pow10_128_dw1(e))
		bh, bl := bits.Mul64(b, t0)
		bh += b * t1
		_, lt := bits.Sub64(a, bl, 0)
		_, lt = bits.Sub64(xhi, bh, lt)
		k = 33 - e + int(lt)
		t0, t1 = uint64(pow10_128_dw0(k)), uint64(pow10_128_dw1(k))
		h00, l00 := bits.Mul64(a, t0)
		h01, l01 := bits.Mul64(a, t1)
		h10, l10 := bits.Mul64(xhi, t0)
		var c1, c2 uint64
		n0 = l00
		n1, c1 = bits.Add64(h00, l01, 0)
		n1, c2 = bits.Add64(n1, l10, 0)
		n2 = h01 + h10 + xhi*t1 + c1 + c2
	}
	half := b >> 1
	var c uint64
	n0, c = bits.Add64(n0, half, 0)
	n1, c = bits.Add64(n1, 0, c)
	n2 += c

	// The quotient is below 2^113, so n2 < b: each step's quotient is one word.
	var q1, q0, rem uint64
	if b>>32 == 0 {
		// 32 bits at a time: each partial dividend is below b * 2^32.
		w := n2<<32 | n1>>32
		qa := w / b
		w = (w-qa*b)<<32 | n1&(1<<32-1)
		qb := w / b
		w = (w-qb*b)<<32 | n0>>32
		q1 = qa<<32 | qb
		qa = w / b
		w = (w-qa*b)<<32 | n0&(1<<32-1)
		qb = w / b
		q0 = qa<<32 | qb
		rem = w - qb*b
	} else {
		s := uint(bits.LeadingZeros64(b))
		d := b << s
		v := vReciprocal_64_64_mgNewton(d)
		u2 := n2<<s | n1>>(64-s)
		u1 := n1<<s | n0>>(64-s)
		st := divRem_q64r64_128x64x64_v_mg(int64(u2), int64(u1), int64(d), int64(v))
		q1 = uint64(st.quotDw0)
		st = divRem_q64r64_128x64x64_v_mg(st.remDw0, int64(n0<<s), int64(d), int64(v))
		q0, rem = uint64(st.quotDw0), uint64(st.remDw0)>>s
	}

	e := exp - k
	if rem == half {
		// Exact: the zeros the scale put in go, as far as the preferred exponent.
		// q * b = x * 10^k, so q has k - j trailing zeros to strip, with j the
		// least power for which b divides x * 10^j (see the one-word test above):
		// one exact division by 10^(k - j).
		tb := bits.TrailingZeros64(b)
		fb, _ := fives_64(b >> uint(tb))
		tx := bits.TrailingZeros64(a)
		if a == 0 {
			tx = 64 + bits.TrailingZeros64(xhi)
		}
		j := max(tb-tx, fb-fivesUpTo_128(a, xhi, fb), 0)
		q0, q1 = divExactPow10_128(q0, q1, k-j)
		e = exp - j
	} else if rem == 0 && b&1 == 0 {
		q0 &^= 1 // a tie: the even neighbour
	}
	if q1 == uint64(pow10_128_dw1(34)) && q0 == uint64(pow10_128_dw0(34)) || e < -6176 || e > 6111 {
		return d128_divide_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
	}
	return d128{ubdLo64: int64(q0), ubdHi64: int64(e)<<50 | signBit | int64(q1)}
}

// divideSlow_tte is what d128_divide_tte's one-word path does not take. A two-word
// divisor (20 to 34 digits) with finite nonzero operands has the same scale, bias and reading of the remainder as the
// one-word path, with the division as two Moller-Granlund 3-by-2 steps (Alg. 5),
// written in: as calls they pass the quotient and remainder words through memory.
// Everything else goes to d128_divide_rnd_ctxnull. It is a function of its own,
// reached from the one-word path's fallback, so that the one-word path is unchanged:
// written into d128_divide_tte, the two-word path slowed the one-word bands (P-fin
// WD 13.9 -> 15.0 ns, PT 2.48 -> 2.70).
func divideSlow_tte(x, y d128) d128 {
	a, xhi, b, yhi := uint64(x.ubdLo64), uint64(d128_hi49(x)), uint64(y.ubdLo64), uint64(d128_hi49(y))
	if !bothFinite(x, y) || yhi == 0 || a|xhi == 0 {
		return d128_divide_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
	}
	signBit := (x.ubdHi64 ^ y.ubdHi64) & (1 << 49)
	exp := d128_qExp(x) - d128_qExp(y)
	// The digit counts of x and y (digitLen_u128, written in: Go does not
	// inline it here).
	nx, ny := bits.Len64(a), 64+bits.Len64(yhi)
	if xhi != 0 {
		nx = 64 + bits.Len64(xhi)
	}
	tx, ty := (nx*1233)>>12, (ny*1233)>>12
	_, bx := bits.Sub64(a, uint64(pow10_128_dw0(tx)), 0)
	_, bx = bits.Sub64(xhi, uint64(pow10_128_dw1(tx)), bx)
	_, by := bits.Sub64(b, uint64(pow10_128_dw0(ty)), 0)
	_, by = bits.Sub64(yhi, uint64(pow10_128_dw1(ty)), by)
	dx, dy := tx+1-int(bx), ty+1-int(by)
	// lt is 1 when x's leading digits are below y's once aligned: the
	// shorter coefficient is scaled by the difference, the other by 10^0.
	e := dy - dx
	ex, ey := e, -e
	if ex < 0 {
		ex = 0
	}
	if ey < 0 {
		ey = 0
	}
	p0, p1 := uint64(pow10_128_dw0(ex)), uint64(pow10_128_dw1(ex))
	xh, xl := bits.Mul64(a, p0)
	xh += a*p1 + xhi*p0
	p0, p1 = uint64(pow10_128_dw0(ey)), uint64(pow10_128_dw1(ey))
	yh, yl := bits.Mul64(b, p0)
	yh += b*p1 + yhi*p0
	_, lt := bits.Sub64(xl, yl, 0)
	_, lt = bits.Sub64(xh, yh, lt)
	k := 33 + e + int(lt)

	// n = x * 10^k, four words. Past the last 128-bit power, x takes the
	// excess first: it has at most dy - 4 digits then, so the product
	// still fits two words.
	ml, mh, kt := a, xhi, k
	if k > 38 {
		p0, p1 = uint64(pow10_128_dw0(k-38)), uint64(pow10_128_dw1(k-38))
		mh, ml = bits.Mul64(a, p0)
		mh += a*p1 + xhi*p0
		kt = 38
	}
	t0, t1 := uint64(pow10_128_dw0(kt)), uint64(pow10_128_dw1(kt))
	h00, n0 := bits.Mul64(ml, t0)
	h01, l01 := bits.Mul64(ml, t1)
	h10, l10 := bits.Mul64(mh, t0)
	h11, l11 := bits.Mul64(mh, t1)
	n1, c1 := bits.Add64(h00, l01, 0)
	n1, c2 := bits.Add64(n1, l10, 0)
	n2, c1 := bits.Add64(h01, h10, c1)
	n2, c2 = bits.Add64(n2, l11, c2)
	n3 := h11 + c1 + c2

	// Biased by half the divisor.
	hl, hh := b>>1|yhi<<63, yhi>>1
	var c uint64
	n0, c = bits.Add64(n0, hl, 0)
	n1, c = bits.Add64(n1, hh, c)
	n2, c = bits.Add64(n2, 0, c)
	n3 += c

	// Normalized: the divisor's top bit set (s >= 15, since yhi < 2^49),
	// the dividend shifted with it. The quotient is below 2^113, so the
	// first step's quotient word is below 2^49 and its precondition
	// (u2:u1) < (v1:v0) holds.
	s := uint(bits.LeadingZeros64(yhi))
	rs := 64 - s
	v1, v0 := yhi<<s|b>>rs, b<<s
	vv := vReciprocal_64_128_mgNewton(v1, v0)
	var q1, q0, r1, r0 uint64
	u2, u1, u0 := n3<<s|n2>>rs, n2<<s|n1>>rs, n1<<s|n0>>rs
	for step := 0; step < 2; step++ {
		// (q : qf) = vv * u2 + (u2 : u1); the remainder against q + 1,
		// corrected once (q) and, rarely, once more.
		q, qf := bits.Mul64(vv, u2)
		qf, c = bits.Add64(qf, u1, 0)
		q += u2 + c
		th, tl := bits.Mul64(q, v0)
		th += q * v1
		r0, c = bits.Sub64(u0, tl, 0)
		r1, _ = bits.Sub64(u1, th, c)
		r0, c = bits.Sub64(r0, v0, 0)
		r1, _ = bits.Sub64(r1, v1, c)
		q++
		if r1 >= qf {
			q--
			r0, c = bits.Add64(r0, v0, 0)
			r1, _ = bits.Add64(r1, v1, c)
		}
		if r1 > v1 || r1 == v1 && r0 >= v0 {
			q++
			r0, c = bits.Sub64(r0, v0, 0)
			r1, _ = bits.Sub64(r1, v1, c)
		}
		if step == 0 {
			q1, u2, u1, u0 = q, r1, r0, n0<<s
		} else {
			q0 = q
		}
	}
	remLo, remHi := r0>>s|r1<<rs, r1>>s
	e2 := exp - k
	if remLo == hl && remHi == hh {
		// Exact: q * y = x * 10^k, so q has k - j trailing zeros to
		// strip, j the least power for which y divides x * 10^j.
		ty := bits.TrailingZeros64(b)
		if b == 0 {
			ty = 64 + bits.TrailingZeros64(yhi)
		}
		tx := bits.TrailingZeros64(a)
		if a == 0 {
			tx = 64 + bits.TrailingZeros64(xhi)
		}
		yol, yoh := b>>uint(ty)|yhi<<(64-uint(ty)), yhi>>uint(ty)
		if ty >= 64 {
			yol, yoh = yhi>>uint(ty-64), 0
		}
		fy := fivesUpTo_128(yol, yoh, 64)
		j := max(ty-tx, fy-fivesUpTo_128(a, xhi, fy), 0)
		q0, q1 = divExactPow10_128(q0, q1, k-j)
		e2 = exp - j
	} else if remLo|remHi == 0 && b&1 == 0 {
		q0 &^= 1 // a tie: the even neighbour
	}
	if q1 == uint64(pow10_128_dw1(34)) && q0 == uint64(pow10_128_dw0(34)) || e2 < -6176 || e2 > 6111 {
		return d128_divide_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
	}
	return d128{ubdLo64: int64(q0), ubdHi64: int64(e2)<<50 | signBit | int64(q1)}
}

// digitLen_word is the number of decimal digits of c (0 for 0): the estimate from
// the bit length is the count or one short, and it is short exactly when
// c - 10^t does not borrow.
func digitLen_word(c uint64) int {
	t := (bits.Len64(c) * 1233) >> 12
	_, b := bits.Sub64(c, uint64(pow10_128_dw0(t)), 0)
	return t + 1 - int(b)
}
func d128_divide_rnd(x, y d128, rnd round) d128 { return d128_divide_rnd_ctxnull(x, y, rnd, nil) }
func d128_divide_ctx(x, y d128, ctx *context) d128 {
	return d128_divide_rnd_ctxnull(x, y, ctx.round, ctx)
}

// divideInteger never rounds: trunc(x/y) is exact (or NaN/Inf).

// divideInteger never rounds -- `trunc(x/y)` is exact (or NaN/Inf), so there
// is no `_rnd`/`_tte` rung. `_ctxnull` is the general body; `_ctx` forwards
// with a flag sink, and the bare form forwards with none. Mirrors remainder.
func d128_divideInteger(x, y d128) d128 { return d128_divideInteger_ctxnull(x, y, nil) }
func d128_divideInteger_ctx(x, y d128, ctx *context) d128 {
	return d128_divideInteger_ctxnull(x, y, ctx)
}

func d128_divide_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	resultSign := d128_signFlag(x) != d128_signFlag(y)
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return divFnzFnz_rnd_ctx(x, y, resultSign, rounding, ctxnull)
		}
		if d128_isZER(x) && d128_isZER(y) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivZeroByZero, const_D128_QNAN0)
		}
		if d128_isZER(x) {
			return d128_zero(resultSign, d128_qExp(x)-d128_qExp(y))
		}
		return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if d128_isINF(x) && d128_isINF(y) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivInfByInf, const_D128_QNAN0)
	}
	if d128_isINF(y) {
		return d128_zero(resultSign, -6176)
	}
	return d128_infinity(resultSign)
}

// Remainder never rounds: |r| < |y| (trunc) or |r| <= |y|/2 (near), exact or NaN.

func d128_remainderTrunc(x, y d128) d128 { return d128_remainderTrunc_ctxnull(x, y, nil) }
func d128_remainderTrunc_ctx(x, y d128, ctx *context) d128 {
	return d128_remainderTrunc_ctxnull(x, y, ctx)
}

func d128_remainderTrunc_ctxnull(x, y d128, ctxnull *context) d128 {
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return remainderTruncFnzFnz(x, y)
		}
		if d128_isZER(x) && d128_isNotZER(y) {
			return d128_zero(d128_signFlag(x), min(d128_qExp(x), d128_qExp(y)))
		}
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivByZeroInRemainderOp, const_D128_QNAN0)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	if d128_isINF(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeInfNumeratorInRemainderOp, const_D128_QNAN0)
	}
	return x
}

func d128_remainderNear(x, y d128) d128 { return d128_remainderNear_ctxnull(x, y, nil) }
func d128_remainderNear_ctx(x, y d128, ctx *context) d128 {
	return d128_remainderNear_ctxnull(x, y, ctx)
}

func d128_remainderNear_ctxnull(x, y d128, ctxnull *context) d128 {
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return remainderNearFnzFnz(x, y)
		}
		if d128_isZER(x) && d128_isNotZER(y) {
			return d128_zero(d128_signFlag(x), min(d128_qExp(x), d128_qExp(y)))
		}
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivByZeroInRemainderOp, const_D128_QNAN0)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	if d128_isINF(x) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeInfNumeratorInRemainderOp, const_D128_QNAN0)
	}
	return x
}

func d128_divideInteger_ctxnull(x, y d128, ctxnull *context) d128 {
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return divIntFnzFnz_rnd_ctx(x, y, ctxnull)
		}
		resultSign := d128_signFlag(x) != d128_signFlag(y)
		if d128_isZER(x) && d128_isZER(y) {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivZeroByZero, const_D128_QNAN0)
		}
		if d128_isZER(x) {
			return d128_zero(resultSign, 0)
		}
		return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	resultSign := d128_signFlag(x) != d128_signFlag(y)
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if d128_isINF(x) && d128_isINF(y) {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivInfByInf, const_D128_QNAN0)
	}
	if d128_isINF(x) {
		return d128_infinity(resultSign)
	}
	return d128_zero(resultSign, 0)
}

func divFnzFnz_rnd_ctx(x, y d128, resultSign bool, rounding round, ctxnull *context) d128 {
	// Power-of-ten divisor fast path (PT). When y is a power of ten,
	// x / y = Cx * 10^(qExp(x) - qExp(y) - m) is exact and coefficient-preserving
	// (m = digitLen(y) - 1), so skip the dividend scale + 256/128 divide and reuse
	// the exact trailing-zero strip + finalize. "Divisor coefficient is a power of ten"
	// is cohort-invariant, so one dword compare detects every representation of y.
	m := d128_digitLen(y) - 1
	yCoeff := d128_u128(y)
	qPreferred := d128_qExp(x) - d128_qExp(y)
	if yCoeff.dw0 == pow10_128_dw0(m) && yCoeff.dw1 == pow10_128_dw1(m) {
		return divPow10Divisor_rnd_ctx(x, resultSign, qPreferred, m, rounding, ctxnull)
	}

	// CD-width divisor (digitLen(y) <= 4, so m <= 3). Two sub-paths:
	//   (a) integer-exact: xCoeff / yCoeff has remainder 0, so qI already sits at the preferred
	//       exponent as the canonical cohort member -- return it directly, no scale and
	//       no trailing-zero strip. This is the ET band: one 128/64 divide.
	//   (b) inexact: the scaled dividend is exactly 34 + digitLen(y) <= 38 digits, so it
	//       fits 128 bits, and the divisor fits 64 bits -- the whole correctly-rounded
	//       result is a single 128/64 on the scaled numerator. This is the WD/XD
	//       scaled-numerator path (below) narrowed to 128-bit width; it stays 128 bits
	//       throughout rather than widening the dividend to 256 for a full-width divide.
	if m <= 3 {
		xCoeff := d128_u128(x)
		qint := divRem_q128r64_128x64(xCoeff.dw0, xCoeff.dw1, yCoeff.dw0)
		if qint.remDw0 == 0 && !isZero_bool_128(qint.quotDw0, qint.quotDw1) {
			return finalize_128_ctxnull(resultSign, qPreferred,
				fromLoHi_128_64x64(qint.quotDw0, qint.quotDw1), const_RESIDUE_EXACT, rounding, ctxnull)
		}
		digitLenDeltaCD := d128_digitLen(x) - (m + 1)
		numeratorScaleCD := 34 - digitLenDeltaCD
		verify("", func() bool { return numeratorScaleCD > 0 })
		scaledNumCD := mulPow10_128_128(x.ubdLo64, d128_hi49(x), numeratorScaleCD)
		qr := divRem_q128r64_128x64(scaledNumCD.dw0, scaledNumCD.dw1, yCoeff.dw0)
		residueCD := residueFromRemainderDivisor_64(qr.remDw0, yCoeff.dw0)
		qZcd := qPreferred - numeratorScaleCD
		coefficientCD := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		if residueCD == const_RESIDUE_EXACT && qZcd < qPreferred {
			ss := strip_s128sc_128(coefficientCD.dw0, coefficientCD.dw1, qPreferred-qZcd)
			coefficientCD = u128{ss.strippedDw0, ss.strippedDw1}
			qZcd += int(ss.stripCount)
		}
		return finalize_128_ctxnull(resultSign, qZcd, coefficientCD, residueCD, rounding, ctxnull)
	}

	digitLenDelta := d128_digitLen(x) - (m + 1)
	numeratorScale := 34 - digitLenDelta
	verify("", func() bool { return numeratorScale > 0 })

	scaledNumerator := mulPow10_256_128(x.ubdLo64, d128_hi49(x), numeratorScale)
	qr := u256DivResidue_128(scaledNumerator, yCoeff)
	coefficient := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	res := qr.residue

	qZ := qPreferred - numeratorScale

	if res == const_RESIDUE_EXACT && qZ < qPreferred {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, qPreferred-qZ)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctxnull(resultSign, qZ, coefficient, res, rounding, ctxnull)
}

// PT fast path: divide by a power of ten (value 10^(m + qExp(y))).
// The result is exact and keeps the dividend's coefficient: `Cx * 10^(qPreferred - m)`,
// where `qPreferred = qExp(x) - qExp(y)`. As in the general exact case, the natural
// exponent sits `m` below the preferred exponent, so strip up to `m` trailing zeros
// from `Cx` back toward it (no-op when `Cx` has none), then let `finalize` apply the
// usual range/clamp/subnormal handling. Never rounds.
func divPow10Divisor_rnd_ctx(x d128, resultSign bool, qPreferred, m int, rounding round, ctxnull *context) d128 {
	qZ := qPreferred - m
	coefficient := d128_u128(x)
	if m > 0 {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, m)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctxnull(resultSign, qZ, coefficient, const_RESIDUE_EXACT, rounding, ctxnull)
}

func divIntFnzFnz_rnd_ctx(x, y d128, ctxnull *context) d128 {
	q1 := d128_divide_rnd_ctxnull(x, y, const_ROUND_TOWARD_ZERO, nil) // quiet
	verify("", func() bool { return d128_isFIN(q1) })
	q2 := d128_roundToIntegralTowardZeroSignalInexact_ctxnull(q1, nil)
	verify("", func() bool { return d128_isFIN(q2) })
	q2Q := d128_qExp(q2)
	verify("", func() bool { return q2Q >= 0 })
	if q2Q == 0 {
		return q2
	}
	if d128_isZER(q2) {
		return d128_zero(d128_signFlag(q2), 0)
	}
	zQ := q2Q
	headroom := 34 - d128_digitLen(q2)
	if headroom < zQ {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeDivIntOverflowsCoefficient, const_D128_QNAN0)
	}
	sign := d128_signFlag(q2)
	scaled := mulPow10_128_128(q2.ubdLo64, d128_hi49(q2), zQ)
	return decimalFNZ_128(sign, 0, scaled.dw0, scaled.dw1)
}

// RemainderExactWorkOrder.md §2. Both remainders are exact for every finite
// non-zero x and y, however large the quotient x/y (IEEE 754-2019 §5.3.1).
// Everything happens on coefficients at the common exponent
// e = min(qExp(x), qExp(y)):
//
//	X = |x|'s coefficient at e,  h = |y|'s coefficient at e,  r2 = X mod 2h
//
// r2 gives both the truncated remainder (r2 or r2 - h) and the parity of the
// truncated quotient (odd iff r2 >= h), which remainderNear needs for ties.
// When qExp(x) > qExp(y), X = xCoeff * 10^k is never formed: r2 = xCoeff * (10^k mod 2h)
// mod 2h. No quotient is ever formed, and the result's coefficient is below h
// (or below xCoeff when qExp(x) < qExp(y)), so it always fits 34 digits at e.
// Signs are applied last: the result takes the sign of x, or its negation
// when remainderNear rounds the quotient away from zero.

// remQuotientIsZero: whether qExp(x) < qExp(y) and the digit lengths alone show
// |y| > 2|x|: both quotients are 0 and both remainders are x itself.
func remQuotientIsZero(x, y d128) bool {
	d := d128_qExp(y) - d128_qExp(x)
	return (d > 0) && (d128_digitLen(y)+d > d128_digitLen(x)+1)
}

// h: |y|'s coefficient at e = min(qExp(x), qExp(y)). Below 10^35 once
// `remQuotientIsZero` has been ruled out.
func remDivisorAtMinExp(x, y d128) u128 {
	d := d128_qExp(y) - d128_qExp(x)
	if d <= 0 {
		return d128_u128(y)
	}
	verify("", func() bool { return d <= 34 })
	return mulPow10_128_128(y.ubdLo64, d128_hi49(y), d)
}

// a mod m.
func remMod(a, m u128) u128 {
	qr := u256DivRem_128(u256{a.dw0, a.dw1, 0, 0}, m)
	return u128{qr.remDw0, qr.remDw1}
}

// remMulMod: (a * b) mod m. The caller guarantees a * b / m < 2^128 (the
// u256DivRem_128 contract): here always a < 2^128 and b < m, or a, b < m.
func remMulMod(a, b, m u128) u128 {
	p := mul_256_128x128(a.dw0, a.dw1, b.dw0, b.dw1)
	qr := u256DivRem_128(p, m)
	return u128{qr.remDw0, qr.remDw1}
}

// 10^k mod m, k in 1...12287, m >= 2. k <= 38 reads the table; larger k
// runs left-to-right over 5-bit windows of k: acc <- acc^32 * 10^w mod m.
// Each squaring has acc < m, and each window multiply has 10^w < 2^103, so
// every quotient stays below 2^128.
func remPow10Mod(k int, m u128) u128 {
	if k <= 38 {
		return remMod(pow10U128(k), m)
	}
	shift := 0
	for (k >> shift) > 31 {
		shift += 5
	}
	acc := remMod(pow10U128(k>>shift), m)
	for shift > 0 {
		shift -= 5
		i := 0
		for i < 5 {
			acc = remMulMod(acc, acc, m)
			i += 1
		}
		w := (k >> shift) & 31
		if w != 0 {
			acc = remMulMod(pow10U128(w), acc, m)
		}
	}
	return acc
}

// r2 = X mod 2h, X = |x|'s coefficient at e = min(qExp(x), qExp(y)).
func remCoeffMod2H(x, y d128, h u128) u128 {
	m := shl_128_128(h.dw0, h.dw1, 1)
	k := d128_qExp(x) - d128_qExp(y)
	if k <= 0 {
		return remMod(d128_u128(x), m)
	}
	return remMulMod(d128_u128(x), remPow10Mod(k, m), m)
}

// remResult: the finite result sign * r * 10^e; a zero r gives a zero of that sign.
func remResult(sign bool, e int, r u128) d128 {
	if isZero_bool_128(r.dw0, r.dw1) {
		return d128_zero(sign, e)
	}
	return decimalFNZ_128(sign, e, r.dw0, r.dw1)
}

// x - trunc(x/y) * y, exact.
func remainderTruncFnzFnz(x, y d128) d128 {
	if remQuotientIsZero(x, y) {
		return x
	}
	h := remDivisorAtMinExp(x, y)
	r2 := remCoeffMod2H(x, y, h)
	e := min(d128_qExp(x), d128_qExp(y))
	if cmp_int_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1) >= 0 {
		return remResult(d128_signFlag(x), e, sub_128_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1))
	}
	return remResult(d128_signFlag(x), e, r2)
}

// x - n * y, n = x/y rounded to the nearest integer, ties to even; exact.
func remainderNearFnzFnz(x, y d128) d128 {
	if remQuotientIsZero(x, y) {
		return x
	}
	h := remDivisorAtMinExp(x, y)
	r2 := remCoeffMod2H(x, y, h)
	e := min(d128_qExp(x), d128_qExp(y))
	truncIsOdd := (cmp_int_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1) >= 0)
	r := r2
	if truncIsOdd {
		r = sub_128_128x128(r2.dw0, r2.dw1, h.dw0, h.dw1)
	}
	// Round the quotient away from zero when r > h/2, or r == h/2 and the
	// truncated quotient is odd: the remainder becomes -(h - r), 0 < h - r < h.
	twoR := shl_128_128(r.dw0, r.dw1, 1)
	cmp := cmp_int_128x128(twoR.dw0, twoR.dw1, h.dw0, h.dw1)
	if (cmp > 0) || ((cmp == 0) && truncIsOdd) {
		sign := !d128_signFlag(x)
		hr := sub_128_128x128(h.dw0, h.dw1, r.dw0, r.dw1)
		return decimalFNZ_128(sign, e, hr.dw0, hr.dw1)
	}
	return remResult(d128_signFlag(x), e, r)
}

// fives_64 returns how many times 5 divides n (n != 0) and n without those
// factors. n is a multiple of 5 exactly when n times the inverse of 5 modulo 2^64
// is at most floor((2^64 - 1) / 5), and that product is then n / 5.
func fives_64(n uint64) (int, uint64) {
	c := 0
	for {
		m := n * 0xCCCCCCCCCCCCCCCD
		if m > 0x3333333333333333 {
			return c, n
		}
		n, c = m, c+1
	}
}

// fivesUpTo_128 returns how many times 5 divides (hi:lo), which is not zero, up
// to limit: the 128-bit form of fives_64's test.
func fivesUpTo_128(lo, hi uint64, limit int) int {
	c := 0
	for c < limit {
		// m = (hi:lo) * (2^128 inverse of 5), modulo 2^128.
		mh, ml := bits.Mul64(lo, 0xCCCCCCCCCCCCCCCD)
		mh += lo*0xCCCCCCCCCCCCCCCC + hi*0xCCCCCCCCCCCCCCCD
		if mh > 0x3333333333333333 || mh == 0x3333333333333333 && ml > 0x3333333333333333 {
			return c
		}
		lo, hi, c = ml, mh, c+1
	}
	return c
}

// const_INV5POW_128[(s<<1)+0..1] is the inverse of 5^s modulo 2^128, lo/hi dwords,
// for s in [0, 34]; the rest is padding for the index mask.
var const_INV5POW_128 = [128]int64{
	1, 0, -3689348814741910323, -3689348814741910324,
	-8116567392432202711, 2951479051793528258, 2066035336255469781, 7968993439842526298,
	-3276141747490816367, 5283147502710415582, 6723469279985657373, -2632719314199827207,
	8723391485480952121, -4215892677581875765, -1944670517645719899, -4532527350258285477,
	-4078282918271054303, -906505470051657096, -8194354213138031507, -7559998723494152066,
	5739826786856214345, 5866697884784990233, 1147965357371242869, 8552037206440818693,
	3918941886216158897, -5668290188195656908, -6594909252240588867, 2555690777102778941,
	6059715779035702873, 511138155420555788, 8590640785290961221, -3587121183657799166,
	-1971220657683718079, -717424236731559834, -4083592946278653939, 7235212782137508679,
	-4506067403997641111, -2242306258314408588, 2788135333942382101, -7827158881146702364,
	-3131721747953433903, -8944129405713161120, -8005041979074507427, 5589871748341188422,
	5777689233668919161, -2571374465073672639, -2533810968008126491, -7892972522498555175,
	3182586621140285025, -1578594504499711035, 636517324228057005, -315718900899942207,
	127303464845611401, 7315553849303832205, -3663888121772788043, -2226238044881143883,
	-8111475253838378255, -445247608976228777, -1622295050767675651, 3600299292946664567,
	3364889804588375193, -6658637770894487733, 8051675590401495685, -1331727554178897547,
	1610335118080299137, 7112352118648041137, -7056630605867760819, 5111819238471518550,
	-5100674935915462487, -6356333781789516937, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
}

// divExactPow10_128 returns (hi:lo) / 10^s for a quotient known to be exact, s in
// [0, 34]: the factor 2^s is a shift, and the factor 5^s a multiply by its inverse
// modulo 2^128.
func divExactPow10_128(lo, hi uint64, s int) (uint64, uint64) {
	sh := uint(s)
	lo = lo>>sh | hi<<(64-sh)
	hi >>= sh
	i0 := uint64(const_INV5POW_128[(s<<1)&127])
	i1 := uint64(const_INV5POW_128[((s<<1)|1)&127])
	qh, ql := bits.Mul64(lo, i0)
	qh += lo*i1 + hi*i0
	return ql, qh
}
