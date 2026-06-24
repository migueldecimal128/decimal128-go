// Package decimal128 is a native Go implementation of IEEE 754-2019 decimal128.
//
// It is a native port (not a binding over the C core), structured per
// CrossPlatformArchitecture.md. Go has no Swift-style `package` access level —
// an identifier is either unexported (package-private) or exported — so the
// three architectural layers live in this single package rather than separate
// Go packages (D-G1, GO_PORT_PLAN.md):
//
//   - Primitive layer — unexported, files prefixed by their kernel (tier0.go,
//     u128*.go, u256*.go, div_pow10*.go, …). Dwords are signed int64 for
//     cross-port field parity (D-G2); bodies cast to uint64 (math/bits) only
//     where unsigned semantics are required.
//   - Core layer — unexported, files prefixed d128_* / d38_* / finalize.go etc.
//   - Wrapper layer — the exported public surface: the Decimal128 value type and
//     its methods, Context, Rounding, Finance.
//
// Unexported engine identifiers keep the exact snake_case names the other ports
// use (add_128_128x128, divPow10_q128res_256_recipmul, d128_add_ctx), so the
// cross-port locator is filename + symbol, unchanged. The Rosetta conformance
// harness lives in white-box _test.go files so it can dispatch to the unexported
// engine, mirroring the Rust in-crate harness.
//
// Reference precedence: Swift (decimal128-swift) is primary; Rust/C/Java/Kotlin
// secondary. The primitive-layer limb synthesis follows the Java/Kotlin model.
package decimal128
