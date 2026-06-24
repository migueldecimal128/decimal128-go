// Core enumerations (CrossPlatformArchitecture.md §6). Exception754's integer
// values are load-bearing bit positions in the accumulated status-flag word.
// SCREAMING_SNAKE per §6.2; Go forces these to be exported (no unexported
// SCREAMING_SNAKE), which keeps the exact cross-port names.

package decimal128

// exception754 is the five IEEE 754-2019 exceptions; the value is the bit
// position within an int64 flag word.
type exception754 = int

const (
	INVALID_OPERATION exception754 = 0
	DIVISION_BY_ZERO  exception754 = 1
	OVERFLOW          exception754 = 2
	UNDERFLOW         exception754 = 3
	INEXACT           exception754 = 4
)

func exc754Mask(e exception754) int64 { return int64(1) << uint(e) }

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

// invalidCause tags an invalid-operation signal (diagnostic; not load-bearing).
// The full set lands with the arithmetic operations in Phase 3; parse needs only
// causeParseMalformed.
type invalidCause int

const (
	causeParseMalformed invalidCause = iota
	causeNanOperand
	causeSnanOperand
	causeMagnitudeSubtractionOfInfinities
	causeMulZeroByInfinity
	causeDivZeroByZero
	causeDivInfByInf
	causeDivByZeroInRemainderOp
	causeInfNumeratorInRemainderOp
	causeDivIntOverflowsCoefficient
	causeConvertNonFiniteToInteger
	causeSquareRootOfNegFiniteNonZero
	causeSquareRootOfNegInfinity
	causeQuantizeExactlyOneOperandIsInfinite
	causeQuantizeResultWouldExceedPrecision
	causeQuantumOfNonFinite
	causeLogOfNegNumber
	causePowerOfNegBaseNonIntegerExponent
	causeOther
)
