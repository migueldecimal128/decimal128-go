// IEEE 754 add. `d128_add_rnd_ctxnull` is the general body; the `_tte` (bare), `_rnd`,
// and `_ctx` forms are thin specializations of it (rounding/ctx folded in). Its fast
// entry returns directly, with no rounding and no flags, for the same sign, the same
// exponent and coefficients that each fit in 64 bits; every other case tail-calls
// the out-of-line `d128_add_slow_rnd_ctxnull`. Subtract reaches the shared
// `addFnzFnz_rnd_ctxnull` for its aligned magnitude-add subcase.

package decimal128

import "math/bits"

// d128_add_tte is the ties-to-even add without a context, in one frame
// (GoUbdKernelGapWorkOrder.md L4): the word paths below, then the general slow
// entry. d128_subtract_tte holds the same body with y's sign flipped; Go does not
// inline it as a helper, and a second frame costs more than the paths save.
//
// The word paths take two finite operands with one-word coefficients (their high
// words hold no coefficient bits, which also excludes the specials): one exponent
// (a sum, or a difference chosen by a select), or exponents at most 19 apart. There
// the operand with the larger exponent is scaled by 10^gap, one 64x64 multiply,
// and the other added or subtracted: no magnitude comparison and no digit count.
// The result is at the smaller exponent, the preferred one, so it is exact; one
// that needs 35 digits or more (it would round) goes to the slow entry. An exact
// cancellation is +0, as rounding ties to even gives. Then coefficients of up to
// two words whose exponents are close enough that the scaled operand still has at
// most 34 digits: the same arithmetic in two words, also exact.
func d128_add_tte(x, y d128) d128 {
	xh, yh := x.ubdHi64, y.ubdHi64
	if ((xh ^ yh) | (xh << 15)) == 0 {
		// One exponent and one sign, coefficients of one word: the sum and its
		// carry into x's high word.
		lo, c := bits.Add64(uint64(x.ubdLo64), uint64(y.ubdLo64), 0)
		return d128{ubdLo64: int64(lo), ubdHi64: xh + int64(c)}
	}
	if ((xh | yh) << 15) == 0 {
		// Both coefficients one word (which excludes the specials).
		if (xh^yh)>>50 == 0 {
			// One exponent, opposite signs: the difference that did not
			// borrow, chosen by a select (either way about half the time).
			// An exact zero is +0.
			d, bw := bits.Sub64(uint64(x.ubdLo64), uint64(y.ubdLo64), 0)
			if bw != 0 {
				d = uint64(y.ubdLo64) - uint64(x.ubdLo64)
			}
			sign := (xh ^ int64(bw)<<49) & (1 << 49)
			if d == 0 {
				sign = 0
			}
			return d128{ubdLo64: int64(d), ubdHi64: xh&^(1<<49) | sign}
		}
		// a has the larger exponent (the high words order by it), b the other,
		// whose exponent is the result's. Which one it is goes with the kind
		// of data, so it is a branch.
		al, ah, bl, bh := uint64(x.ubdLo64), xh, uint64(y.ubdLo64), yh
		if xh < yh {
			al, ah, bl, bh = bl, bh, al, ah
		}
		if gap := uint64(ah>>50 - bh>>50); gap <= 19 {
			hi, lo := bits.Mul64(al, uint64(pow10_128_dw0(int(gap))))
			sign := ah & (1 << 49)
			if (ah^bh)&(1<<49) == 0 {
				var c uint64
				lo, c = bits.Add64(lo, bl, 0)
				hi += c
			} else {
				// a * 10^gap - b is negative only when a * 10^gap is below
				// 2^64: then b's magnitude, the larger, less it, by a select.
				d, b := bits.Sub64(lo, bl, 0)
				var bw uint64
				hi, bw = bits.Sub64(hi, 0, b)
				lo = d
				if bw != 0 {
					lo, hi, sign = -d, 0, sign^(1<<49)
				}
				if hi|lo == 0 {
					sign = 0 // an exact cancellation: +0
				}
			}
			// Below 10^34: the borrow of the subtraction.
			_, bw := bits.Sub64(lo, uint64(pow10_128_dw0(34)), 0)
			_, bw = bits.Sub64(hi, uint64(pow10_128_dw1(34)), bw)
			if bw != 0 {
				return d128{ubdLo64: int64(lo), ubdHi64: bh>>50<<50 | sign | int64(hi)}
			}
		}
	}
	if bothFinite(x, y) {
		// Coefficients of up to two words, exponents close enough that the one
		// with the larger exponent, scaled to the other's, still has at most 34
		// digits: the sum or difference is exact at the smaller exponent, the
		// preferred one, unless it reaches 10^34.
		if (xh^yh)>>49 == 0 {
			// One exponent and one sign: the two-word sum, exact below 10^34.
			lo, c := bits.Add64(uint64(x.ubdLo64), uint64(y.ubdLo64), 0)
			hi := uint64(xh)&(1<<49-1) + uint64(yh)&(1<<49-1) + c
			_, bw := bits.Sub64(lo, uint64(pow10_128_dw0(34)), 0)
			_, bw = bits.Sub64(hi, uint64(pow10_128_dw1(34)), bw)
			if bw != 0 {
				return d128{ubdLo64: int64(lo), ubdHi64: xh>>49<<49 | int64(hi)}
			}
		}
		al, ah, bl, bh := uint64(x.ubdLo64), xh, uint64(y.ubdLo64), yh
		if xh < yh {
			al, ah, bl, bh = bl, bh, al, ah
		}
		a1, b1 := uint64(ah)&(1<<49-1), uint64(bh)&(1<<49-1)
		gap := int(ah>>50 - bh>>50)
		da := 0 // a's digits; a gap of 0 always fits, so it is not counted then
		if gap != 0 {
			// digitLen_u128(al, a1), written in: Go does not inline it here.
			n := bits.Len64(al)
			if a1 != 0 {
				n = 64 + bits.Len64(a1)
			}
			t := (n * 1233) >> 12
			_, b := bits.Sub64(al, uint64(pow10_128_dw0(t)), 0)
			_, b = bits.Sub64(a1, uint64(pow10_128_dw1(t)), b)
			da = t + 1 - int(b)
		}
		if gap <= 34-da {
			t0, t1 := uint64(pow10_128_dw0(gap)), uint64(pow10_128_dw1(gap))
			hi, lo := bits.Mul64(al, t0)
			hi += al*t1 + a1*t0
			sign := ah & (1 << 49)
			if (ah^bh)&(1<<49) == 0 {
				var c uint64
				lo, c = bits.Add64(lo, bl, 0)
				hi, _ = bits.Add64(hi, b1, c)
			} else {
				var b uint64
				lo, b = bits.Sub64(lo, bl, 0)
				hi, b = bits.Sub64(hi, b1, b)
				if b != 0 {
					// b's magnitude was the larger: negate, take its sign.
					lo, b = bits.Sub64(0, lo, 0)
					hi, _ = bits.Sub64(0, hi, b)
					sign ^= 1 << 49
				}
				if hi|lo == 0 {
					sign = 0 // an exact cancellation: +0
				}
			}
			_, bw := bits.Sub64(lo, uint64(pow10_128_dw0(34)), 0)
			_, bw = bits.Sub64(hi, uint64(pow10_128_dw1(34)), bw)
			if bw != 0 {
				return d128{ubdLo64: int64(lo), ubdHi64: bh>>50<<50 | sign | int64(hi)}
			}
		} else if da != 0 {
			// The result rounds (GoUbdKernelGapWorkOrder.md L5). a is scaled to
			// 34 digits, p = a * 10^m, and the result is p + or - b / 10^d at
			// the exponent exp = bExp + d. b / 10^d is rounded half up by adding
			// half of 10^d first: q, and the remainder rb of b + half, which is 0
			// exactly at a tie. Past d = 34, b is below half a unit: q = 0 and it
			// only says the result is inexact.
			m := 34 - da
			d := gap - m
			exp := int(bh>>50) + d
			t0, t1 := uint64(pow10_128_dw0(m)), uint64(pow10_128_dw1(m))
			sh, sl := bits.Mul64(al, t0)
			sh += al*t1 + a1*t0
			var q0, q1, rb0, rb1, h0, h1 uint64
			rb0 = 1
			if d <= 34 {
				d0, d1 := uint64(pow10_128_dw0(d)), uint64(pow10_128_dw1(d))
				h0, h1 = d0>>1|d1<<63, d1>>1
				tl, c := bits.Add64(bl, h0, 0)
				th, _ := bits.Add64(b1, h1, c)
				// (th:tl) / 10^d by the reciprocal (finalize_tte.go), written in
				// so that the division is not a second frame.
				rs := uint(const_RECIP10_128_S[d&63])
				rsh := rs & 63
				v0, v1 := tl>>rsh|th<<(64-rsh), th>>rsh
				if rs >= 64 {
					v0, v1 = th>>rsh, 0
				}
				m0 := uint64(const_RECIP10_128_M[(d<<1)&127])
				m1 := uint64(const_RECIP10_128_M[((d<<1)|1)&127])
				g00, _ := bits.Mul64(v0, m0)
				g01, f01 := bits.Mul64(v0, m1)
				g10, f10 := bits.Mul64(v1, m0)
				g11, f11 := bits.Mul64(v1, m1)
				mid, k1 := bits.Add64(g00, f01, 0)
				_, k2 := bits.Add64(mid, f10, 0)
				q0, k1 = bits.Add64(g01, g10, k1)
				q0, k2 = bits.Add64(q0, f11, k2)
				q1 = g11 + k1 + k2
				ph, pl := bits.Mul64(q0, d0)
				ph += q0*d1 + q1*d0
				rb0, c = bits.Sub64(tl, pl, 0)
				rb1, _ = bits.Sub64(th, ph, c)
				_, lt := bits.Sub64(rb0, d0, 0)
				_, lt = bits.Sub64(rb1, d1, lt)
				ge := lt ^ 1 // 1 when the remainder reaches 10^d
				q0, c = bits.Add64(q0, ge, 0)
				q1 += c
				rb0, c = bits.Sub64(rb0, d0&-ge, 0)
				rb1, _ = bits.Sub64(rb1, d1&-ge, c)
			}
			tie := rb0|rb1 == 0
			sign := ah & (1 << 49) // p is the larger magnitude
			if (ah^bh)&(1<<49) == 0 {
				var c uint64
				sl, c = bits.Add64(sl, q0, 0)
				sh += q1 + c
				if _, bw := bits.Sub64(sl, uint64(pow10_128_dw0(34)), 0); true {
					_, bw = bits.Sub64(sh, uint64(pow10_128_dw1(34)), bw)
					if bw == 0 {
						// 35 digits: round once more, at exp + 1, from the
						// truncated sum (less the unit q's rounding added) and
						// whether anything below it is nonzero.
						_, lt := bits.Sub64(rb0, h0, 0)
						_, lt = bits.Sub64(rb1, h1, lt)
						if d <= 34 && lt != 0 {
							sl, c = bits.Sub64(sl, 1, 0)
							sh -= c
						}
						sticky := d > 34 || rb0 != h0 || rb1 != h1
						var u uint64
						sl, sh, u, _ = divRemPow10_recip128(sl, sh, 1)
						if u > 5 || u == 5 && (sticky || sl&1 != 0) {
							sl, c = bits.Add64(sl, 1, 0)
							sh += c
						}
						exp++
					} else if tie {
						sl &^= 1 // a tie: the even neighbour
					}
				}
			} else {
				var c uint64
				sl, c = bits.Sub64(sl, q0, 0)
				sh -= q1 + c
				if tie {
					// p - b / 10^d lies halfway between s and s + 1: the even one.
					sl, c = bits.Add64(sl, sl&1, 0)
					sh += c
				}
				if _, bw := bits.Sub64(uint64(pow10_128_dw0(33)), sl, 0); true {
					if _, bw = bits.Sub64(uint64(pow10_128_dw1(33)), sh, bw); bw == 0 {
						// At most 10^33: a digit was lost to the scale. For d up
						// to 4 the exact difference at bExp still fits two words;
						// it is rounded to 34 digits from there.
						if d > 4 {
							goto slow
						}
						g0, g1 := uint64(pow10_128_dw0(gap)), uint64(pow10_128_dw1(gap))
						sh, sl = bits.Mul64(al, g0)
						sh += al*g1 + a1*g0
						sl, c = bits.Sub64(sl, bl, 0)
						sh, _ = bits.Sub64(sh, b1, c)
						exp = int(bh >> 50)
						if j := digitLen_u128(sl, sh) - 34; j > 0 {
							j0, j1 := uint64(pow10_128_dw0(j)), uint64(pow10_128_dw1(j))
							sl, c = bits.Add64(sl, j0>>1|j1<<63, 0)
							sh += j1>>1 + c
							var r0, r1 uint64
							sl, sh, r0, r1 = divRemPow10_recip128(sl, sh, j)
							if r0|r1 == 0 {
								sl &^= 1 // a tie: the even neighbour
							}
							exp += j
						}
					}
				}
			}
			_, bw := bits.Sub64(sl, uint64(pow10_128_dw0(34)), 0)
			_, bw = bits.Sub64(sh, uint64(pow10_128_dw1(34)), bw)
			if bw != 0 && exp <= 6111 {
				return d128{ubdLo64: int64(sl), ubdHi64: int64(exp)<<50 | sign | int64(sh)}
			}
		}
	}
slow:
	return d128_add_slow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
}

