// Tier 0 (PrimitiveLayer.md §3.2): the irreducible scalar operations. Dword
// operands cross the uniform API as signed int64 (the JVM `long` contract,
// D-G2); the `u` op prefix marks the unsigned semantics the bodies apply by
// casting to uint64. Where a scalar maps to a math/bits intrinsic (Mul64,
// LeadingZeros64) the body uses it; otherwise it casts and uses native uint64
// operators.

package decimal128

import "math/bits"

// umulHi_64_64x64 returns the high 64 bits of the unsigned 64×64 product.
func umulHi_64_64x64(a, b int64) int64 {
	hi, _ := bits.Mul64(uint64(a), uint64(b))
	return int64(hi)
}

// udiv_64_64x64 returns the unsigned 64÷64 quotient.
func udiv_64_64x64(a, b int64) int64 {
	return int64(uint64(a) / uint64(b))
}

// urem_64_64x64 returns the unsigned 64÷64 remainder.
func urem_64_64x64(a, b int64) int64 {
	return int64(uint64(a) % uint64(b))
}

// ucmp_int_64x64 returns -1/0/+1 from an unsigned compare.
func ucmp_int_64x64(a, b int64) int {
	ua := uint64(a)
	ub := uint64(b)
	if ua < ub {
		return -1
	}
	if ua == ub {
		return 0
	}
	return 1
}

// clz_int_64 returns the leading-zero count; clz_int_64(0) == 64.
func clz_int_64(a int64) int {
	return bits.LeadingZeros64(uint64(a))
}

// ushr_64_64 is a logical right shift; n in 0..<64.
func ushr_64_64(a int64, n int) int64 {
	verify("", n >= 0 && n < 64)
	return int64(uint64(a) >> uint(n))
}

// hi32_64 returns the high 32 bits of a dword, zero-extended (value in 0..<32).
func hi32_64(dw int64) int64 {
	return int64(uint64(dw) >> 32)
}

// lo32_64 returns the low 32 bits of a dword, zero-extended.
func lo32_64(dw int64) int64 {
	return int64(uint64(dw) & 0xFFFF_FFFF)
}
