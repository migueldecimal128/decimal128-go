// Textual-rendering styles for Decimal128.Text.

package decimal128

// Style is a textual-rendering style for `Decimal128.Text`.
//
// The wrapper face of the core's `FORMAT_*` codes (CrossPlatformArchitecture.md
// Section 6.4, where the concept is named `FormatStyle`); the raw values mirror
// the core ordering.
type Style int

const (
	// StyleAutomatic: GDAS "to-scientific-string": plain decimal when the adjusted
	// exponent allows, scientific otherwise. This is what `String` uses.
	StyleAutomatic Style = iota
	// StyleExponential: always exponential notation.
	StyleExponential
	// StyleEngineering: GDAS "to-engineering-string": exponent constrained to a
	// multiple of 3.
	StyleEngineering
	// StyleCoefficientExponent: raw cohort form, `coefficient * 10^qExp` --
	// preserves exact internal state.
	StyleCoefficientExponent
)

// String returns a readable name for the style.
func (s Style) String() string {
	switch s {
	case StyleAutomatic:
		return "Automatic"
	case StyleExponential:
		return "Exponential"
	case StyleEngineering:
		return "Engineering"
	case StyleCoefficientExponent:
		return "CoefficientExponent"
	default:
		return "Style(?)"
	}
}

// Text renders this value in the given style.
func (d Decimal128) Text(style Style) string {
	return d128_format(d.core, int(style))
}
