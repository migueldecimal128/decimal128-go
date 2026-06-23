// The rrmp10 kernel unit (PrimitiveLayer.md D16): range-reciprocal division by
// powers of 10 — multiply by a precomputed variable-length multiplier and shift.
// Everything below the two uniform entry points is platform-local decomposition
// (§3.4); Swift pipelines the partial products through a UInt128 carry-save
// accumulator, which Go synthesizes from uint64 limb sums (addCols).
//
// The entries return Quot128Residue directly: residue = (round << 1) | sticky,
// packed here from the kernels' raw round/sticky facts. The quotient fits 128
// bits by table construction; the entries check it.

package decimal128

import "math/bits"

// --- descriptor unpacking ---
//
//	bits  0..15 : mDwordLen        bits 32..39 : maxProdDwordLen
//	bits 16..31 : shift S          bits 40..47 : K = pow10
//	bits 48..55 : qMax             bits 56..63 : qMin

func unpackQMin(d uint64) int         { return int(d >> 56) }
func unpackQMax(d uint64) int         { return int((d >> 48) & 0xFF) }
func unpackK(d uint64) int            { return int((d >> 40) & 0xFF) }
func unpackProdDwordLen(d uint64) int { return int((d >> 32) & 0xFF) }
func unpackS(d uint64) int            { return int((d >> 16) & 0xFFFF) }
func unpackMDwordLen(d uint64) int    { return int(d & 0xFFFF) }

// --- quotient assembler (kernels write 64-bit limbs in little-endian order) ---

type quotientLimbs struct {
	limbs     [4]uint64
	nextIndex int
}

func (q *quotientLimbs) append(value uint64) {
	q.limbs[q.nextIndex] = value
	q.nextIndex += 1
}

func (q *quotientLimbs) set(index int, value uint64) {
	q.limbs[index] = value
}

// --- shift-parameter packet ---

type shiftParams struct {
	fractionBitLen        int
	stickyBitsPow2        uint64
	shiftRightNonZeroMask uint64
	shiftRight            int
	shiftLeft             int
	halfUlpBitMask        uint64
	fractionTailMask      uint64
}

func makeShiftParams(descriptor uint64, kPow10 int, x0 uint64) shiftParams {
	shift := unpackS(descriptor)
	fractionBitLen := shift + 1

	dividendShiftRight := kPow10 - 1
	verify("", dividendShiftRight >= 1 && dividendShiftRight < 64)
	stickyBitsPow2 := x0 & ((uint64(1) << uint(dividendShiftRight)) - 1)

	modulusBitLen := fractionBitLen & 0x3F
	var shiftRightNonZeroMask uint64 = 0
	if modulusBitLen != 0 {
		shiftRightNonZeroMask = ^uint64(0)
	}
	shiftRight := modulusBitLen
	shiftLeft := (64 - modulusBitLen) & 0x3F
	halfUlpBitIndex := (fractionBitLen - 1) & 0x3F
	halfUlpBitMask := uint64(1) << uint(halfUlpBitIndex)
	fractionTailMask := halfUlpBitMask - 1

	return shiftParams{
		fractionBitLen, stickyBitsPow2, shiftRightNonZeroMask,
		shiftRight, shiftLeft, halfUlpBitMask, fractionTailMask,
	}
}

// addCols sums uint64 columns into a 128-bit (lo, hi) pair via a carry chain.
func addCols(vals ...uint64) (uint64, uint64) {
	var lo, hi uint64
	i := 0
	for i < len(vals) {
		var c uint64
		lo, c = bits.Add64(lo, vals[i], 0)
		hi += c
		i += 1
	}
	return lo, hi
}

// --- residue packing ---

func residueFromRoundBitSticky(isolatedRoundBit uint64, stickyBitsFracCompare int, stickyBitsPow2 uint64) residue {
	stickyBit := 0
	if stickyBitsFracCompare >= 0 || stickyBitsPow2 != 0 {
		stickyBit = 1
	}
	roundBit := 0
	if isolatedRoundBit != 0 {
		roundBit = 1
	}
	return (roundBit << 1) | stickyBit
}

