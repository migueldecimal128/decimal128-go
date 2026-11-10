//go:build d48

package decimal128

// useD48 is true under `-tags d48`: the transcendentals (exp, exp10, ln, log10,
// pow, pown, rootn) run on D48 (D48GoWorkOrder.md G-14). Temporary: Phase 4
// deletes this file, d48_off.go and every D38 branch.
const useD48 = true
