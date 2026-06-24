//go:build !verify

package decimal128

// verifyEnabled is false by default: verify (§9.1) compiles to nothing and the
// pure condition expressions at call sites become dead code the optimizer drops.
// Build with `-tags verify` (verify_on.go) to arm the assertions.
const verifyEnabled = false
