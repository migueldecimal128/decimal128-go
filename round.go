// The five IEEE 754 rounding directions plus two GDAS extensions (round-up and
// round-05up), and the rounding decisions as branchless packed-table lookups:
// needsRoundUp (round/sticky -> up?), the magnitude-space direction remap
// (forMagnitude), and the overflow/underflow collapse predicates. The direction codes
// are load-bearing: they index the packed bit-tables (CrossPlatformArchitecture.md
// Section 6.4), so do not reorder.
//
// Round-05up never reaches the tables: each operation's entry sends it to a cold
// route that rounds toward zero and then fixes the last digit (round05_up.go), so
// the predicates below accept codes 0..5 only.
//
// In Go: a typed int constant: the codes are true compile-time constants, and the
// named type keeps the selector out of plain-int positions.

package decimal128

type round int

const (
	const_ROUND_TIES_TO_EVEN    round = 0
	const_ROUND_TIES_TO_AWAY    round = 1
	const_ROUND_TOWARD_ZERO     round = 2
	const_ROUND_TOWARD_POSITIVE round = 3
	const_ROUND_TOWARD_NEGATIVE round = 4
	// GDAS round-up: away from zero whenever any discarded digit is nonzero.
	const_ROUND_AWAY_FROM_ZERO round = 5
	// GDAS round-05up: away from zero only when the kept last digit is 0 or 5.
	const_ROUND_05_UP round = 6
)

func (r round) description() string {
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
	if r == const_ROUND_AWAY_FROM_ZERO {
		return "awayFromZero"
	}
	if r == const_ROUND_05_UP {
		return "round05Up"
	}
	return "<< INVALID Round >>"
}

// Packed bias table for needsRoundUp, eight bits per rounding code, indexed by
// (code << 3) + residue, with the isOdd bit OR'd into bit 2 of each group:
//
//	code 0 (TIES_TO_EVEN)     0x08
//	code 1 (TIES_TO_AWAY)     0x0C
//	code 2 (TOWARD_ZERO)      0x00
//	code 3 (TOWARD_POSITIVE)  0x0E
//	code 4 (TOWARD_NEGATIVE)  0x00
//	code 5 (AWAY_FROM_ZERO)   0x0E
//
// ROUND_05_UP (code 6) has no entry: it never reaches the rounding engine. Each
// operation's entry sends it to a cold route that rounds toward zero and then
// fixes the last digit (round05_up.go).
const const_ULP_BIAS_MAP int64 = 0b0_00001110_00000000_00001110_00000000_00001100_00001000

// Whether a truncated magnitude rounds up under this direction, given the
// result sign, the parity of the retained low digit, and the discarded
// residue.
func needsRoundUp(rounding round, sign, isOdd bool, res residue) bool {
	verify("", func() bool { return rounding != const_ROUND_05_UP })
	return ulpBias(forMagnitude(rounding, sign), isOdd, res) != 0
}

// Bias for the magnitude-mapped rounding direction: 1 if the truncated value
// rounds up, 0 otherwise.
func ulpBias(roundingCode round, isOdd bool, res residue) int64 {
	verify("", func() bool { return res&^3 == 0 })
	biasMapEvenOdd := const_ULP_BIAS_MAP | (boolToInt64(isOdd) << 2)
	bitIndex := (int(roundingCode) << 3) + res
	return (biasMapEvenOdd >> uint(bitIndex)) & 1
}

// forMagnitude returns the rounding-direction code to apply when operating on the
// magnitude of a signed value with this rounding direction.
//
// For sign-magnitude representations, arithmetic is performed on the
// unsigned magnitude with the sign reattached at the end. This requires
// mapping the original (signed-space) rounding direction to an equivalent
// direction in magnitude space:
//
// | Direction          | Positive sign | Negative sign     |
// |--------------------|---------------|-------------------|
// | `tiesToEven`       | unchanged     | unchanged         |
// | `tiesToAway`       | unchanged     | unchanged         |
// | `towardZero`       | unchanged     | unchanged         |
// | `towardPositive`   | unchanged     | `towardNegative`  |
// | `towardNegative`   | unchanged     | `towardPositive`  |
// | `awayFromZero`     | unchanged     | unchanged         |
//
// Rounding toward +inf on a negative value moves toward zero in magnitude
// space -- equivalent to rounding toward -inf on the magnitude. The two
// directional rules swap; the sign-symmetric rules are unchanged.
//
// Operates on raw codes (in and out): the result immediately indexes the
// ulp-bias table in `ulpBias`, so composing in `Int` space avoids an
// enum round-trip in the hot path.
//
// ## Encoding
//
// Branchless via a packed lookup. The map holds 4-bit nybbles indexed by
// `(sign ? 8 : 0) | code`:
//
//	Positive sign (indices 0..5):  identity map (low 32 bits)
//	Negative sign (indices 8..13): swap of 3 and 4 (high 32 bits)
//
// - Parameter sign: `true` if the operand is negative, `false` if positive.
func forMagnitude(roundingCode round, sign bool) round {
	magnitudeMap := int64(0x0053421000543210)
	signBits := 0
	if sign {
		signBits = 8
	}
	index := signBits | int(roundingCode)
	shift := index << 2
	result := int((magnitudeMap >> uint(shift)) & 0xF)
	return round(result)
}

