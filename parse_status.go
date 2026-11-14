// The parse status codes (CrossPlatformArchitecture.md §6). SCREAMING_SNAKE per §6.2.

package decimal128

// parseStatus is the internal parse outcome. Not load-bearing or boundary-
// crossing, so (unlike the SCREAMING boundary enums) it is unexported camelCase.
type parseStatus int

const (
	psEmptyString parseStatus = iota
	psFailureParsingInfinity
	psFailureParsingNan
	psDoubleDot
	psNoCoefficientDigit
	psNoExponentDigit
	psUnexpectedChar
	psInvalidUnderscoreLocation
	psInvalidFormat
	psSuccess
)
