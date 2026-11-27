// IEEE 754 subtract. Subtraction is add-with-negated-y after its own aligned fast
// path: `d128_subtract_rnd_ctxnull` is the general body under the `_tte`, `_rnd` and
// `_ctx` forms. The aligned magnitude-add subcase delegates to add's
// `addFnzFnz_rnd_ctxnull`; the unaligned magnitude-subtract kernel
// `subMagUnalignedFnzFnz_rnd_ctxnull` lives here, and add's unlike-sign path reaches it
// too.

package decimal128

import "math/bits"

// d128_subtract_tte is the ties-to-even subtract without a context, in one frame
// (GoUbdKernelGapWorkOrder.md L4): d128_add_tte's word paths with y's sign flipped
// (x - y = x + (-y)), then the general slow entry, which gets y itself. A NaN y
// never reaches the word paths (its high word holds coefficient bits), so the flip
// never changes which NaN is returned.
func d128_subtract_tte(x, y d128) d128 {
	xh, yh := x.ubdHi64, y.ubdHi64^(1<<49) // -y
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
		} else if (xh^yh)>>50 == 0 {
			// One exponent, opposite signs: the two-word difference that did
			// not borrow, chosen by selects. Its magnitude is at most the
			// larger operand's, below 10^34, so it is exact: no digit count and
			// no 10^34 check. An exact zero is +0.
			lo, b := bits.Sub64(uint64(x.ubdLo64), uint64(y.ubdLo64), 0)
			hi, bw := bits.Sub64(uint64(xh)&(1<<49-1), uint64(yh)&(1<<49-1), b)
			m := -bw // all ones when y's magnitude was the larger
			nlo, nb := bits.Sub64(0, lo, 0)
			nhi, _ := bits.Sub64(0, hi, nb)
			lo = lo&^m | nlo&m
			hi = hi&^m | nhi&m
			sign := (xh ^ int64(bw)<<49) & (1 << 49)
			if hi|lo == 0 {
				sign = 0 // an exact cancellation: +0
			}
			return d128{ubdLo64: int64(lo), ubdHi64: xh>>50<<50 | sign | int64(hi)}
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
	return d128_subtract_slow_rnd_ctxnull(x, y, const_ROUND_TIES_TO_EVEN, nil)
}
func d128_subtract_rnd(x, y d128, rnd round) d128 { return d128_subtract_rnd_ctxnull(x, y, rnd, nil) }
func d128_subtract_ctx(x, y d128, ctx *context) d128 {
	return d128_subtract_rnd_ctxnull(x, y, ctx.round, ctx)
}

// Fast entry: same sign and exponent, and both coefficients below 2^64 (an
// effective subtraction), returns directly unless the difference is zero. A
// zero difference, and every other case, tail-calls the out-of-line
// d128_subtract_slow_rnd_ctxnull.
func d128_subtract_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// Add's one-test guard: same qExp, same sign, no coefficient bits in
	// either high word.
	if ((x.ubdHi64 ^ y.ubdHi64) | (x.ubdHi64 << 15)) == 0 {
		// The two-word difference's high dword is all ones exactly when the
		// low words borrow (|y| > |x|): it is the mask that negates the
		// difference and flips the sign. A zero difference goes to the slow
		// entry, since its sign depends on the rounding direction.
		diff := sub_128_128x128(x.ubdLo64, 0, y.ubdLo64, 0)
		if diff.dw0 != 0 {
			borrowMask := diff.dw1
			return d128{ubdLo64: (diff.dw0 ^ borrowMask) - borrowMask, ubdHi64: x.ubdHi64 ^ (borrowMask & (1 << 49))}
		}
	}
	return d128_subtract_slow_rnd_ctxnull(x, y, rounding, ctxnull)
}

// Slow entry: NaN operands, then everything else the fast entry did not
// resolve, through add's non-finite, zero and finite-nonzero paths with y
// negated (subtract(x, y) = add(x, negate(y))).
//
//go:noinline
func d128_subtract_slow_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	// The go:noinline directive pins this out of line so the fast entry stays the
	// small hot function (see d128_add_slow_rnd_ctxnull).
	// Aligned operands whose coefficients sum below 2^112, ahead of the cold
	// ladder. hi49Sum < 2^48 proves both operands finite (non-finite
	// coefficients are >= 10^34, whose bit 48 is set). Opposite signs add the
	// magnitudes; same signs subtract them unless the difference is zero.
	xHi49 := d128_hi49(x)
	yHi49 := d128_hi49(y)
	hi49Sum := xHi49 + yHi49
	if ushr_64_64(hi49Sum, 48) == 0 && d128_qExp(x) == d128_qExp(y) {
		if d128_qExpAndSignBit(x) != d128_qExpAndSignBit(y) {
			// Opposite signs: magnitude addition; x's qExp and sign carry through.
			sum := add_128_128x128(x.ubdLo64, x.ubdHi64, y.ubdLo64, yHi49)
			return d128{ubdLo64: sum.dw0, ubdHi64: sum.dw1}
		}
		ds := subAbs_d128swap_128x128(x.ubdLo64, xHi49, y.ubdLo64, yHi49)
		if !isZero_bool_128(ds.diffDw0, ds.diffDw1) {
			resultQExpAndSignBit := d128_qExpAndSignBit(x) ^ boolToInt64(ds.swap)
			return d128FromQExpSignBit(resultQExpAndSignBit, ds.diffDw0, ds.diffDw1)
		}
	}
	// The NaN, non-finite, zero and general cases.
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2_ctxnull(ctxnull, x, y)
	}
	yNeg := d128_negate(y)
	if d128_isNonFinite(x) || d128_isNonFinite(yNeg) || d128_isZER(x) || d128_isZER(yNeg) {
		return d128_add_rnd_ctxnull(x, yNeg, rounding, ctxnull)
	}
	return addFnzFnz_rnd_ctxnull(x, yNeg, rounding, ctxnull)
}

