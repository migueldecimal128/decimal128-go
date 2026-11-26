package decimal128

// Result aggregate for trailing-zero-digit stripping (D18): the stripped
// 128-bit magnitude paired with the number of zero digits removed.
//
// Stored as dword limbs (PrimitiveLayer.md D23): the field list -- names,
// types, declaration order (role-major, ascending; non-limb scalars
// last) -- is the uniform cross-platform surface; constructors and typed
// accessors are platform-local spellings.
//
// Cross-platform shape (CrossPlatformArchitecture.md Section 5): Swift and C
// return this struct by value; Java and Kotlin value-return under
// verified escape analysis, with measured fallback to fill-in.
type stripped128StripCount struct {
	strippedDw0 int64
	strippedDw1 int64
	stripCount  int64
}