// Whether overflow under this rounding direction produces infinity
// (vs. the largest representable finite value).
//
// | Rounding direction | Result    |
// |--------------------|-----------|
// | `tiesToEven`       | `true`    |
// | `tiesToAway`       | `true`    |
// | `towardZero`       | `false`   |
// | `towardPositive`   | `!sign`   |
// | `towardNegative`   | `sign`    |
// | `awayFromZero`     | `true`    |
//
// ## Encoding
//
// Branchless via a packed lookup. Each rounding mode is assigned a 4-bit
// nybble; within a nybble:
//
//	bit 0 = answer when sign = false
//	bit 1 = answer when sign = true
//	bits 2, 3 = unused (zero)
//
// Per-mode nybble values:
//
// | Mode (code)          | sign=false | sign=true | nybble |
// |----------------------|------------|-----------|--------|
// | `tiesToEven`     (0) | 1          | 1         | `0x3`  |
// | `tiesToAway`     (1) | 1          | 1         | `0x3`  |
// | `towardZero`     (2) | 0          | 0         | `0x0`  |
// | `towardPositive` (3) | 1          | 0         | `0x1`  |
// | `towardNegative` (4) | 0          | 1         | `0x2`  |
// | `awayFromZero`   (5) | 1          | 1         | `0x3`  |
//
// Nybbles are packed with code 0 in the lowest nybble:
//
//	code:     5     4     3     2     1     0
//	nybble:  0x3   0x2   0x1   0x0   0x3   0x3
//	table = 0x321033
//
// Lookup is `(table >> (code * 4 + signBit)) & 1`.
func roundingOverflowsToInfinity(rounding round, sign bool) bool {
	verify("", func() bool { return rounding != const_ROUND_05_UP })
	table := 0x321033
	shift := (int(rounding) << 2) + signBitOf(sign)
	return ((table >> uint(shift)) & 1) != 0
}

// Whether underflow under this rounding direction collapses to zero
// (vs. the smallest representable subnormal magnitude).
//
// | Rounding direction | Result    |
// |--------------------|-----------|
// | `tiesToEven`       | `true`    |
// | `tiesToAway`       | `true`    |
// | `towardZero`       | `true`    |
// | `towardPositive`   | `sign`    |
// | `towardNegative`   | `!sign`   |
// | `awayFromZero`     | `false`   |
//
// ## Encoding
//
// Branchless via a packed lookup. Each rounding mode is assigned a 4-bit
// nybble; within a nybble:
//
//	bit 0 = answer when sign = false
//	bit 1 = answer when sign = true
//	bits 2, 3 = unused (zero)
//
// Per-mode nybble values:
//
// | Mode (code)          | sign=false | sign=true | nybble |
// |----------------------|------------|-----------|--------|
// | `tiesToEven`     (0) | 1          | 1         | `0x3`  |
// | `tiesToAway`     (1) | 1          | 1         | `0x3`  |
// | `towardZero`     (2) | 1          | 1         | `0x3`  |
// | `towardPositive` (3) | 0          | 1         | `0x2`  |
// | `towardNegative` (4) | 1          | 0         | `0x1`  |
// | `awayFromZero`   (5) | 0          | 0         | `0x0`  |
//
// Nybbles are packed with code 0 in the lowest nybble:
//
//	code:     5     4     3     2     1     0
//	nybble:  0x0   0x1   0x2   0x3   0x3   0x3
//	table = 0x012333
//
// Lookup is `(table >> (code * 4 + signBit)) & 1`.
func roundingUnderflowsToZero(rounding round, sign bool) bool {
	verify("", func() bool { return rounding != const_ROUND_05_UP })
	table := 0x12333
	shift := (int(rounding) << 2) + signBitOf(sign)
	return ((table >> uint(shift)) & 1) != 0
}
