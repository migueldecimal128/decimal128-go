// The three internal assertion contracts (CrossPlatformArchitecture.md §9),
// identical in behavior across every port. In the single-package Go layout they
// are unexported, so the bare Swift call-site names (verify/demand/impossible)
// are preserved exactly.

package decimal128

// verifyEnabled gates verify (§9.1) — the Go analogue of Swift's
// #if VERIFY_ENABLED. Its value is set by the build tag in verify_off.go /
// verify_on.go: false by default (the compiler eliminates the verify body and
// the dead condition expressions), true under `-tags verify`. See those files.

// impossible aborts unconditionally, marking unreachable code (§9.3). It never
// returns; Go spells "does not return" as panic. An empty msg renders a default.
func impossible(msg string) {
	if msg == "" {
		panic("impossible")
	}
	panic(msg)
}

// demand is a production runtime guard (§9.2): always evaluated, routing through
// impossible on failure. The contract is exactly demand(msg, cond) ≡
// if !cond { impossible(msg) }. Message first, condition second, mirroring the
// Swift reference demand("msg") { cond }. An empty msg ("") legally omits the
// message and renders "demand failed".
func demand(msg string, cond bool) {
	if !cond {
		if msg == "" {
			impossible("demand failed")
		}
		impossible(msg)
	}
}

// verify is a compile-gated state assertion (§9.1): it checks cond only when
// verifyEnabled. With verifyEnabled == false it is a no-op the compiler removes.
func verify(msg string, cond bool) {
	if verifyEnabled {
		if !cond {
			if msg == "" {
				impossible("verify failed")
			}
			impossible(msg)
		}
	}
}
