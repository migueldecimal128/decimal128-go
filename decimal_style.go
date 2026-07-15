// Textual-rendering styles for Decimal128.Text.

package decimal128

// Style is a textual-rendering style for Decimal128.Text. (Named Style rather
// than FormatStyle; its values mirror the core FORMAT_* ordering.)
type Style int

const (
	// StyleAutomatic is the GDAS "to-scientific-string": plain decimal when the
	// adjusted exponent allows, scientific otherwise. This is what String uses.
	StyleAutomatic Style = iota
	// StyleExponential always uses exponential notation.
	StyleExponential
	// StyleEngineering is the GDAS "to-engineering-string": the exponent is
	// constrained to a multiple of 3.
	StyleEngineering
	// StyleCoefficientExponent is the raw cohort form coefficient × 10^qExp,
	// preserving the exact internal state.
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

// Text renders d in the given style. (The no-argument String method uses
// StyleAutomatic.)
func (d Decimal128) Text(style Style) string {
	return d128_format(d.core, int(style))
}
