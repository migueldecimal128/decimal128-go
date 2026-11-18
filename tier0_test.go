package decimal128

// Tier-0 conformance (PrimitiveLayer.md Section 3.2). The cases deliberately
// exercise the sign boundary, where the unsigned contract diverges from a
// signed int64, since that divergence is the whole reason these ops are named.

import (
	"math"
	"math/big"
	"testing"
)

const tier0MaxU int64 = -1             // 2^64 - 1, "-1" signed
const tier0Bit63 int64 = math.MinInt64 // 2^63, "min" signed

func TestTier0UmulHi_matchesBigInteger(t *testing.T) {
	xs := []int64{0, 1, 2, 10, tier0Bit63, tier0MaxU, 0x0123_4567_89AB_CDEF, s64(0xDEAD_BEEF_CAFE_BABE)}
	for _, a := range xs {
		for _, b := range xs {
			want := new(big.Int).Rsh(new(big.Int).Mul(ubig(a), ubig(b)), 64)
			if ubig(umulHi_64_64x64(a, b)).Cmp(want) != 0 {
				t.Errorf("umulHi(%d, %d)", uint64(a), uint64(b))
			}
		}
	}
}

func TestTier0Udiv_urem_unsignedSemantics(t *testing.T) {
	// MAX_U / 2: signed division of -1 would give 0; unsigned must give 2^63 - 1.
	if udiv_64_64x64(tier0MaxU, 2) != 0x7FFF_FFFF_FFFF_FFFF || urem_64_64x64(tier0MaxU, 2) != 1 {
		t.Error("MAX_U / 2")
	}
	// BIT63 (2^63) / 3.
	if uint64(udiv_64_64x64(tier0Bit63, 3)) != (uint64(1)<<63)/3 || uint64(urem_64_64x64(tier0Bit63, 3)) != (uint64(1)<<63)%3 {
		t.Error("2^63 / 3")
	}
	if udiv_64_64x64(5, 7) != 0 || urem_64_64x64(5, 7) != 5 {
		t.Error("5 / 7")
	}
}

func TestTier0Ucmp_isMinusOneZeroPlusOne_acrossSignBoundary(t *testing.T) {
	// Signed: BIT63 < 1. Unsigned: BIT63 (2^63) > 1.
	if ucmp_int_64x64(tier0Bit63, 1) != 1 || ucmp_int_64x64(1, tier0Bit63) != -1 ||
		ucmp_int_64x64(tier0MaxU, tier0MaxU) != 0 || ucmp_int_64x64(tier0MaxU, 0) != 1 || ucmp_int_64x64(0, 1) != -1 {
		t.Error("ucmp")
	}
}

func TestTier0Clz_definedAtZeroAndBoundaries(t *testing.T) {
	if clz_int_64(0) != 64 || clz_int_64(tier0Bit63) != 0 || clz_int_64(tier0MaxU) != 0 ||
		clz_int_64(1) != 63 || clz_int_64(0xFF) != 56 {
		t.Error("clz")
	}
}

func TestTier0Ushr_isLogicalNotArithmetic(t *testing.T) {
	// Arithmetic >> on MAX_U (-1) stays -1; logical >>> must zero-fill.
	if ushr_64_64(tier0MaxU, 1) != 0x7FFF_FFFF_FFFF_FFFF || ushr_64_64(tier0MaxU, 63) != 1 ||
		ushr_64_64(tier0MaxU, 0) != tier0MaxU || ushr_64_64(tier0Bit63, 63) != 1 {
		t.Error("ushr")
	}
}

func TestTier0Hi32_lo32_splitDword(t *testing.T) {
	v := s64(0xDEAD_BEEF_CAFE_BABE)
	if hi32_64(v) != 0x0000_0000_DEAD_BEEF || lo32_64(v) != 0x0000_0000_CAFE_BABE {
		t.Error("split")
	}
	// Reassembly round-trips.
	if (hi32_64(v)<<32)|lo32_64(v) != v || hi32_64(0xCAFE_BABE&0xFFFF_FFFF) != 0 {
		t.Error("reassembly")
	}
}
