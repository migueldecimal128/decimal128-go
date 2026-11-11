// The rounding direction selector and the rounding-decision logic. Round is a
// plain selector (code 0..4); the decision functions are branchless packed-table
// lookups. needsRoundUp maps the signed-space direction into magnitude space
// (forMagnitude) and reads the ulp-bias table.

package decimal128

// round is a C-style int-backed selector: the constants are true compile-time
// constants (immutable, foldable immediates), and the named type keeps the
// selector out of plain-int positions. The values are load-bearing: they index
// the packed bit-tables below (§6.4), so do not reorder.
type round int

const (
	const_ROUND_TIES_TO_EVEN    round = 0
	const_ROUND_TIES_TO_AWAY    round = 1
	const_ROUND_TOWARD_ZERO     round = 2
	const_ROUND_TOWARD_POSITIVE round = 3
	const_ROUND_TOWARD_NEGATIVE round = 4
)

func roundName(r round) string {
	if r == const_ROUND_TIES_TO_EVEN {
		return "tiesToEven"
	}
	if r == const_ROUND_TIES_TO_AWAY {
		return "tiesToAway"
	}
	if r == const_ROUND_TOWARD_ZERO {
		return "towardZero"
	}
	if r == const_ROUND_TOWARD_POSITIVE {
		return "towardPositive"
	}
	if r == const_ROUND_TOWARD_NEGATIVE {
		return "towardNegative"
	}
	return "<< INVALID Round >>"
}

const const_ULP_BIAS_MAP int64 = 0b0_00000000_00001110_00000000_00001100_00001000

func needsRoundUp(rounding round, sign, isOdd bool, res residue) bool {
	return ulpBias(forMagnitude(rounding, sign), isOdd, res) != 0
}

func ulpBias(rounding round, isOdd bool, res residue) int64 {
	verify("", func() bool { return res&^3 == 0 })
	biasMapEvenOdd := const_ULP_BIAS_MAP | (boolToInt64(isOdd) << 2)
	bitIndex := (int(rounding) << 3) + res
	return (biasMapEvenOdd >> uint(bitIndex)) & 1
}

// forMagnitude maps a signed-space rounding direction into magnitude space: the
// two directional rules (toward ±) swap on a negative sign; the three
// sign-symmetric rules are unchanged. Branchless via a packed nybble map.
func forMagnitude(rounding round, sign bool) round {
	magnitudeMap := int64(0x0003421000043210)
	index := (signBitOf(sign) << 3) | int(rounding)
	shift := index << 2
	result := int((magnitudeMap >> uint(shift)) & 0xF)
	return round(result)
}

// roundingOverflowsToInfinity reports whether overflow under this direction
// yields infinity (vs the largest finite value).
func roundingOverflowsToInfinity(rounding round, sign bool) bool {
	table := 0x21033
	shift := (int(rounding) << 2) + signBitOf(sign)
	return ((table >> uint(shift)) & 1) != 0
}

// roundingUnderflowsToZero reports whether underflow under this direction
// collapses to zero (vs the smallest subnormal magnitude).
func roundingUnderflowsToZero(rounding round, sign bool) bool {
	table := 0x12333
	shift := (int(rounding) << 2) + signBitOf(sign)
	return ((table >> uint(shift)) & 1) != 0
}
