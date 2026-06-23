// Double bridges for the square-root seed (Core MagnitudeSqrt). Decimal-agnostic
// bit math: render a 256-bit unsigned integer down to a float64 (floor of its top
// 53 bits) so hardware sqrt produces a ~15-digit seed, then widen that seed back
// into a 128-bit unsigned integer.

package decimal128

import "math"

// scalb2 returns x * 2^n, building the power of two directly (no libm). n must
// keep n+1023 inside the float64 exponent field, which holds for every U256 bit
// index.
func scalb2(x float64, n int) float64 {
	return x * math.Float64frombits(uint64(n+1023)<<52)
}

// u256ToFloorDouble renders the top (at most) 53 bits of x scaled by their bit
// position — x floored to float64 precision. Used only to seed Newton iteration.
func u256ToFloorDouble(x u256) float64 {
	bitLen := bitLen_int_256(x.dw0, x.dw1, x.dw2, x.dw3)
	if bitLen == 0 {
		return 0.0
	}
	hiBitLen := min(53, bitLen)
	hiBitIndex := bitLen - hiBitLen
	topBits := shr_256_256(x.dw0, x.dw1, x.dw2, x.dw3, hiBitIndex)
	hiBits := float64(uint64(topBits.dw0))
	return scalb2(hiBits, hiBitIndex)
}

// u128FromDouble widens a finite, non-negative float64's integer value to a
// u128, truncating any fraction.
func u128FromDouble(d float64) u128 {
	b := math.Float64bits(d)
	exp := int((b>>52)&0x7FF) - 1023
	if exp < 0 {
		return u128{0, 0}
	}
	significand := (b & ((uint64(1) << 52) - 1)) | (uint64(1) << 52)
	if exp <= 52 {
		return fromLoHi_128_64x64(int64(significand>>uint(52-exp)), 0)
	}
	return shl_128_128(int64(significand), 0, exp-52)
}
