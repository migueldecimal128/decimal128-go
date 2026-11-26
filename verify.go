// The three internal assertion contracts (CrossPlatformArchitecture.md Section 9),
// identical in behavior across every port and spelled in each language's idiom:
//   - verify -- compile-gated state assertion (Section 9.1): checked only in a verify
//     build, compiled to nothing otherwise.
//   - demand -- production runtime guard (Section 9.2): always evaluated; routes
//     through impossible on failure.
//   - impossible -- unconditional abort for unreachable code (Section 9.3); it does
//     not return.
// The empty string "" is the legal way to omit a message (the regime prohibits
// defaulting it).
//
// In Go: in the single-package layout they are unexported, so the bare Swift call-site
// names (verify/demand/impossible) are preserved exactly. verifyEnabled gates verify;
// its value is set by the build tag in verify_off.go / verify_on.go: false by default
// (the compiler eliminates the verify body and the dead condition expressions), true
// under -tags verify.

package decimal128

// Compile-gated state assertion (CrossPlatformArchitecture.md Section 9.1):
// the condition is evaluated and checked only in a verify build; otherwise
// this is a no-op and the condition is never evaluated. An empty msg renders
// as "verify failed".
func verify(msg string, cond func() bool) {
	// cond is a thunk (func() bool), not a bare bool, so that with verifyEnabled ==
	// false the predicate is NEVER evaluated: the inlined body is dead and the
	// never-called closure -- together with everything it computes -- is
	// dead-code-eliminated (no call, no allocation). Go has no @autoclosure (Swift)
	// or macro (C), so the deferral must be spelled at the call site as
	// verify("msg", func() bool { return ... }). Passing a bare bool would evaluate
	// the predicate eagerly even when disabled; for non-inlinable predicates
	// (digitLen/cmp table lookups, whose bounds checks block elimination) that cost
	// survives into production builds -- the reason this is a thunk.
	if verifyEnabled {
		if !cond() {
			if msg == "" {
				impossible("verify failed")
			}
			impossible(msg)
		}
	}
}

// Production runtime guard (CrossPlatformArchitecture.md Section 9.2): always
// evaluated; aborts through impossible when cond is false. Message first,
// condition second. An empty msg renders as "demand failed".
func demand(msg string, cond bool) {
	if !cond {
		if msg == "" {
			impossible("demand failed")
		}
		impossible(msg)
	}
}

// Unconditional abort marking unreachable code (CrossPlatformArchitecture.md
// Section 9.3); never returns.
func impossible(msg string) {
	// Go spells "does not return" as panic.
	if msg == "" {
		panic("impossible")
	}
	panic(msg)
}