// --- uniform entry points (D16) ---

func divPow10_q128res_128_rrmp10(xDw0, xDw1 int64, pIndex, pow10 int) quot128Residue {
	x0 := uint64(xDw0)
	x1 := uint64(xDw1)
	descriptor := rrmp10Params[pIndex]
	verify("", pow10 == unpackK(descriptor))
	verify("", x1 != 0)

	sp := makeShiftParams(descriptor, pow10, x0)
	dsr := pow10 - 1
	dsl := 64 - dsr

	d0 := (x1 << uint(dsl)) | uint64(ushr_64_64(int64(x0), dsr))
	d1 := uint64(ushr_64_64(int64(x1), dsr))

	mOff := pIndex + 1
	mLen := unpackMDwordLen(descriptor)

	var quot quotientLimbs
	roundBit, stickyCmp := rrmp10_kernel_128(&quot, mOff, mLen, d1, d0, sp)

	verify("", (quot.limbs[3]|quot.limbs[2]) == 0)
	res := residueFromRoundBitSticky(roundBit, stickyCmp, sp.stickyBitsPow2)
	return quot128Residue{int64(quot.limbs[0]), int64(quot.limbs[1]), res}
}

func divPow10_q128res_256_rrmp10(xDw0, xDw1, xDw2, xDw3 int64, pIndex, pow10 int) quot128Residue {
	x0 := uint64(xDw0)
	x1 := uint64(xDw1)
	x2 := uint64(xDw2)
	x3 := uint64(xDw3)
	descriptor := rrmp10Params[pIndex]
	verify("", pow10 == unpackK(descriptor))
	verify("", x2 != 0)

	sp := makeShiftParams(descriptor, pow10, x0)
	dsr := pow10 - 1
	dsl := 64 - dsr

	d0 := (x1 << uint(dsl)) | uint64(ushr_64_64(int64(x0), dsr))
	d1 := (x2 << uint(dsl)) | uint64(ushr_64_64(int64(x1), dsr))
	d2 := (x3 << uint(dsl)) | uint64(ushr_64_64(int64(x2), dsr))
	d3 := uint64(ushr_64_64(int64(x3), dsr))

	mOff := pIndex + 1
	mLen := unpackMDwordLen(descriptor)

	var quot quotientLimbs
	var roundBit uint64
	var stickyCmp int
	if d3 != 0 {
		roundBit, stickyCmp = rrmp10_kernel_256(&quot, mOff, mLen, d3, d2, d1, d0, sp)
	} else {
		roundBit, stickyCmp = rrmp10_kernel_192(&quot, mOff, mLen, d2, d1, d0, sp)
	}

	demand("rrmp10: result did not fit in 128 bits", (quot.limbs[3]|quot.limbs[2]) == 0)
	res := residueFromRoundBitSticky(roundBit, stickyCmp, sp.stickyBitsPow2)
	return quot128Residue{int64(quot.limbs[0]), int64(quot.limbs[1]), res}
}

// --- kernels ---

