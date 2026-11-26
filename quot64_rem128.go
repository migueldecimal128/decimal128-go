package decimal128

// Division result aggregate: 64-bit quotient paired with 128-bit remainder.
//
// Produced by the Moller-Granlund DIV3BY2 step
// (`divRem_q64r128_192x128x64_v_mg`): one quotient word, a < d remainder that
// can occupy the full 128 bits of the two-limb divisor.
//
// Stored as dword limbs (PrimitiveLayer.md D23): the field list -- names,
// types, declaration order (role-major, ascending) -- is the uniform
// cross-platform surface; constructors and typed accessors are
// platform-local spellings.
//
// Cross-platform shape (CrossPlatformArchitecture.md Section 5): Swift and C
// return this struct by value; Java and Kotlin fill a caller-supplied mutable
// instance (fill-in, PrimitiveLayer.md Section 6.4).
type quot64Rem128 struct {
	quotDw0 int64
	remDw0  int64
	remDw1  int64
}
