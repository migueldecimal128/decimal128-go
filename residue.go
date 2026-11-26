// A residue classifies the remainder discarded by a truncating operation, relative to
// half an ulp of the kept result (D17):
//   RESIDUE_EXACT   (0) -- remainder == 0
//   RESIDUE_LT_HALF (1) -- 0 < 2*remainder < divisor
//   RESIDUE_HALF    (2) -- 2*remainder == divisor
//   RESIDUE_GT_HALF (3) -- 2*remainder > divisor
// Encoding: bit 1 is the round bit (>= half), bit 0 is the sticky bit -- residue =
// (round << 1) | sticky. A residue is a plain 2-bit int defined at the primitive layer
// (no wrapper type). The primitive layer holds the constants and the classifiers over
// 64- and 128-bit remainders; the Core helpers merge residues, map them through
// subtraction, and classify 256-bit remainders and decades. Direction semantics -- what
// a residue means under a rounding mode -- live in Round.
//
// In Go: the constants are const_RESIDUE_*; the value is carried in an int, a bit
// bucket like the dwords (there is no zero-cost wrapper).

package decimal128

type residue = int

const (
	const_RESIDUE_EXACT   residue = 0
	const_RESIDUE_LT_HALF residue = 1
	const_RESIDUE_HALF    residue = 2
	const_RESIDUE_GT_HALF residue = 3
)

// Debug spelling of a residue value.
func residueName(r residue) string {
	if r == 0 {
		return "EXACT"
	} else if r == 1 {
		return "LT_HALF"
	} else if r == 2 {
		return "HALF"
	} else if r == 3 {
		return "GT_HALF"
	} else {
		return "<<< INVALID RESIDUE >>>"
	}
}

// const_DIGIT_MAP packs the residue of a single discarded decimal digit two bits
// per digit: 0 → EXACT, 1–4 → LT_HALF, 5 → HALF, 6–9 → GT_HALF.
const const_DIGIT_MAP = 0b11_11_11_11_10_01_01_01_01_00

// Residue of a single discarded decimal digit: 0 -> EXACT, 1-4 -> LT_HALF, 5
// -> HALF, 6-9 -> GT_HALF.
func residueFromDecimalDigit(digit int) residue {
	verify("", func() bool { return digit < 10 })
	return (const_DIGIT_MAP >> (digit << 1)) & 3
}

// Residue of the remainder of a truncating division. The divPow10 kernels
// classify in place and return Quot*Residue directly (the D17 shape). The
// _pow10 forms look up the 10^pow10 divisor; the _divisor forms take an
// arbitrary divisor (also used by Core's general-division residue).
func residueFromRemainderPow10_64(remainder int64, pow10 int) residue {
	return residueFromRemainderDivisor_64(remainder, pow10_128_dw0(pow10))
}

func residueFromRemainderDivisor_64(remainder, divisor int64) residue {
	if ushr_64_64(remainder, 63) == 0 {
		if remainder == 0 {
			return const_RESIDUE_EXACT
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
	return const_RESIDUE_GT_HALF
}

// The _pow10 forms look up the 10^pow10 divisor in the pow10 tables.

// Residue of a 128-bit remainder against 10^pow10.
func residueFromRemainderPow10_128(remDw0, remDw1 int64, pow10 int) residue {
	verify("", func() bool { return pow10 >= 0 && pow10 <= 38 })
	return residueFromRemainderDivisor_128(remDw0, remDw1, pow10_128_dw0(pow10), pow10_128_dw1(pow10))
}

func residueFromRemainderDivisor_128(remDw0, remDw1, divDw0, divDw1 int64) residue {
	if (remDw1 >> 63) == 0 {
		if isZero_bool_128(remDw0, remDw1) {
			return const_RESIDUE_EXACT
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
	return const_RESIDUE_GT_HALF
}
