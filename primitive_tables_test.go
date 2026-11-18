package decimal128

// The shared power tables (PrimitiveLayer.md §4.5, D16/D53), checked through
// the dword read surface every port exposes (the storage layout is per port).

import (
	"math/big"
	"testing"
)

func TestPrimitiveTablesPow10_u128_entriesMatch(t *testing.T) {
	// Region A: 10^0..10^38.
	for p := 0; p <= 38; p++ {
		if big128(pow10_128_dw0(p), pow10_128_dw1(p)).Cmp(ten(p)) != 0 {
			t.Errorf("10^%d", p)
		}
	}
}

func TestPrimitiveTablesPow10_256_entriesMatch(t *testing.T) {
	// Region B: 10^39..10^77, indexed by i = p - 39.
	for p := 39; p <= 77; p++ {
		i := p - 39
		if big256(pow10_256_dw0(i), pow10_256_dw1(i), pow10_256_dw2(i), pow10_256_dw3(i)).Cmp(ten(p)) != 0 {
			t.Errorf("10^%d", p)
		}
	}
}

func TestPrimitiveTablesPow10_320_entriesMatch(t *testing.T) {
	// Region C (D48): 10^78..10^96, indexed by i = p - 78.
	for p := 78; p <= 96; p++ {
		i := p - 78
		v := big256(pow10_320_dw0(i), pow10_320_dw1(i), pow10_320_dw2(i), pow10_320_dw3(i))
		v.Add(v, new(big.Int).Lsh(ubig(pow10_320_dw4(i)), 256))
		if v.Cmp(ten(p)) != 0 {
			t.Errorf("10^%d", p)
		}
	}
}

func TestPrimitiveTablesPow5_64_entriesMatch(t *testing.T) {
	// 5^0..5^13.
	for p := 0; p <= 13; p++ {
		if ubig(pow5_64(p)).Cmp(new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(p)), nil)) != 0 {
			t.Errorf("5^%d", p)
		}
	}
}
