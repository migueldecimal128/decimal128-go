// Public rounding-direction enum and the modifier-as-receiver arithmetic that
// rounds in a chosen direction (no flag tracking — the no-flag counterpart of
// Context arithmetic).

package decimal128

import "strconv"

// Rounding is a rounding direction for inexact results: the five IEEE 754-2019
// directions plus two from the General Decimal Arithmetic Specification (GDAS),
// round-up (AwayFromZero) and round-05up (Round05Up).
//
// The wrapper face of the core's `Round` codes (CrossPlatformArchitecture.md
// Section 6.4); the raw values mirror the core ordering exactly.
type Rounding int

const (
	// TiesToEven: round to nearest, ties to even.
	TiesToEven Rounding = iota
	// TiesToAway: round to nearest, ties away from zero.
	TiesToAway
	// TowardZero: round toward zero (truncate).
	TowardZero
	// TowardPositive: round toward positive infinity (ceiling).
	TowardPositive
	// TowardNegative: round toward negative infinity (floor).
	TowardNegative
	// AwayFromZero: round away from zero whenever any discarded digit is nonzero
	// (GDAS round-up).
	AwayFromZero
	// Round05Up: round away from zero only when the kept last digit is 0 or 5,
	// otherwise truncate (GDAS round-05up). Rounding at a longer precision this
	// way and then rounding again to a shorter one gives the same result as
	// rounding once, without double rounding.
	Round05Up
)

// core bridges to the package-internal round value (identical ordering).
func (r Rounding) core() round { return round(r) }

// String returns a short name for the rounding direction.
func (r Rounding) String() string {
	switch r {
	case TiesToEven:
		return "TiesToEven"
	case TiesToAway:
		return "TiesToAway"
	case TowardZero:
		return "TowardZero"
	case TowardPositive:
		return "TowardPositive"
	case TowardNegative:
		return "TowardNegative"
	case AwayFromZero:
		return "AwayFromZero"
	case Round05Up:
		return "Round05Up"
	default:
		return "Rounding(" + strconv.Itoa(int(r)) + ")"
	}
}

// --- arithmetic in this direction (modifier-as-receiver) ---

// Add returns `a + b`, rounded in this direction.
func (r Rounding) Add(a, b Decimal128) Decimal128 {
	return Decimal128{d128_add_rnd(a.core, b.core, r.core())}
}

// Sub returns `a - b`, rounded in this direction.
func (r Rounding) Sub(a, b Decimal128) Decimal128 {
	return Decimal128{d128_subtract_rnd(a.core, b.core, r.core())}
}

// Mul returns `a * b`, rounded in this direction.
func (r Rounding) Mul(a, b Decimal128) Decimal128 {
	return Decimal128{d128_multiply_rnd(a.core, b.core, r.core())}
}

// Quo returns `a / b`, rounded in this direction.
func (r Rounding) Quo(a, b Decimal128) Decimal128 {
	return Decimal128{d128_divide_rnd(a.core, b.core, r.core())}
}
