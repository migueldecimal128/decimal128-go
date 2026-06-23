// Package verify provides the three internal assertion contracts from
// CrossPlatformArchitecture.md §9, identical in behavior across every port and
// spelled in Go's idiom:
//
//   - [Verify]     — compile-gated state assertion (§9.1).
//   - [Demand]     — production runtime guard (§9.2).
//   - [Impossible] — unconditional abort for unreachable code (§9.3).
//
// Bare lower-case call-site parity with the Swift reference (`demand`, `verify`,
// `impossible`) yields to Go's rule that cross-package identifiers must be
// exported (capitalized). This is a sanctioned §9-style per-port divergence — the
// behavior and argument order stay identical; only the spelling differs.
package verify

// Enabled gates [Verify] (§9.1) — the Go analogue of Swift's `#if VERIFY_ENABLED`
// and C's `VERIFY_ENABLED` macro guard. With Enabled == false the compiler
// eliminates the Verify body, and pure condition expressions at the call sites
// become dead code the optimizer drops.
const Enabled = false

// Impossible aborts unconditionally, marking unreachable code (§9.3). It never
// returns; Go spells "does not return" as panic (cf. Swift `-> Never`, C
// `noreturn`). An empty msg renders a default.
func Impossible(msg string) {
	if msg == "" {
		panic("impossible")
	}
	panic(msg)
}

// Demand is a production runtime guard (§9.2): always evaluated, routing through
// [Impossible] on failure. The contract is exactly demand(msg, cond) ≡
// if !cond { impossible(msg) }. Message first, condition second, mirroring the
// Swift reference `demand("msg") { cond }`. An empty msg ("") is the legal way to
// omit a message and renders as "demand failed".
func Demand(msg string, cond bool) {
	if !cond {
		if msg == "" {
			Impossible("demand failed")
		}
		Impossible(msg)
	}
}

// Verify is a compile-gated state assertion (§9.1): it checks cond only when
// [Enabled]. With Enabled == false it is a no-op the compiler removes. Use it
// early and often during development.
func Verify(msg string, cond bool) {
	if Enabled {
		if !cond {
			if msg == "" {
				Impossible("verify failed")
			}
			Impossible(msg)
		}
	}
}
