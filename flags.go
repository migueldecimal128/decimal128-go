// The accumulating IEEE 754 exception-flag sink. A pointer type so contexts
// derived from a common parent share one sink (mutations visible through all).
// Not safe for concurrent use — one context per task.

package decimal128

import "strings"

type flags struct {
	bits int64
}

func newFlags() *flags { return &flags{} }

func (f *flags) set(e exception754)   { f.bits |= exc754Mask(e) }
func (f *flags) clear(e exception754) { f.bits &^= exc754Mask(e) }
func (f *flags) clearAll()            { f.bits = 0 }
func (f *flags) isSet(e exception754) bool {
	return (f.bits & exc754Mask(e)) != 0
}
func (f *flags) raw() int64 { return f.bits }

// exceptionLetters renders the signaled exceptions in canonical xuozi order
// (inexact, underflow, overflow, divByZero, invalid) — the lingua franca of the
// three test corpora.
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
