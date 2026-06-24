//go:build verify

package decimal128

// verifyEnabled is true under `-tags verify`: every verify (§9.1) state
// assertion is evaluated, routing through impossible on failure. Used by CI to
// run the full conformance suite with the internal invariants armed.
const verifyEnabled = true
