// The rounding direction selector and the rounding-decision logic. Round is a
// plain selector (code 0..4); the decision functions are branchless packed-table
// lookups. needsRoundUp maps the signed-space direction into magnitude space
// (forMagnitude) and reads the ulp-bias table.

package decimal128

type round struct{ code int }

var (
	TIES_TO_EVEN    = round{0}
	TIES_TO_AWAY    = round{1}
	TOWARD_ZERO     = round{2}
	TOWARD_POSITIVE = round{3}
	TOWARD_NEGATIVE = round{4}
)

func roundName(r round) string {
	if r == TIES_TO_EVEN {
		return "tiesToEven"
	}
	if r == TIES_TO_AWAY {
		return "tiesToAway"
	}
	if r == TOWARD_ZERO {
		return "towardZero"
	}
	if r == TOWARD_POSITIVE {
		return "towardPositive"
	}
	if r == TOWARD_NEGATIVE {
		return "towardNegative"
	}
	return "<< INVALID Round >>"
}

const ulpBiasMap int64 = 0b0_00000000_00001110_00000000_00001100_00001000

func needsRoundUp(rounding round, sign, isOdd bool, res residue) bool {
	return ulpBias(forMagnitude(rounding, sign), isOdd, res) != 0
}

func ulpBias(rounding round, isOdd bool, res residue) int64 {
	verify("", res&^3 == 0)
	biasMapEvenOdd := ulpBiasMap | (boolToInt64(isOdd) << 2)
	bitIndex := (rounding.code << 3) + res
	return (biasMapEvenOdd >> uint(bitIndex)) & 1
}

// forMagnitude maps a signed-space rounding direction into magnitude space: the
// two directional rules (toward ±) swap on a negative sign; the three
// sign-symmetric rules are unchanged. Branchless via a packed nybble map.
func forMagnitude(rounding round, sign bool) round {
	magnitudeMap := int64(0x0003421000043210)
	index := (signBitOf(sign) << 3) | rounding.code
	shift := index << 2
	result := int((magnitudeMap >> uint(shift)) & 0xF)
	return round{result}
}

// roundingOverflowsToInfinity reports whether overflow under this direction
// yields infinity (vs the largest finite value).
func roundingOverflowsToInfinity(rounding round, sign bool) bool {
	table := 0x21033
	shift := (rounding.code << 2) + signBitOf(sign)
	return ((table >> uint(shift)) & 1) != 0
}

// roundingUnderflowsToZero reports whether underflow under this direction
// collapses to zero (vs the smallest subnormal magnitude).
func roundingUnderflowsToZero(rounding round, sign bool) bool {
	table := 0x12333
	shift := (rounding.code << 2) + signBitOf(sign)
	return ((table >> uint(shift)) & 1) != 0
}
