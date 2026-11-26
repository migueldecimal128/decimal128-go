// Accumulating set of IEEE 754 exception flags: a packed int64 bit word, bit positions
// per Exception754. Contexts derived from a common parent share a single flag sink, so
// mutations through any of them are visible through all, which is the intended
// behavior for chained calculations under varying rounding modes. Flags performs no
// internal synchronization: a Context (and therefore its Flags) is for use within a
// single task, so with concurrency, construct one context per task.
//
// In Go: a pointer type.

package decimal128

import "strings"

type flags struct {
	bits int64
}

func newFlags() *flags { return &flags{} }

func (f *flags) set(exception exception754)   { f.bits |= exc754Mask(exception) }
func (f *flags) clear(exception exception754) { f.bits &^= exc754Mask(exception) }
func (f *flags) clearAll()                    { f.bits = 0 }
func (f *flags) isSet(exception exception754) bool {
	return (f.bits & exc754Mask(exception)) != 0
}

// Raw flag word. Provided for fast "any flags set?" checks
// (`flags.raw() == 0`) and for serialization in test harnesses.
func (f *flags) raw() int64 { return f.bits }

// Canonical letter rendering of the signaled exceptions.
//
// This is *the* standard flag representation across all three test
// suites (fptest, Intel, dectest) -- fptest's letter string, promoted to
// the lingua franca. Maps each flag to its single letter:
//
//	`x` = inexact, `u` = underflow, `o` = overflow,
//	`z` = divisionByZero, `i` = invalidOperation
//
// Returned in `xuozi` order regardless of which combination is set.
func (f *flags) exceptionLetters() string {
	var b strings.Builder
	if f.isSet(const_INEXACT) {
		b.WriteByte('x')
	}
	if f.isSet(const_UNDERFLOW) {
		b.WriteByte('u')
	}
	if f.isSet(const_OVERFLOW) {
		b.WriteByte('o')
	}
	if f.isSet(const_DIVISION_BY_ZERO) {
		b.WriteByte('z')
	}
	if f.isSet(const_INVALID_OPERATION) {
		b.WriteByte('i')
	}
	return b.String()
}