func d128_add_rnd(x, y d128, rnd round) d128 { return d128_add_rnd_ctxnull(x, y, rnd, nil) }
func d128_add_ctx(x, y d128, ctx *context) d128 {
	return d128_add_rnd_ctxnull(x, y, ctx.round, ctx)
}

// Fast entry: same sign and exponent, and both coefficients below 2^64,
// returns directly (no rounding, no flags). Every other case tail-calls the
// out-of-line d128_add_slow_rnd_ctxnull.
func d128_add_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// One test: (xHi ^ yHi) | xHi << 15 == 0 holds exactly when the high words
	// are equal (same qExp and sign) and hold no coefficient bits, which also
	// excludes the non-finite values (their coefficient bits are set).
	if ((x.ubdHi64 ^ y.ubdHi64) | (x.ubdHi64 << 15)) == 0 {
		// The sum is below 2^65, so canonical; its carry goes into x's high
		// word, whose qExp and sign carry through unchanged.
		sum := add_128_128x128(x.ubdLo64, x.ubdHi64, y.ubdLo64, 0)
		return d128{ubdLo64: sum.dw0, ubdHi64: sum.dw1}
	}
	return d128_add_slow_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Slow entry: the non-finite, zero, and general finite-nonzero cases. Kept out
// of line so the fast entry stays small.
//
//go:noinline
func d128_add_slow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Kept out of line so the fast entry stays a small, i-cache-friendly hot function.
	// Go has no always_inline, and the fast entry's guard + inline arithmetic already
	// exceeds the ~80 inline budget, so it does not collapse into Add (unlike the
	// native ports' operators); the split's Go payoff is hot/cold code separation.
	// The go:noinline directive pins that contract.
	// Aligned operands whose coefficients sum below 2^112, ahead of the cold
	// ladder. hi49Sum fitting in 48 bits proves BOTH operands finite (nonfinite
	// coefficients are >= 10^34, whose bit 48 is set). The same sign adds the
	// two-word coefficients; otherwise the signs differ and the magnitudes
	// subtract. Both coefficients are canonical and unscaled, so the
	// difference is < 10^34 and packs raw; swapped means |x| < |y|, and the
	// result takes the larger magnitude's sign. Mirror of the subtract slow
	// entry's first test.
	xHi49 := d128_hi49(x)
	yHi49 := d128_hi49(y)
	hi49Sum := xHi49 + yHi49
	if ushr_64_64(hi49Sum, 48) == 0 && d128_qExp(x) == d128_qExp(y) {
		if d128_qExpAndSignBit(x) == d128_qExpAndSignBit(y) {
			// Same sign, coefficients of up to 112 bits: the two-word sum.
			// hi49Sum < 2^48, so the sum never carries into the qExp field and
			// x's qExp and sign carry through in its high word.
			sum := add_128_128x128(x.ubdLo64, x.ubdHi64, y.ubdLo64, yHi49)
			return d128{ubdLo64: sum.dw0, ubdHi64: sum.dw1}
		}
		verify("", func() bool { return d128_signFlag(x) != d128_signFlag(y) })
		ds := subAbs_d128swap_128x128(x.ubdLo64, xHi49, y.ubdLo64, yHi49)
		if isZero_bool_128(ds.diffDw0, ds.diffDw1) {
			return d128_zero(rounding == const_ROUND_TOWARD_NEGATIVE, d128_qExp(x))
		}
		resultQExpAndSignBit := d128_qExpAndSignBit(x) ^ boolToInt64(ds.swap)
		return d128FromQExpSignBit(resultQExpAndSignBit, ds.diffDw0, ds.diffDw1)
	}
	if d128_isNonFinite(x) || d128_isNonFinite(y) {
		return addNonFinite_rnd_ctxnull(x, y, ctxnull)
	}
	if d128_isZER(x) {
		if d128_isZER(y) {
			resultSign := (d128_signFlag(x) && d128_signFlag(y)) ||
				(d128_signFlag(x) != d128_signFlag(y) && rounding == const_ROUND_TOWARD_NEGATIVE)
			return d128_zero(resultSign, min(d128_qExp(x), d128_qExp(y)))
		}
		return preferMinimumQExp_exact(y, d128_qExp(x))
	}
	if d128_isZER(y) {
		return preferMinimumQExp_exact(x, d128_qExp(y))
	}
	return addFnzFnz_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Finite-nonzero + finite-nonzero add core: nearly aligned operands are
