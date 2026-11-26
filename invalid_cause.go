// The cause carried with a signaled INVALID_OPERATION: a diagnostic tag that lets
// callers (and the test harness) tell causes apart. The engine sets the
// INVALID_OPERATION flag regardless of the cause; the integer values are not
// load-bearing (no wire or bit encoding depends on them).

package decimal128

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
	causeRootnBadArgs
	causeCompoundXLtNegOne
	causeOther
)