func rrmp10_kernel_256(quot *quotientLimbs, mOff, mLen int, n3, n2, n1, n0 uint64, sp shiftParams) (uint64, int) {
	var isolatedRoundBit uint64 = 0
	fractionBitsRemaining := sp.fractionBitLen
	stickyBitsFracCompare := 0
	verify("", (sp.fractionBitLen+63)>>6 >= mLen)

	var z1 uint64 = 0
	var pp31_4, pp31_3, pp31_2, pp31_1 uint64
	var pp30_3, pp30_2, pp30_1 uint64
	var pp21_3, pp21_2, pp21_1 uint64
	var pp20_2, pp20_1 uint64
	var pp11_2, pp11_1 uint64
	var pp10_1, pp01_1, carry_1 uint64

	mI := 0
	for mI < mLen {
		mX := rrmp10Params[mOff+mI]
		pp01_0, pp00_0 := bits.Mul64(mX, n0)
		pp11_0, pp10_0 := bits.Mul64(mX, n1)
		pp21_0, pp20_0 := bits.Mul64(mX, n2)
		pp31_0, pp30_0 := bits.Mul64(mX, n3)

		z0, carry_0 := addCols(pp31_4, pp30_3, pp21_3, pp20_2, pp11_2, pp10_1, pp01_1, pp00_0, carry_1)

		processLimbLoop(z0, z1, mX, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)
		z1 = z0

		pp31_4, pp31_3, pp31_2, pp31_1 = pp31_3, pp31_2, pp31_1, pp31_0
		pp30_3, pp30_2, pp30_1 = pp30_2, pp30_1, pp30_0
		pp21_3, pp21_2, pp21_1 = pp21_2, pp21_1, pp21_0
		pp20_2, pp20_1 = pp20_1, pp20_0
		pp11_2, pp11_1 = pp11_1, pp11_0
		pp10_1 = pp10_0
		pp01_1 = pp01_0
		carry_1 = carry_0
		mI += 1
	}

	zt1, carry1 := addCols(pp31_4, pp30_3, pp21_3, pp20_2, pp11_2, pp10_1, pp01_1, carry_1)
	processLimbTail(zt1, z1, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	zt2, carry2 := addCols(pp31_3, pp30_2, pp21_2, pp20_1, pp11_1, carry1)
	processLimbTail(zt2, zt1, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	zt3, carry3 := addCols(pp31_2, pp30_1, pp21_1, carry2)
	processLimbTail(zt3, zt2, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	zt4 := pp31_1 + carry3
	processLimbTail(zt4, zt3, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	verify("", fractionBitsRemaining <= 0)
	q4 := (zt4 >> uint(sp.shiftRight)) & sp.shiftRightNonZeroMask
	if q4 != 0 {
		quot.set(quot.nextIndex, q4)
	}
	return isolatedRoundBit, stickyBitsFracCompare
}

func rrmp10_kernel_192(quot *quotientLimbs, mOff, mLen int, n2, n1, n0 uint64, sp shiftParams) (uint64, int) {
	var isolatedRoundBit uint64 = 0
	fractionBitsRemaining := sp.fractionBitLen
	stickyBitsFracCompare := 0
	verify("", (sp.fractionBitLen+63)>>6 >= mLen)

	var z1 uint64 = 0
	var pp21_3, pp21_2, pp21_1 uint64
	var pp20_2, pp20_1 uint64
	var pp11_2, pp11_1 uint64
	var pp10_1, pp01_1, carry_1 uint64

	mI := 0
	for mI < mLen {
		mX := rrmp10Params[mOff+mI]
		pp01_0, pp00_0 := bits.Mul64(mX, n0)
		pp11_0, pp10_0 := bits.Mul64(mX, n1)
		pp21_0, pp20_0 := bits.Mul64(mX, n2)

		z0, carry_0 := addCols(pp21_3, pp20_2, pp11_2, pp10_1, pp01_1, pp00_0, carry_1)

		processLimbLoop(z0, z1, mX, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)
		z1 = z0

		pp21_3, pp21_2, pp21_1 = pp21_2, pp21_1, pp21_0
		pp20_2, pp20_1 = pp20_1, pp20_0
		pp11_2, pp11_1 = pp11_1, pp11_0
		pp10_1 = pp10_0
		pp01_1 = pp01_0
		carry_1 = carry_0
		mI += 1
	}

	zt1, carry1 := addCols(pp21_3, pp20_2, pp11_2, pp10_1, pp01_1, carry_1)
	processLimbTail(zt1, z1, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	zt2, carry2 := addCols(pp21_2, pp20_1, pp11_1, carry1)
	processLimbTail(zt2, zt1, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	zt3 := pp21_1 + carry2
	processLimbTail(zt3, zt2, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	verify("", fractionBitsRemaining <= 0)
	q3 := (zt3 >> uint(sp.shiftRight)) & sp.shiftRightNonZeroMask
	if q3 != 0 {
		quot.set(quot.nextIndex, q3)
	}
	return isolatedRoundBit, stickyBitsFracCompare
}

func rrmp10_kernel_128(quot *quotientLimbs, mOff, mLen int, n1, n0 uint64, sp shiftParams) (uint64, int) {
	var isolatedRoundBit uint64 = 0
	fractionBitsRemaining := sp.fractionBitLen
	stickyBitsFracCompare := 0
	verify("", (sp.fractionBitLen+63)>>6 >= mLen)

	var z1 uint64 = 0
	var pp11_2, pp11_1 uint64
	var pp10_1, pp01_1, carry_1 uint64

	mI := 0
	for mI < mLen {
		mX := rrmp10Params[mOff+mI]
		pp01_0, pp00_0 := bits.Mul64(mX, n0)
		pp11_0, pp10_0 := bits.Mul64(mX, n1)

		z0, carry_0 := addCols(pp11_2, pp10_1, pp01_1, pp00_0, carry_1)

		processLimbLoop(z0, z1, mX, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)
		z1 = z0

		pp11_2, pp11_1 = pp11_1, pp11_0
		pp10_1 = pp10_0
		pp01_1 = pp01_0
		carry_1 = carry_0
		mI += 1
	}

	zt1, carry1 := addCols(pp11_2, pp10_1, pp01_1, carry_1)
	processLimbTail(zt1, z1, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	zt2 := pp11_1 + carry1
	processLimbTail(zt2, zt1, &fractionBitsRemaining, &isolatedRoundBit, &stickyBitsFracCompare, sp, quot)

	verify("", fractionBitsRemaining <= 0)
	q2 := (zt2 >> uint(sp.shiftRight)) & sp.shiftRightNonZeroMask
	if q2 != 0 {
		quot.set(quot.nextIndex, q2)
	}
	return isolatedRoundBit, stickyBitsFracCompare
}

// --- limb processing helpers ---

func processLimbLoop(zCurr, zPrev, mX uint64, fractionBitsRemaining *int, isolatedRoundBit *uint64, stickyBitsFracCompare *int, sp shiftParams, quot *quotientLimbs) {
	if *fractionBitsRemaining > 0 {
		*fractionBitsRemaining -= 64
		var mask uint64
		if *fractionBitsRemaining <= 0 {
			*isolatedRoundBit = zCurr & sp.halfUlpBitMask
			mask = sp.fractionTailMask
		} else {
			mask = ^uint64(0)
		}
		if mask != 0 {
			masked := zCurr & mask
			cmp := 0
			if masked < mX {
				cmp = -1
			} else if masked > mX {
				cmp = 1
			}
			if cmp != 0 {
				*stickyBitsFracCompare = cmp
			}
		}
	} else {
		q0 := (zCurr << uint(sp.shiftLeft)) | ((zPrev >> uint(sp.shiftRight)) & sp.shiftRightNonZeroMask)
		quot.append(q0)
	}
}

func processLimbTail(zCurr, zPrev uint64, fractionBitsRemaining *int, isolatedRoundBit *uint64, stickyBitsFracCompare *int, sp shiftParams, quot *quotientLimbs) {
	if *fractionBitsRemaining > 0 {
		*fractionBitsRemaining -= 64
		var mask uint64
		if *fractionBitsRemaining <= 0 {
			*isolatedRoundBit = zCurr & sp.halfUlpBitMask
			mask = sp.fractionTailMask
		} else {
			mask = ^uint64(0)
		}
		if mask != 0 && (zCurr&mask) != 0 {
			*stickyBitsFracCompare = 1
		}
	} else {
		q0 := (zCurr << uint(sp.shiftLeft)) | ((zPrev >> uint(sp.shiftRight)) & sp.shiftRightNonZeroMask)
		quot.append(q0)
	}
}
