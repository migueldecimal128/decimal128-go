// Package decimal128 is a native Go implementation of IEEE 754-2019 decimal128
// — the public wrapper layer (CrossPlatformArchitecture.md §2.1) over an engine
// that lives under internal/. It is a native port, not a binding over the C core.
//
// The public Decimal128 value type, its methods, the Context/Rounding surface,
// and the Finance facade are added in Phase 5 (GO_PORT_PLAN.md). The Swift port
// is the primary reference; Rust/C/Java/Kotlin are secondary.
package decimal128
