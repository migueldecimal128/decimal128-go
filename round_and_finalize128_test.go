package decimal128

import "testing"

var roundAndFinalize128Tcs = []struct {
	tag      string
	sign     bool
	qExpIn   int
	coeffIn  string
	residue  residue
	rounding round
	expected string
}{
	{"zero", false, 0, "0", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "0E+0"},
	{"neg_zero", true, 0, "0", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "-0E+0"},
	{"one_e1", false, 1, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1E+1"},
	{"neg_two_em2", true, -2, "2", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "-2E-2"},
	{"max34_digits", false, 0, "1234567890123456789012345678901234", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901234E+0"},
	{"qExp_at_eMax", false, 6111, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1E+6111"},
	{"qExp_at_qTiny", false, -6176, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1E-6176"},
	{"zero_lthalf_tte", false, 0, "0", const_RESIDUE_LT_HALF, const_ROUND_TIES_TO_EVEN, "0E+0"},
	{"zero_half_tte", false, 0, "0", const_RESIDUE_HALF, const_ROUND_TIES_TO_EVEN, "0E+0"},
	{"zero_gthalf_tte", false, 0, "0", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "1E+0"},
	{"zero_lthalf_tta", false, 0, "0", const_RESIDUE_LT_HALF, const_ROUND_TIES_TO_AWAY, "0E+0"},
	{"zero_half_tta", false, 0, "0", const_RESIDUE_HALF, const_ROUND_TIES_TO_AWAY, "1E+0"},
	{"zero_gthalf_tta", false, 0, "0", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_AWAY, "1E+0"},
	{"zero_lthalf_tz", false, 0, "0", const_RESIDUE_LT_HALF, const_ROUND_TOWARD_ZERO, "0E+0"},
	{"zero_half_tz", false, 0, "0", const_RESIDUE_HALF, const_ROUND_TOWARD_ZERO, "0E+0"},
	{"zero_gthalf_tz", false, 0, "0", const_RESIDUE_GT_HALF, const_ROUND_TOWARD_ZERO, "0E+0"},
	{"zero_lthalf_tp_pos", false, 0, "0", const_RESIDUE_LT_HALF, const_ROUND_TOWARD_POSITIVE, "1E+0"},
	{"zero_half_tp_pos", false, 0, "0", const_RESIDUE_HALF, const_ROUND_TOWARD_POSITIVE, "1E+0"},
	{"zero_gthalf_tp_pos", false, 0, "0", const_RESIDUE_GT_HALF, const_ROUND_TOWARD_POSITIVE, "1E+0"},
	{"zero_lthalf_tp_neg", true, 0, "0", const_RESIDUE_LT_HALF, const_ROUND_TOWARD_POSITIVE, "-0E+0"},
	{"zero_half_tp_neg", true, 0, "0", const_RESIDUE_HALF, const_ROUND_TOWARD_POSITIVE, "-0E+0"},
	{"zero_gthalf_tp_neg", true, 0, "0", const_RESIDUE_GT_HALF, const_ROUND_TOWARD_POSITIVE, "-0E+0"},
	{"zero_lthalf_tn_neg", true, 0, "0", const_RESIDUE_LT_HALF, const_ROUND_TOWARD_NEGATIVE, "-1E+0"},
	{"zero_gthalf_tn_neg", true, 0, "0", const_RESIDUE_GT_HALF, const_ROUND_TOWARD_NEGATIVE, "-1E+0"},
	{"zero_qexp_pos", false, 5, "0", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "0E+5"},
	{"zero_qexp_neg", false, -5, "0", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "0E-5"},
	{"c2_lthalf_tte", false, 0, "2", const_RESIDUE_LT_HALF, const_ROUND_TIES_TO_EVEN, "2E+0"},
	{"c2_half_tte", false, 0, "2", const_RESIDUE_HALF, const_ROUND_TIES_TO_EVEN, "2E+0"},
	{"c2_gthalf_tte", false, 0, "2", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "3E+0"},
	{"c3_lthalf_tte", false, 0, "3", const_RESIDUE_LT_HALF, const_ROUND_TIES_TO_EVEN, "3E+0"},
	{"c3_half_tte", false, 0, "3", const_RESIDUE_HALF, const_ROUND_TIES_TO_EVEN, "4E+0"},
	{"c3_gthalf_tte", false, 0, "3", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "4E+0"},
	{"c2_half_tta", false, 0, "2", const_RESIDUE_HALF, const_ROUND_TIES_TO_AWAY, "3E+0"},
	{"c3_gthalf_tz", false, 0, "3", const_RESIDUE_GT_HALF, const_ROUND_TOWARD_ZERO, "3E+0"},
	{"trunc_half_even", false, 0, "12345678901234567890123456789012345", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901234E+1"},
	{"trunc_half_odd", false, 0, "12345678901234567890123456789012355", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901236E+1"},
	{"trunc_lthalf", false, 0, "12345678901234567890123456789012341", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901234E+1"},
	{"trunc_gthalf", false, 0, "12345678901234567890123456789012347", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901235E+1"},
	{"trunc_merge_inbound", false, 0, "12345678901234567890123456789012340", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901234E+1"},
	{"trunc_merge_half_half", false, 0, "12345678901234567890123456789012345", const_RESIDUE_HALF, const_ROUND_TIES_TO_EVEN, "1234567890123456789012345678901235E+1"},
	{"trunc_half_even_tta", false, 0, "12345678901234567890123456789012345", const_RESIDUE_EXACT, const_ROUND_TIES_TO_AWAY, "1234567890123456789012345678901235E+1"},
	{"rollover_gthalf", false, 0, "9999999999999999999999999999999999", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "1000000000000000000000000000000000E+1"},
	{"rollover_half", false, 0, "9999999999999999999999999999999999", const_RESIDUE_HALF, const_ROUND_TIES_TO_EVEN, "1000000000000000000000000000000000E+1"},
	{"rollover_via_trunc", false, 0, "99999999999999999999999999999999999", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1000000000000000000000000000000000E+2"},
	{"no_rollover_lthalf", false, 0, "9999999999999999999999999999999999", const_RESIDUE_LT_HALF, const_ROUND_TIES_TO_EVEN, "9999999999999999999999999999999999E+0"},
	{"clamp_3digit", false, 6113, "123", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "12300E+6111"},
	{"clamp_1digit", false, 6112, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "10E+6111"},
	{"clamp_33digit", false, 6112, "999999999999999999999999999999999", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "9999999999999999999999999999999990E+6111"},
	{"at_max_34digit", false, 6111, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1000000000000000000000000000000000E+6111"},
	{"overflow_tte", false, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "Infinity"},
	{"overflow_neg_tte", true, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "-Infinity"},
	{"overflow_tz", false, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TOWARD_ZERO, "9999999999999999999999999999999999E+6111"},
	{"overflow_tp_pos", false, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TOWARD_POSITIVE, "Infinity"},
	{"overflow_tp_neg", true, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TOWARD_POSITIVE, "-9999999999999999999999999999999999E+6111"},
	{"overflow_tn_pos", false, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TOWARD_NEGATIVE, "9999999999999999999999999999999999E+6111"},
	{"overflow_tn_neg", true, 6112, "1000000000000000000000000000000000", const_RESIDUE_EXACT, const_ROUND_TOWARD_NEGATIVE, "-Infinity"},
	{"overflow_via_rollover", false, 6111, "9999999999999999999999999999999999", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "Infinity"},
	{"ub_lthalf_tte", false, -6177, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "0E-6176"},
	{"ub_half_tte", false, -6177, "5", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "0E-6176"},
	{"ub_half_tta", false, -6177, "5", const_RESIDUE_EXACT, const_ROUND_TIES_TO_AWAY, "1E-6176"},
	{"ub_gthalf_tte", false, -6177, "6", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1E-6176"},
	{"ub_lthalf_tp_pos", false, -6177, "1", const_RESIDUE_LT_HALF, const_ROUND_TOWARD_POSITIVE, "1E-6176"},
	{"ub_lthalf_tn_pos", false, -6177, "1", const_RESIDUE_LT_HALF, const_ROUND_TOWARD_NEGATIVE, "0E-6176"},
	{"sub_exact", false, -6177, "10", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1E-6176"},
	{"sub_half_odd", false, -6177, "15", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "2E-6176"},
	{"sub_half_even", false, -6177, "25", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "2E-6176"},
	{"sub_lthalf", false, -6177, "11", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "1E-6176"},
	{"sub_gthalf", false, -6177, "19", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "2E-6176"},
	{"sub_merge_residue", false, -6177, "10", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "1E-6176"},
	{"sub_rollover", false, -6177, "99", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "10E-6176"},
	{"swamp_tte", false, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_EVEN, "0E-6176"},
	{"swamp_lthalf_tte", false, -6200, "1", const_RESIDUE_LT_HALF, const_ROUND_TIES_TO_EVEN, "0E-6176"},
	{"swamp_gthalf_tte", false, -6200, "1", const_RESIDUE_GT_HALF, const_ROUND_TIES_TO_EVEN, "0E-6176"},
	{"swamp_tp_pos", false, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TOWARD_POSITIVE, "1E-6176"},
	{"swamp_tn_pos", false, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TOWARD_NEGATIVE, "0E-6176"},
	{"swamp_tp_neg", true, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TOWARD_POSITIVE, "-0E-6176"},
	{"swamp_tn_neg", true, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TOWARD_NEGATIVE, "-1E-6176"},
	{"swamp_tz", false, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TOWARD_ZERO, "0E-6176"},
	{"swamp_tta", false, -6200, "1", const_RESIDUE_EXACT, const_ROUND_TIES_TO_AWAY, "0E-6176"},
}

func TestRoundAndFinalize128TestRoundAndFinalize128(t *testing.T) {
	for _, tc := range roundAndFinalize128Tcs {
		c := u128Of(bigStr(tc.coeffIn))
		if got := d128_format(finalize_128_ctxnull(tc.sign, tc.qExpIn, c, tc.residue, tc.rounding, nil), const_FORMAT_COEFFICIENT_QEXP); got != tc.expected {
			t.Errorf("%s: got %s, expected %s", tc.tag, got, tc.expected)
		}
		if tc.rounding != const_ROUND_TIES_TO_EVEN {
			continue
		}
		if got := d128_format(finalize_128_quiet(tc.sign, tc.qExpIn, c, tc.residue), const_FORMAT_COEFFICIENT_QEXP); got != tc.expected {
			t.Errorf("%s (quiet): got %s, expected %s", tc.tag, got, tc.expected)
		}
	}
}
