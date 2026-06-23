// Package rosetta is the conformance harness that drives the decimal128 engine
// against the four corpora — IBM/Cowlishaw dectest, IBM FPgen/fptest, Intel
// libbid, and the native corpus — at bitwise-value + exception-flag parity
// (CrossPlatformArchitecture.md §2.4; the canonical harness is decimal128-c's
// rosetta/). It co-evolves with the core: each operator's dispatch arm lights up
// as the operator lands, so coverage and implementation advance together.
//
// As a test harness this code is non-shippable scaffolding, so the §3 constraint
// regime does not bind it (switch/closures/stdlib are used freely). Phase 0
// stands up the case model, the native-corpus parser, and a zero-arm dispatch so
// the skeleton compiles and runs (every case reports Unsupported, zero failures).
package rosetta

import "strings"

// Source selects the corpus dialect — the operand/result encoding and the
// source-specific compare quirks applied at run time.
type Source int

const (
	Dectest Source = iota
	Fptest
	Intel
	Native
)

// Case is one parsed, in-scope conformance case. Operand and result strings are
// kept verbatim; decoding to a decimal value is the dispatch layer's job, done at
// run time because each corpus encodes operands differently. Mirrors the Swift
// RosettaCase + Source and the Rust Case.
type Case struct {
	Source        Source
	Text          string   // source line, for diagnostics / skip-list keys
	Operation     string   // canonical op token, e.g. "d128_add_ctx"
	Operands      []string // verbatim operand tokens
	Result        string   // verbatim expected-result token
	ExpectedFlags string   // canonical xuozi letter string
	Rounding      string   // canonical rounding token, e.g. "TIES_TO_EVEN"
	Traps         string   // fptest trap column (xuozi); empty elsewhere
}

// FlagLettersCanonical returns the xuozi-ordered subset of exception letters
// present in s (inexact, underflow, overflow, divByZero, invalid). Unknown
// characters are ignored. Matches the Rust flag_letters_canonical helper so
// expected/observed flag strings compare by exact string equality.
func FlagLettersCanonical(s string) string {
	var x, u, o, z, i bool
	for _, ch := range s {
		switch ch {
		case 'x':
			x = true
		case 'u':
			u = true
		case 'o':
			o = true
		case 'z':
			z = true
		case 'i':
			i = true
		}
	}
	var b strings.Builder
	if x {
		b.WriteByte('x')
	}
	if u {
		b.WriteByte('u')
	}
	if o {
		b.WriteByte('o')
	}
	if z {
		b.WriteByte('z')
	}
	if i {
		b.WriteByte('i')
	}
	return b.String()
}
