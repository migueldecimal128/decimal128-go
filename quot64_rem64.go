package decimal128

// Division result aggregate: 64-bit quotient paired with 64-bit remainder.
//
// Stored as dword limbs (PrimitiveLayer.md D23): the field list -- names,
// types, declaration order (role-major, ascending) -- is the uniform
// cross-platform surface; constructors and typed accessors are
// platform-local spellings.
//
// Cross-platform shape (CrossPlatformArchitecture.md Section 5): Swift and C
// return this struct by value. The JVM ports have no Quot64Rem64: they use
// Quot128Rem64 for the 64-bit-quotient step.
type quot64Rem64 struct {
	quotDw0 int64
	remDw0  int64
}
