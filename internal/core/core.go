// Package core is the cross-platform decimal128 engine — the middle of the three
// layers (CrossPlatformArchitecture.md §2.1): alignment, rounding/finalize,
// dispatch, BID/DPD encode/decode, parse, and format, plus the D38
// extended-precision intermediate engine. It depends only on the primitive
// layer.
//
// Conceptually it subdivides into the core/math/print/interchange/finance tiers
// (§2.5); in Go these are semantically-inert groupings of flat files within this
// one package (Go has no sub-package privacy gradient — §2.5.3), matched across
// ports by filename + symbol prefix, not path. The memory model is value-return
// (D-G4): result aggregates and the D128/D38 value types are returned by value,
// no Java/Kotlin fill-in pool.
//
// Populated from Phase 2. Swift's Core target is the primary reference.
package core
