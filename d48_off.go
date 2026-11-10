//go:build !d48

package decimal128

// useD48 is false by default: the transcendentals stay on D38 until the D48
// switch (D48GoWorkOrder.md G-14, Phase 4). Build with `-tags d48` (d48_on.go)
// to run them on D48; the dead branch is removed at compile time.
const useD48 = false
