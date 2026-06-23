// Package primitive is the hand-tuned, fixed-width unsigned-arithmetic layer of
// decimal128 — the lowest of the three layers (CrossPlatformArchitecture.md
// §2.1): U128/U256 magnitudes, the divPow10 kernels (Barrett, reciprocal-mul,
// RRMP10, Möller–Granlund), mulPow10, and the pow10 tables. It is not
// decimal-specific.
//
// Per D-G2 (GO_PORT_PLAN.md) the public dword fields and signatures are signed
// int64 for cross-port field parity; bodies cast to uint64 — using math/bits
// (Add64/Sub64/Mul64/Div64/LeadingZeros64) — only where unsigned semantics are
// required. Synthesizing 128/256-bit arithmetic from 64-bit limbs follows the
// Java/Kotlin model (Go has no native uint128), but with the hardware
// carry/wide-mul intrinsics the JVM lacks.
//
// Populated in Phase 1. The Swift Primitive target is the primary reference; the
// Java/Kotlin primitive layers are the secondary reference for limb synthesis.
package primitive
