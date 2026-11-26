package decimal128

// Result aggregate for the `div_*_pow10_q128res_*` family: a 128-bit
// quotient magnitude paired with the `Residue` categorizing the digits
// shifted out to the right (the remainder).
//
// Stored as dword limbs (PrimitiveLayer.md D23): the field list -- names,
// types, declaration order (role-major, ascending; non-limb scalars
// last) -- is the uniform cross-platform surface; constructors and typed
// accessors are platform-local spellings.
//
// Primitive-owned by the D11 shape rule since D17 made `Residue` a
// 2-bit int: every field is a regime scalar. Cross-platform shape
// (CrossPlatformArchitecture.md Section 5): Swift and C return this struct by
// value; Java and Kotlin value-return under verified escape analysis,
// with measured fallback to fill-in.
type quot128Residue struct {
	quotDw0 int64
	quotDw1 int64
	residue residue
}
