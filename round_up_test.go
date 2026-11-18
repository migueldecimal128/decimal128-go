package decimal128

import "testing"

var roundUpCases = []struct {
	rounding        round
	residue         residue
	lsdw            int
	sign            bool
	expectedRoundUp bool
}{
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_HALF, 0, false, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_HALF, 2, true, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_HALF, 5, false, true},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_HALF, 7, true, true},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_LT_HALF, 4, false, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_LT_HALF, 6, true, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_LT_HALF, 9, false, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_LT_HALF, 1, true, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_GT_HALF, 0, false, true},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_GT_HALF, 2, true, true},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_GT_HALF, 9, false, true},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_GT_HALF, 1, true, true},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_EXACT, 4, false, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_EXACT, 6, true, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_EXACT, 1, false, false},
	{const_ROUND_TIES_TO_EVEN, const_RESIDUE_EXACT, 3, true, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_HALF, 8, false, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_HALF, 0, true, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_HALF, 7, false, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_HALF, 9, true, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_LT_HALF, 2, false, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_LT_HALF, 4, true, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_LT_HALF, 1, false, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_LT_HALF, 3, true, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_GT_HALF, 6, false, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_GT_HALF, 8, true, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_GT_HALF, 1, false, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_GT_HALF, 3, true, true},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_EXACT, 2, false, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_EXACT, 4, true, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_EXACT, 3, false, false},
	{const_ROUND_TIES_TO_AWAY, const_RESIDUE_EXACT, 5, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_HALF, 6, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_HALF, 8, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_HALF, 9, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_HALF, 1, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_LT_HALF, 0, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_LT_HALF, 2, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_LT_HALF, 3, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_LT_HALF, 5, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_GT_HALF, 4, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_GT_HALF, 6, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_GT_HALF, 3, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_GT_HALF, 5, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_EXACT, 8, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_EXACT, 0, true, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_EXACT, 5, false, false},
	{const_ROUND_TOWARD_ZERO, const_RESIDUE_EXACT, 7, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_HALF, 2, false, true},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_HALF, 4, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_HALF, 1, false, true},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_HALF, 3, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_LT_HALF, 6, false, true},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_LT_HALF, 8, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_LT_HALF, 5, false, true},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_LT_HALF, 7, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_GT_HALF, 0, false, true},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_GT_HALF, 0, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_GT_HALF, 5, false, true},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_GT_HALF, 7, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_EXACT, 0, false, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_EXACT, 0, true, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_EXACT, 7, false, false},
	{const_ROUND_TOWARD_POSITIVE, const_RESIDUE_EXACT, 9, true, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_HALF, 0, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_HALF, 0, true, true},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_HALF, 3, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_HALF, 5, true, true},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_LT_HALF, 0, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_LT_HALF, 0, true, true},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_LT_HALF, 7, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_LT_HALF, 9, true, true},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_GT_HALF, 0, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_GT_HALF, 0, true, true},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_GT_HALF, 7, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_GT_HALF, 9, true, true},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_EXACT, 0, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_EXACT, 0, true, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_EXACT, 9, false, false},
	{const_ROUND_TOWARD_NEGATIVE, const_RESIDUE_EXACT, 1, true, false},
}

func TestRoundUpUlpBiasCase(t *testing.T) {
	for _, tc := range roundUpCases {
		isOdd := tc.lsdw&1 != 0
		if needsRoundUp(tc.rounding, tc.sign, isOdd, tc.residue) != tc.expectedRoundUp {
			t.Errorf("%d residue %d sign:%v lsb:%d", tc.rounding, tc.residue, tc.sign, tc.lsdw)
		}
	}
}
