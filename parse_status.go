// Internal status returned by the numeric parser, mapped to an InvalidCause / result
// by the caller. A diagnostic tag only; the integer values are not load-bearing.
//
// In Go: not load-bearing or boundary-crossing, so (unlike the SCREAMING boundary
// enums) it is unexported camelCase.

package decimal128

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
