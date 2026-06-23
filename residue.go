// Residue classification (PrimitiveLayer.md §4; CrossPlatformArchitecture.md
// §6.4 — the Residue D17 type). A residue classifies the remainder discarded by
// a truncating operation, relative to half an ulp of the kept result:
//
//	RESIDUE_EXACT   (0) — remainder == 0
//	RESIDUE_LT_HALF (1) — 0 < 2·remainder < divisor
//	RESIDUE_HALF    (2) — 2·remainder == divisor
//	RESIDUE_GT_HALF (3) — 2·remainder > divisor
//
// Encoding: bit 1 is the round bit (≥ half), bit 0 is the sticky bit —
// residue = (round << 1) | sticky. A residue is a 2-bit value carried in an int,
// a bit bucket like the dwords (no zero-cost wrapper, so no core wraps it).
//
// The _pow10 forms (which look up the 10^pow10 divisor in the pow10 tables) land
// with PrimitiveTables in a later increment; the _divisor and digit forms here
// need no tables.

package decimal128

type residue = int

const (
	RESIDUE_EXACT   residue = 0
	RESIDUE_LT_HALF residue = 1
	RESIDUE_HALF    residue = 2
	RESIDUE_GT_HALF residue = 3
)

// residueName is the debug spelling of a residue value.
func residueName(r residue) string {
	switch r {
	case 0:
		return "EXACT"
	case 1:
		return "LT_HALF"
	case 2:
		return "HALF"
	case 3:
		return "GT_HALF"
	default:
		return "<<< INVALID RESIDUE >>>"
	}
}

// digitResidueMap packs the residue of a single discarded decimal digit two bits
// per digit: 0 → EXACT, 1–4 → LT_HALF, 5 → HALF, 6–9 → GT_HALF.
const digitResidueMap = 0b11_11_11_11_10_01_01_01_01_00

func residueFromDecimalDigit(digit int) residue {
	verify("", digit < 10)
	return (digitResidueMap >> (digit << 1)) & 3
}

func residueFromRemainderDivisor_64(remainder, divisor int64) residue {
	if ushr_64_64(remainder, 63) == 0 {
		if remainder == 0 {
			return RESIDUE_EXACT
		}
		doubled := int64(uint64(remainder) << 1)
		c := ucmp_int_64x64(doubled, divisor)
		ge := 0
		if c >= 0 {
			ge = 1
		}
		eq := 0
		if c == 0 {
			eq = 1
		}
		// round = ge, sticky = !eq → LT:0|1=1, HALF:1|0=2, GT:1|1=3
		return (ge << 1) | (1 ^ eq)
	}
	return RESIDUE_GT_HALF
}

func residueFromRemainderDivisor_128(remDw0, remDw1, divDw0, divDw1 int64) residue {
	if (remDw1 >> 63) == 0 {
		if isZero_bool_128(remDw0, remDw1) {
			return RESIDUE_EXACT
		}
		doubled := shl_128_128(remDw0, remDw1, 1)
		c := cmp_int_128x128(doubled.dw0, doubled.dw1, divDw0, divDw1)
		ge := 0
		if c >= 0 {
			ge = 1
		}
		eq := 0
		if c == 0 {
			eq = 1
		}
		return (ge << 1) | (1 ^ eq)
	}
	return RESIDUE_GT_HALF
}
