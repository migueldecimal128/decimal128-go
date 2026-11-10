// Floor square root of a u320 for the D48 sqrt (D48GoWorkOrder.md G-8; Swift
// U320Isqrt.swift). Newton from above: a start >= the root from a float64
// sqrt of the top 62-63 bits, then s <- (s + r/s) / 2 with the 320÷160 MG
// divide until s stops decreasing (about three divides for D48's 95-96-digit
// radicands). Radicands of at most 128 bits take the same iteration on 128 bits.

package decimal128

import (
	"math"
	"math/bits"
)

// isqrt_320 is floor(sqrt(r)); the root of a u320 is < 2^160, returned as a u192.
func isqrt_320(r u320) u192 {
	nbits := bitLen_int_320(r)
	if nbits <= 128 {
		return u192{dw0: int64(isqrt_128(uint64(r.dw0), uint64(r.dw1)))}
	}
	// Top 62-63 bits with an even shift, so sqrt(r) ~ sqrt(t)·2^(shift/2).
	shift := (nbits - 62) &^ 1
	t := uint64(shr_320_320(r, shift).dw0)
	f := math.Sqrt(float64(t)) // < 2^32, relative error < 2^-52
	// sqrt(r) < sqrt(t + 1)·2^(shift/2); scale f by 2^20 and pad so the start
	// is never below the root (Newton from above needs that).
	half := shift >> 1             // >= 33
	start := uint64(f * 1048576.0) // < 2^52
	x := shl_320_320(u320FromDw0(int64(start+4)), half-20)
	// Near 2^320 the padded start can reach 2^160; the root is below it, so the
	// largest 160-bit value is still a start from above (and a valid divisor).
	if bitLen_int_320(x) > 160 {
		x = u320{-1, -1, int64(^uint64(0) >> 32), 0, 0}
	}
	for {
		q := divRem_320_160_mg(r, x.u192()).quot()
		y := shr_320_320(add_320_320x320(x, q), 1)
		if !lt_320(y, x) {
			break
		}
		x = y
	}
	return x.u192()
}

// isqrt_128 is floor(sqrt(r)) for a 128-bit radicand (lo, hi), Newton from
// above.
func isqrt_128(lo, hi uint64) uint64 {
	if (hi == 0) && (lo < 2) {
		return lo
	}
	// Start above the root: the float64 estimate (relative error ~2^-52) padded
	// by 2^-40 of itself plus 2, capped at 2^64 - 1 (the largest possible root).
	f := math.Sqrt(float64(hi)*0x1p64 + float64(lo))
	padded := f*(1.0+0x1p-40) + 2.0
	x := ^uint64(0)
	if padded < 0x1p64 {
		x = uint64(padded)
	}
	for {
		// y = (x + r/x) >> 1 on 128 bits (x < 2^64, r/x < 2^128, no overflow).
		// A y of 2^64 or more is >= x, so the iteration stops (as Swift's UInt128).
		qr := divRem_q128r128_128x128(int64(lo), int64(hi), int64(x), 0)
		sLo, c := bits.Add64(uint64(qr.quotDw0), x, 0)
		sHi := uint64(qr.quotDw1) + c
		y := (sLo >> 1) | (sHi << 63)
		if ((sHi >> 1) != 0) || (y >= x) {
			break
		}
		x = y
	}
	return x
}
