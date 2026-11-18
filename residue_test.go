package decimal128

// Residue conformance (D17): the (round << 1) | sticky encoding, and the
// single-discarded-digit map (0 -> EXACT, 1-4 -> LT_HALF, 5 -> HALF,
// 6-9 -> GT_HALF).

import "testing"

func TestResidueEncodingValuesAreFrozen(t *testing.T) {
	if const_RESIDUE_EXACT != 0 || const_RESIDUE_LT_HALF != 1 || const_RESIDUE_HALF != 2 || const_RESIDUE_GT_HALF != 3 {
		t.Error("residue encoding changed")
	}
}

func TestResidueFromDecimalDigit_mapsEachDigit(t *testing.T) {
	want := map[int]residue{0: const_RESIDUE_EXACT, 1: const_RESIDUE_LT_HALF, 4: const_RESIDUE_LT_HALF,
		5: const_RESIDUE_HALF, 6: const_RESIDUE_GT_HALF, 9: const_RESIDUE_GT_HALF}
	for d, r := range want {
		if residueFromDecimalDigit(d) != r {
			t.Errorf("digit %d", d)
		}
	}
}
