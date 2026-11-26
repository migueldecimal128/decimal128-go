package decimal128

// Division result aggregate: quotient magnitude paired with remainder.
//
// Stored as dword limbs (PrimitiveLayer.md D23): the field list -- names,
// types, declaration order (role-major, ascending) -- is the uniform
// cross-platform surface; constructors and typed accessors are
// platform-local spellings.
//
// Cross-platform shape (CrossPlatformArchitecture.md Section 5): Swift and C
// return this struct by value; Java and Kotlin fill a caller-supplied mutable
// instance (fill-in, PrimitiveLayer.md Section 6.4).
type quot128Rem64 struct {
	quotDw0 int64
	quotDw1 int64
	remDw0  int64
}
