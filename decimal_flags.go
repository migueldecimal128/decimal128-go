// The public IEEE 754-2019 exception-flag set, a value snapshot read from a
// Context. The bit positions are identical to the core's exception754 ordering,
// so a snapshot crosses the wrapper/core boundary as a single int with no
// translation.

package decimal128

import "strings"

// Flags is the set of IEEE 754-2019 exceptions raised by an operation, read as a
// snapshot from a Context. Combine and test with the bitwise operators or the
// Has method.
type Flags int

const (
	// FlagInvalidOperation: an operand or operation was mathematically undefined
	// (e.g. 0/0, ∞ − ∞, a signaling-NaN operand).
	FlagInvalidOperation Flags = 1 << 0
	// FlagDivisionByZero: a finite nonzero value was divided by zero.
	FlagDivisionByZero Flags = 1 << 1
	// FlagOverflow: the result's magnitude exceeded the largest finite value.
	FlagOverflow Flags = 1 << 2
	// FlagUnderflow: the result was tiny and lost precision to subnormal form.
	FlagUnderflow Flags = 1 << 3
	// FlagInexact: the result had to be rounded; it is not exactly the true value.
	FlagInexact Flags = 1 << 4
)

// Has reports whether all of the flags in mask are set in f.
func (f Flags) Has(mask Flags) bool { return f&mask == mask }

// IsEmpty reports whether no exception flags are set.
func (f Flags) IsEmpty() bool { return f == 0 }

// String renders the set flags as a brace-enclosed, comma-separated list (e.g.
// "{Inexact, Rounded}" style), or "{}" when empty.
func (f Flags) String() string {
	if f == 0 {
		return "{}"
	}
	var parts []string
	if f.Has(FlagInvalidOperation) {
		parts = append(parts, "InvalidOperation")
	}
	if f.Has(FlagDivisionByZero) {
		parts = append(parts, "DivisionByZero")
	}
	if f.Has(FlagOverflow) {
		parts = append(parts, "Overflow")
	}
	if f.Has(FlagUnderflow) {
		parts = append(parts, "Underflow")
	}
	if f.Has(FlagInexact) {
		parts = append(parts, "Inexact")
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
