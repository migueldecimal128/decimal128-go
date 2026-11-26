package decimal128

// Result aggregate for the 128-bit subAbs form (PrimitiveLayer.md D2, D14):
// the absolute difference |x - y| paired with a flag reporting whether the
// operands were swapped (i.e. y > x).
//
// Stored as dword limbs (PrimitiveLayer.md D23): the field list -- names,
// types, declaration order (role-major, ascending; non-limb scalars
// last) -- is the uniform cross-platform surface; constructors and typed
// accessors are platform-local spellings.
//
// Cross-platform shape (CrossPlatformArchitecture.md Section 5): Swift and C
// return this struct by value; Java and Kotlin value-return under
// verified escape analysis, with measured fallback to fill-in.
type diff128Swap struct {
	diffDw0 int64
	diffDw1 int64
	swap    bool
}
