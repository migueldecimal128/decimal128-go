// round05Up_fromTowardZero: the last-digit fix that turns a toward-zero result
// into the round-05up result.

package decimal128

import "testing"

// The toward-zero result gains one unit only when it is inexact and its last
// digit is 0 or 5, either sign.
var round05UpDigitCases = []struct {
	coeff      int64
	sign       bool
	inexact    bool
	expectedUp bool
}{
	{0, false, true, true},
	{5, true, true, true},
	{10, true, true, true},
	{15, false, true, true},
	{1, false, true, false},
	{4, true, true, false},
	{6, false, true, false},
	{9, true, true, false},
	{0, false, false, false},
	{5, true, false, false},
	{10, true, false, false},
	{15, false, false, false},
	{1, false, false, false},
	{4, true, false, false},
	{6, false, false, false},
	{9, true, false, false},
}

func TestRound05UpLastDigit(t *testing.T) {
	for _, tc := range round05UpDigitCases {
		x := d128Pack(tc.sign, -3, tc.coeff, 0)
		got := round05Up_fromTowardZero(x, tc.inexact)
		expected := tc.coeff
		if tc.expectedUp {
			expected++
		}
		if !d128_bitwiseEQ(got, d128Pack(tc.sign, -3, expected, 0)) {
			t.Errorf("coeff:%d sign:%v inexact:%v", tc.coeff, tc.sign, tc.inexact)
		}
	}
}

// The last-digit test reads the whole coefficient (dw1:dw0), not just its low
// word: 2^64 = 1 (mod 5), so the high word changes the answer.
var round05UpWideCases = []struct {
	dw0, dw1    uint64
	multipleOf5 bool
}{
	{0x0, 0x1, false},                                  // 2^64
	{0x4, 0x1, true},                                   // 2^64 + 4
	{0xFFFF_FFFF_FFFF_FFFF, 0x0, true},                 // 2^64 - 1
	{0x5, 0x3, false},                                  // 3*2^64 + 5
	{0x2, 0x3, true},                                   // 3*2^64 + 2
	{0x35C9_ADC5_DEA0_0000, 0x36, true},                // 10^21
	{0x35C9_ADC5_DEA0_0001, 0x36, false},               // 10^21 + 1
	{0x378D_8E63_FFFF_FFFB, 0x1_ED09_BEAD_87C0, true},  // 10^34 - 5
	{0x378D_8E63_FFFF_FFFF, 0x1_ED09_BEAD_87C0, false}, // 10^34 - 1
}

func TestRound05UpWideCoefficient(t *testing.T) {
	for _, tc := range round05UpWideCases {
		c := fromLoHi_128_64x64(int64(tc.dw0), int64(tc.dw1))
		got := round05Up_fromTowardZero(d128Pack(false, 0, c.dw0, c.dw1), true)
		e := c
		if tc.multipleOf5 {
			e = add_128_128x128(c.dw0, c.dw1, 1, 0)
		}
		if !d128_bitwiseEQ(got, d128Pack(false, 0, e.dw0, e.dw1)) {
			t.Errorf("dw1:%d dw0:%d", tc.dw1, tc.dw0)
		}
	}
}

// Non-finite results pass through, even when inexact.
func TestRound05UpNonFinite(t *testing.T) {
	if !d128_bitwiseEQ(round05Up_fromTowardZero(const_D128_POS_INFINITY, true), const_D128_POS_INFINITY) {
		t.Errorf("inf")
	}
	if !d128_bitwiseEQ(round05Up_fromTowardZero(const_D128_QNAN0, true), const_D128_QNAN0) {
		t.Errorf("nan")
	}
}
