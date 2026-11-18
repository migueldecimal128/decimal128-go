package decimal128

import (
	"math"
	"testing"
)

func TestU128Div10TestCases(t *testing.T) {
	u64max := fromLoHi_128_64x64(-1, 0)
	tcs := []struct {
		name    string
		n       u128
		q       u128
		residue residue
	}{
		// Move a case to the top to debug it easily
		{"zero", u128{}, u128{}, const_RESIDUE_EXACT},
		{"small_exact", u128{dw0: 100}, u128{dw0: 10}, const_RESIDUE_EXACT},
		{"small_ltHalf", u128{dw0: 102}, u128{dw0: 10}, const_RESIDUE_LT_HALF},
		{"small_half", u128{dw0: 105}, u128{dw0: 10}, const_RESIDUE_HALF},
		{"small_gtHalf", u128{dw0: 107}, u128{dw0: 10}, const_RESIDUE_GT_HALF},
		{"uint64_boundary", u64max, u128{dw0: int64(uint64(math.MaxUint64) / 10)}, const_RESIDUE_HALF},
		{"uint128_max", u128{dw0: -1, dw1: -1}, u128Of(new(bigIntT).Div(big128(-1, -1), ten(1))), const_RESIDUE_HALF},
	}
	for _, tc := range tcs {
		qr := div1e1_q128res_128_magic(tc.n.dw0, tc.n.dw1)
		if qr.quotDw0 != tc.q.dw0 || qr.quotDw1 != tc.q.dw1 {
			t.Errorf("quotient mismatch: %s", tc.name)
		}
		if qr.residue != tc.residue {
			t.Errorf("residue mismatch: %s", tc.name)
		}
	}
}
