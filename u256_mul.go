// The U256 mul family (PrimitiveLayer.md §4.2), D9 segment grammar. Limb forms
// are the uniform API. Swift uses native UInt128 full-width products; Go
// synthesizes from uint64 limbs via math/bits.
//
// mul_256_256x256 faithfully replicates Swift's 128-bit column accumulator with
// an overflow flag: it traps when a column sum exceeds 2^128 (not only when the
// product exceeds 2^256), so callers pass operands bounded well below the limit
// (decimal128 coefficients < 10^34).

package decimal128

import "math/bits"

// mul_256_128x64 returns x * y; fits in 192 bits (dw3 == 0). Interior-only (§3.4).
func mul_256_128x64(x u128, y uint64) u256 {
	xLo := uint64(x.dw0)
	xHi := uint64(x.dw1)
	p0Hi, p0Lo := bits.Mul64(xLo, y)
	p1Hi, p1Lo := bits.Mul64(xHi, y)
	mid, carry := bits.Add64(p0Hi, p1Lo, 0)
	hi := p1Hi + carry
	return u256{int64(p0Lo), int64(mid), int64(hi), 0}
}

// mul_256_128x128 returns the full-width 128×128→256 product.
func mul_256_128x128(xDw0, xDw1, yDw0, yDw1 int64) u256 {
	a0, a1 := uint64(xDw0), uint64(xDw1)
	b0, b1 := uint64(yDw0), uint64(yDw1)
	h00, l00 := bits.Mul64(a0, b0)
	h01, l01 := bits.Mul64(a0, b1)
	h10, l10 := bits.Mul64(a1, b0)
	h11, l11 := bits.Mul64(a1, b1)
	r0 := l00
	r1, c1 := bits.Add64(h00, l01, 0)
	r1, c2 := bits.Add64(r1, l10, 0)
	r2, c3 := bits.Add64(h01, h10, 0)
	r2, c4 := bits.Add64(r2, l11, 0)
	r2, c5 := bits.Add64(r2, c1+c2, 0)
	r3 := h11 + c3 + c4 + c5
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}

// mul_256_256x64 returns x * y, trapping if the result exceeds 2^256.
func mul_256_256x64(xDw0, xDw1, xDw2, xDw3, y int64) u256 {
	a0, a1 := uint64(xDw0), uint64(xDw1)
	a2, a3 := uint64(xDw2), uint64(xDw3)
	d := uint64(y)

	p0Hi, p0Lo := bits.Mul64(a0, d)
	r0 := p0Lo
	p1Hi, p1Lo := bits.Mul64(a1, d)
	r1, c1 := bits.Add64(p1Lo, p0Hi, 0)
	p1HiC := p1Hi + c1
	p2Hi, p2Lo := bits.Mul64(a2, d)
	r2, c2 := bits.Add64(p2Lo, p1HiC, 0)
	p2HiC := p2Hi + c2
	p3Hi, p3Lo := bits.Mul64(a3, d)
	r3, c3 := bits.Add64(p3Lo, p2HiC, 0)
	if p3Hi != 0 || c3 != 0 {
		impossible("mul_256_256x64 overflow")
	}
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}

// mul_256_256x128 returns x * y, trapping if the result exceeds 2^256.
func mul_256_256x128(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1 int64) u256 {
	prodLo := mul_256_128x128(xDw0, xDw1, yDw0, yDw1)
	prodHi := mul_256_128x128(xDw2, xDw3, yDw0, yDw1)
	if prodHi.dw2 != 0 || prodHi.dw3 != 0 {
		impossible("mul_256_256x128 overflow")
	}
	sumLo, c := bits.Add64(uint64(prodLo.dw2), uint64(prodHi.dw0), 0)
	sumHi, c2 := bits.Add64(uint64(prodLo.dw3), uint64(prodHi.dw1), c)
	if c2 != 0 {
		impossible("mul_256_256x128 overflow")
	}
	return u256{prodLo.dw0, prodLo.dw1, int64(sumLo), int64(sumHi)}
}

// mul_256_256x256 returns x * y, trapping if the result exceeds 2^256 (or any
// column sum exceeds the 128-bit accumulator — see file header).
func mul_256_256x256(xDw0, xDw1, xDw2, xDw3, yDw0, yDw1, yDw2, yDw3 int64) u256 {
	a0, a1 := uint64(xDw0), uint64(xDw1)
	a2, a3 := uint64(xDw2), uint64(xDw3)
	b0, b1 := uint64(yDw0), uint64(yDw1)
	b2, b3 := uint64(yDw2), uint64(yDw3)

	var colLo, colHi uint64
	didOverflow := false

	mulAddCol256(a0, b0, &colLo, &colHi, &didOverflow)
	r0 := colLo
	colLo, colHi = colHi, 0

	mulAddCol256(a1, b0, &colLo, &colHi, &didOverflow)
	mulAddCol256(a0, b1, &colLo, &colHi, &didOverflow)
	r1 := colLo
	colLo, colHi = colHi, 0

	mulAddCol256(a2, b0, &colLo, &colHi, &didOverflow)
	mulAddCol256(a1, b1, &colLo, &colHi, &didOverflow)
	mulAddCol256(a0, b2, &colLo, &colHi, &didOverflow)
	r2 := colLo
	colLo, colHi = colHi, 0

	mulAddCol256(a3, b0, &colLo, &colHi, &didOverflow)
	mulAddCol256(a2, b1, &colLo, &colHi, &didOverflow)
	mulAddCol256(a1, b2, &colLo, &colHi, &didOverflow)
	mulAddCol256(a0, b3, &colLo, &colHi, &didOverflow)
	r3 := colLo
	colLo, colHi = colHi, 0

	// Columns 4+: any non-zero contribution here is overflow.
	mulAddCol256(a3, b1, &colLo, &colHi, &didOverflow)
	mulAddCol256(a2, b2, &colLo, &colHi, &didOverflow)
	mulAddCol256(a1, b3, &colLo, &colHi, &didOverflow)
	mulAddCol256(a3, b2, &colLo, &colHi, &didOverflow)
	mulAddCol256(a2, b3, &colLo, &colHi, &didOverflow)
	mulAddCol256(a3, b3, &colLo, &colHi, &didOverflow)

	if didOverflow || colLo != 0 || colHi != 0 {
		impossible("mul_256_256x256 overflow")
	}
	return u256{int64(r0), int64(r1), int64(r2), int64(r3)}
}

// mulAddCol256 adds the 128-bit product a*b into the 128-bit accumulator
// (colLo:colHi), setting didOverflow on carry-out — the Go spelling of Swift's
// UInt128 addingReportingOverflow column step.
func mulAddCol256(a, b uint64, colLo, colHi *uint64, didOverflow *bool) {
	pHi, pLo := bits.Mul64(a, b)
	sumLo, c := bits.Add64(*colLo, pLo, 0)
	sumHi, c2 := bits.Add64(*colHi, pHi, c)
	*colLo = sumLo
	*colHi = sumHi
	if c2 != 0 {
		*didOverflow = true
	}
}