func subMagUnalignedFnzFnz_rnd_ctxnull(zSign bool, m, s d128, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return d128_isFNZ(m) && d128_isFNZ(s) })
	verify("", func() bool { return d128_qExp(m) != d128_qExp(s) })
	verify("", func() bool { return cmpMagFnzFnz(m, s) > 0 })
	mQ := d128_qExp(m)
	mCoeff := d128_u128(m)
	mDigitLen := digitLen_int_128(mCoeff.dw0, mCoeff.dw1)
	sQ := d128_qExp(s)
	sCoeff := d128_u128(s)
	sDigitLen := -1
	gap := mQ - sQ
	if gap < 0 {
		prod := mulPow10_128_128(sCoeff.dw0, sCoeff.dw1, -gap)
		diff := sub_128_128x128(mCoeff.dw0, mCoeff.dw1, prod.dw0, prod.dw1)
		return finalize_128_ctxnull(zSign, mQ, diff, const_RESIDUE_EXACT, rounding, ctxnull)
	}
	verify("", func() bool { return mDigitLen <= 34 })
	headroom := 34 - mDigitLen
	// MQ FAST TAIL (magnitude subtraction): gap fully absorbed into m's headroom
	// (gap <= headroom), so s already sits at the result scale and m*10^gap - s is
	// EXACT — no guard digit, no divide, no residue. diff < 10^34 because |m| > |s|
	// and m*10^gap has <= 34 digits; result exponent sQ is the preferred (minimum)
	// exponent for an exact subtraction.
	if gap <= headroom {
		mScaled := mulPow10_128_128(mCoeff.dw0, mCoeff.dw1, gap)
		diff := sub_128_128x128(mScaled.dw0, mScaled.dw1, sCoeff.dw0, sCoeff.dw1)
		if isLt_pow10_34_bool_128(diff.dw0, diff.dw1) {
			return d128Pack(zSign, sQ, diff.dw0, diff.dw1)
		}
		return finalize_128_ctxnull(zSign, sQ, diff, const_RESIDUE_EXACT, rounding, ctxnull)
	}

	// Fully/Borderline swamped fast path at 34-digit scale.
	shiftM34 := min(gap, headroom)
	qAlign34 := mQ - shiftM34
	shiftS34 := qAlign34 - sQ
	isSwamped := (shiftS34 >= 35)
	if !isSwamped {
		sDigitLen = digitLen_int_128(sCoeff.dw0, sCoeff.dw1)
		isSwamped = (shiftS34 >= sDigitLen)
	}
	if isSwamped {
		var mScaled u128
		if shiftM34 == 0 {
			mScaled = mCoeff
		} else {
			mScaled = mulPow10_128_128(mCoeff.dw0, mCoeff.dw1, shiftM34)
		}
		if cmp_int_128x128_pow10(mScaled.dw0, mScaled.dw1, 33) != 0 { // not 10^33
			raw := const_RESIDUE_LT_HALF
			if !(shiftS34 >= 35 || shiftS34 > sDigitLen) {
				raw = residueFromDecadeWithDigitLen_128(sCoeff.dw0, sCoeff.dw1, sDigitLen)
			}
			z := sub_128_128x128(mScaled.dw0, mScaled.dw1, 1, 0)
			return finalizeSwampedFnz_128_ctxnull(zSign, qAlign34, z, residueSubtractionInverse(raw), rounding, ctxnull)
		}
		// mScaled == 10^33: fall through to the guard path.
	}

	// Guard-digit path (the 10^33 exception and partial overlap).
	headroomWithGuard := 1 + headroom
	shiftMLeft := min(gap, headroomWithGuard)
	qAlign := mQ - shiftMLeft
	shiftSRight := qAlign - sQ
	verify("", func() bool { return shiftSRight >= 0 })

	zCoeff := mulPow10_128_128(mCoeff.dw0, mCoeff.dw1, shiftMLeft)
	if sDigitLen == -1 {
		sDigitLen = digitLen_int_128(sCoeff.dw0, sCoeff.dw1)
	}
	if shiftSRight > sDigitLen {
		borrowed := sub_128_128x128(zCoeff.dw0, zCoeff.dw1, 1, 0)
		return finalizeAddSubFnz_128_ctxnull(zSign, qAlign, borrowed, residueSubtractionInverse(const_RESIDUE_LT_HALF), rounding, ctxnull)
	}
	qr := div_128_pow10_checked_q128res(sCoeff, shiftSRight)
	rawResidue := qr.residue
	zCoeff = sub_128_128x128(zCoeff.dw0, zCoeff.dw1, qr.quotDw0, qr.quotDw1)
	var res residue
	if rawResidue == const_RESIDUE_EXACT {
		res = const_RESIDUE_EXACT
	} else {
		res = residueSubtractionInverse(rawResidue)
		zCoeff = sub_128_128x128(zCoeff.dw0, zCoeff.dw1, 1, 0)
	}
	return finalizeAddSubFnz_128_ctxnull(zSign, qAlign, zCoeff, res, rounding, ctxnull)
}
