// The quot64Rem64 result aggregate, and the rule for all of them
// (CrossPlatformArchitecture.md §5, §4.8; PrimitiveLayer.md §4.2). Several core operations must return more than one
// value; under the Go value-return memory model (D-G4) these are returned by
// value, like Swift/C — no Java/Kotlin fill-in pool, so no destination-first
// rewrite. Fields are role-major, ascending limb order (D23). The residue field
// trails the quotient limbs (the JVM header-gap reorder of §4.8 applies only to
// the JVM ports, not to a Go value type).

package decimal128

type quot64Rem64 struct {
	quotDw0 int64
	remDw0  int64
}
