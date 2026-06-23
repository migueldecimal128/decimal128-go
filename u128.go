// The U128 magnitude type (PrimitiveLayer.md §4.2). Swift uses the native
// UInt128; Go has none, so — per D-G5 — u128 is a struct of two signed int64
// dwords (dw0 least significant), the Java/Kotlin limb model. It is a pure
// arithmetic magnitude: coefficient/quotient/intermediate are roles layered on
// by usage, not subtypes.

package decimal128

type u128 struct {
	dw0 int64
	dw1 int64
}

// fromLoHi_128_64x64 builds a u128 from its low and high dwords (signed-dword
// uniform-API spelling).
func fromLoHi_128_64x64(dw0, dw1 int64) u128 {
	return u128{dw0: dw0, dw1: dw1}
}

// u128FromU64 builds a u128 from unsigned low/high halves (primitive-interior
// spelling, §3.4).
func u128FromU64(dw0, dw1 uint64) u128 {
	return u128{dw0: int64(dw0), dw1: int64(dw1)}
}