// aligned exactly, the rest go to the unaligned kernels. Subtract delegates
// here through subtract(x, y) = add(x, negate(y)), so it is module-internal
// rather than file-private.
func addFnzFnz_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	qAligned := min(d128_qExp(x), d128_qExp(y))
	absXShift := d128_qExp(x) - qAligned // one of these is zero
	absYShift := d128_qExp(y) - qAligned
	if absXShift+absYShift <= 4 {
		// Nearly aligned: align to qAligned without overflowing 128 bits.
		var xCoeffAligned u128
		if absXShift == 0 {
			xCoeffAligned = d128_u128(x)
		} else {
			xCoeffAligned = mulPow10_128_128(x.ubdLo64, d128_hi49(x), absXShift)
		}
		var yCoeffAligned u128
		if absYShift == 0 {
			yCoeffAligned = d128_u128(y)
		} else {
			yCoeffAligned = mulPow10_128_128(y.ubdLo64, d128_hi49(y), absYShift)
		}

		var resultSign bool
		var resultCoeff u128
		if d128_signFlag(x) == d128_signFlag(y) {
			resultSign = d128_signFlag(x)
			resultCoeff = add_128_128x128(xCoeffAligned.dw0, xCoeffAligned.dw1, yCoeffAligned.dw0, yCoeffAligned.dw1)
		} else {
			ds := subAbs_d128swap_128x128(xCoeffAligned.dw0, xCoeffAligned.dw1, yCoeffAligned.dw0, yCoeffAligned.dw1)
			if isZero_bool_128(ds.diffDw0, ds.diffDw1) {
				return d128_zero(rounding == const_ROUND_TOWARD_NEGATIVE, qAligned)
			}
			resultSign = d128_signFlag(x) != ds.swap
			resultCoeff = u128{ds.diffDw0, ds.diffDw1}
		}
		if isLt_pow10_34_bool_128(resultCoeff.dw0, resultCoeff.dw1) {
			return d128Pack(resultSign, qAligned, resultCoeff.dw0, resultCoeff.dw1)
		}
		return finalize_128_ctxnull(resultSign, qAligned, resultCoeff, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	if d128_signFlag(x) == d128_signFlag(y) {
		return addFnzFnzUnalignedLikeSign_rnd_ctxnull(x, y, rounding, ctxnull)
	}
	return addFnzFnzUnalignedUnlikeSign_rnd_ctxnull(x, y, rounding, ctxnull)
}

func addFnzFnzUnalignedLikeSign_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	xQ := d128_qExp(x)
	yQ := d128_qExp(y)
	verify("", func() bool { return xQ != yQ })
	verify("", func() bool { return d128_signFlag(x) == d128_signFlag(y) })

	var m, n d128
	var mQ, nQ int
	if xQ > yQ {
		m, n = x, y
		mQ = xQ
		nQ = yQ
	} else {
		m, n = y, x
		mQ = yQ
		nQ = xQ
	}
	mDigitLen := d128_digitLen(m)
	qDelta := mQ - nQ
	verify("", func() bool { return qDelta > 0 })
	headroom := 34 - mDigitLen
	shiftLeft := min(max(headroom, 0), qDelta)
	var zCoeff u128
	if shiftLeft == 0 {
		zCoeff = d128_u128(m)
	} else {
		zCoeff = mulPow10_128_128(m.ubdLo64, d128_hi49(m), shiftLeft)
	}
	qAlign := mQ - shiftLeft
	shiftRight := qAlign - nQ
	// MQ FAST TAIL: gap fully absorbed into m's headroom (shiftRight == 0), so n
	// already sits at the result scale and, being the same sign, its coefficient
	// simply adds. The sum is EXACT: skip the div-by-zero and residue finalize.
	if shiftRight == 0 {
		nCoeff := d128_u128(n)
		sum := add_128_128x128(zCoeff.dw0, zCoeff.dw1, nCoeff.dw0, nCoeff.dw1)
		if isLt_pow10_34_bool_128(sum.dw0, sum.dw1) {
			return d128Pack(d128_signFlag(m), qAlign, sum.dw0, sum.dw1)
		}
		return finalize_128_ctxnull(d128_signFlag(m), qAlign, sum, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	// Swamped fast path: n lies strictly below the result's LSD (LT_HALF residue).
	if shiftRight >= 35 || shiftRight > d128_digitLen(n) {
		return finalizeSwampedFnz_128_ctxnull(d128_signFlag(m), qAlign, zCoeff, const_RESIDUE_LT_HALF, rounding, ctxnull)
	}
	qr := div_128_pow10_checked_q128res(d128_u128(n), shiftRight)
	res := qr.residue
	zCoeff = add_128_128x128(zCoeff.dw0, zCoeff.dw1, qr.quotDw0, qr.quotDw1)
	return finalizeAddSubFnz_128_ctxnull(d128_signFlag(m), qAlign, zCoeff, res, rounding, ctxnull)
}

func addFnzFnzUnalignedUnlikeSign_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return d128_qExp(x) != d128_qExp(y) })
	verify("", func() bool { return d128_signFlag(x) != d128_signFlag(y) })
	cmp := cmpMagFnzFnz(x, y)
	if cmp > 0 {
		return subMagUnalignedFnzFnz_rnd_ctxnull(d128_signFlag(x), x, y, rounding, ctxnull)
	}
	if cmp < 0 {
		return subMagUnalignedFnzFnz_rnd_ctxnull(d128_signFlag(y), y, x, rounding, ctxnull)
	}
	return d128_zero(rounding == const_ROUND_TOWARD_NEGATIVE, min(d128_qExp(x), d128_qExp(y)))
}

