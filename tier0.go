// Tier 0 (PrimitiveLayer.md Section 3.2): the irreducible scalar operations. Dword
// operands cross the uniform API as signed int64 (the JVM's 64-bit contract); the u op
// prefix marks the unsigned semantics the bodies apply via bit-pattern
// reinterpretation (D11).
//
// In Go: where a scalar maps to a math/bits intrinsic (Mul64, LeadingZeros64) the body
// uses it; otherwise it casts to uint64 and uses native operators (D-G2).

package decimal128

import "math/bits"

// High 64 bits of the unsigned 64*64 product.
func umulHi_64_64x64(a, b int64) int64 {
	hi, _ := bits.Mul64(uint64(a), uint64(b))
	return int64(hi)
}

// Unsigned 64/64 quotient.
func udiv_64_64x64(a, b int64) int64 {
	return int64(uint64(a) / uint64(b))
}

// Unsigned 64/64 remainder.
func urem_64_64x64(a, b int64) int64 {
	return int64(uint64(a) % uint64(b))
}

// -1/0/+1 unsigned compare.
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

// Leading zero count; `clz_int_64(0) == 64`.
func clz_int_64(a int64) int {
	return bits.LeadingZeros64(uint64(a))
}

// Logical right shift; `n` in `0..<64`.
func ushr_64_64(a int64, n int) int64 {
	verify("", func() bool { return n >= 0 && n < 64 })
	return int64(uint64(a) >> uint(n))
}

// High 32 bits of a dword, zero-extended (the value occupies bits 0..<32).
func hi32_64(dw int64) int64 {
	return int64(uint64(dw) >> 32)
}

// Low 32 bits of a dword, zero-extended.
func lo32_64(dw int64) int64 {
	return int64(uint64(dw) & 0xFFFF_FFFF)
}
