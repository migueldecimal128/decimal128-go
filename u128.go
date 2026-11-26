// U128 -- a pure 128-bit arithmetic magnitude (PrimitiveLayer.md Section 4.2). No
// role-specific meaning: coefficient, quotient and intermediate are roles layered on by
// usage, not subtypes. Core sees only the signed dword surface dw0/dw1, dw0 least
// significant (D23); the arithmetic vocabulary over this type lives in U128Arith.
//
// In Go: there is no native 128-bit type, so (D-G5) u128 is a struct of two signed
// int64 dwords, the Java/Kotlin limb model.

package decimal128

type u128 struct {
	dw0 int64
	dw1 int64
}

// u128FromU64 builds a u128 from unsigned low/high halves (primitive-interior
// spelling, §3.4).
func u128FromU64(dw0, dw1 uint64) u128 {
	return u128{dw0: int64(dw0), dw1: int64(dw1)}
}