func addNonFinite_rnd_ctxnull(x, y d128, ctxnull *context) d128 {
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	verify("", func() bool { return d128_isINF(x) || d128_isINF(y) })
	if d128_isFIN(x) {
		return y
	}
	if d128_isFIN(y) {
		return x
	}
	if d128_signFlag(x) == d128_signFlag(y) {
		return x
	}
	return ctxSignalInvalidOperation_ctxnull(ctxnull, causeMagnitudeSubtractionOfInfinities, const_D128_QNAN0)
}

// preferMinimumQExp_exact lowers x's exponent toward otherExp by scaling its coefficient up into the
// free digits, exactly (the value is unchanged). Returns x unchanged when
// there is no headroom or otherExp >= qExp(x). Shared with fma.
func preferMinimumQExp_exact(x d128, otherExp int) d128 {
	xSign := d128_signFlag(x)
	xQ := d128_qExp(x)
	xCoeff := d128_u128(x)
	delta := xQ - otherExp
	if delta <= 0 {
		return x
	}
	headroom := max(0, 34-d128_digitLen(x))
	if headroom == 0 {
		return x
	}
	expRoom := xQ + 6176
	shiftLeft := min(min(headroom, delta), expRoom)
	var zCoeff u128
	if shiftLeft == 0 {
		zCoeff = xCoeff
	} else {
		zCoeff = mulPow10_128_128(xCoeff.dw0, xCoeff.dw1, shiftLeft)
	}
	zQ := xQ - shiftLeft
	verify("", func() bool { return isCanonical34_128(zQ, zCoeff) })
	return d128Pack(xSign, zQ, zCoeff.dw0, zCoeff.dw1)
}

// digitLen_u128 is the number of decimal digits of (hi:lo) (0 for 0): the
// estimate from the bit length is the count or one short, and it is short exactly
// when the value less 10^t does not borrow. No data-dependent branch.
func digitLen_u128(lo, hi uint64) int {
	n := bits.Len64(lo)
	if hi != 0 {
		n = 64 + bits.Len64(hi)
	}
	t := (n * 1233) >> 12
	_, b := bits.Sub64(lo, uint64(pow10_128_dw0(t)), 0)
	_, b = bits.Sub64(hi, uint64(pow10_128_dw1(t)), b)
	return t + 1 - int(b)
}
