package decimal128

import (
	"math/big"
	"testing"
)

func TestCalcBitLenDigitLenTestCases(t *testing.T) {
	for _, tc := range []struct {
		a        string
		expected int
	}{
		{"1", 1},
		{"99999999999999999999999999999999999999", 38},
		{"100000000000000000000000000000000000000", 39},
		{"100000000000000000000000000000000000001", 39},
	} {
		a := u128Of(bigStr(tc.a))
		if got := digitLen_int_128(a.dw0, a.dw1); got != tc.expected {
			t.Errorf("calcDigitLen(%s) = %d, expected %d", tc.a, got, tc.expected)
		}
	}
}

// Boundary sweep ported from decimal128-kotlin TestCalcDigitCount64: every
// power of 10 is tested at 10^k - 1, 10^k, and 10^k + 1, since the
// bitLen-based estimate is only ever off at a power-of-10 boundary.

func TestCalcBitLenDigitLenBoundarySweep128(t *testing.T) {
	if digitLen_int_128(0, 0) != 0 || digitLen_int_128(-1, -1) != 39 {
		t.Error("0 / max")
	}
	for k := 0; k <= 38; k++ {
		p := ten(k)
		for d, want := range map[int64]int{-1: k, 0: k + 1, 1: k + 1} {
			v := u128Of(new(big.Int).Add(p, big.NewInt(d)))
			if digitLen_int_128(v.dw0, v.dw1) != want {
				t.Errorf("10^%d %+d", k, d)
			}
		}
	}
}

func TestCalcBitLenDigitLenBoundarySweep256(t *testing.T) {
	if digitLen_int_256(0, 0, 0, 0) != 0 || digitLen_int_256(-1, -1, -1, -1) != 78 {
		t.Error("0 / max")
	}
	// bitLen 129: (129 * 1233) >> 12 == 38, exercises the max(_, 39) clamp
	if digitLen_int_256(0, 0, 1, 0) != 39 {
		t.Error("bitLen 129 clamp")
	}
	one := u256{dw0: 1}
	p := one // 10^k
	for k := 0; k <= 77; k++ {
		// for k == 0, 10^0 - 1 == 0, which has digitLen 0 == k
		pm1 := sub_256_256x256(p.dw0, p.dw1, p.dw2, p.dw3, one.dw0, one.dw1, one.dw2, one.dw3)
		pp1 := add_256_256x64(p.dw0, p.dw1, p.dw2, p.dw3, 1)
		if digitLen_int_256(pm1.dw0, pm1.dw1, pm1.dw2, pm1.dw3) != k ||
			digitLen_int_256(p.dw0, p.dw1, p.dw2, p.dw3) != k+1 ||
			digitLen_int_256(pp1.dw0, pp1.dw1, pp1.dw2, pp1.dw3) != k+1 {
			t.Errorf("10^%d", k)
		}
		if k < 77 {
			p = mul_256_256x64(p.dw0, p.dw1, p.dw2, p.dw3, 10)
		}
	}
}

func calcDigitLenOf(v *big.Int) int {
	if v.Sign() == 0 {
		return 0
	}
	return len(v.String())
}

func TestCalcBitLenDigitLenRandomized128(t *testing.T) {
	rng := splitMix64{0x0128_D161_7E11}
	for i := 0; i < 10_000; i++ {
		bitLen := rng.rangeIn(1, 128)
		x := new(big.Int).Rsh(big128(int64(rng.next()), int64(rng.next())), uint(128-bitLen))
		v := u128Of(x)
		if got := digitLen_int_128(v.dw0, v.dw1); got != calcDigitLenOf(x) {
			t.Errorf("calcDigitLen(%v) = %d", x, got)
		}
	}
}

func TestCalcBitLenDigitLenRandomized256(t *testing.T) {
	rng := splitMix64{0x0256_D161_7E11}
	for i := 0; i < 2_000; i++ {
		digitLen := rng.rangeIn(1, 77)
		v := big.NewInt(int64(rng.rangeIn(1, 9)))
		for j := 1; j < digitLen; j++ {
			v.Mul(v, big.NewInt(10)).Add(v, big.NewInt(int64(rng.rangeIn(0, 9))))
		}
		x := u256Of(v)
		if got := digitLen_int_256(x.dw0, x.dw1, x.dw2, x.dw3); got != digitLen {
			t.Errorf("calcDigitLen(%v) = %d, expected %d", v, got, digitLen)
		}
	}
}

func TestCalcBitLenDigitLenBitLen_and_digitLen_128(t *testing.T) {
	limbs := []int64{0, 1, -1, s64(0x8000_0000_0000_0000), 0x0123_4567_89AB_CDEF}
	for _, hi := range limbs {
		for _, lo := range limbs {
			v := big128(lo, hi)
			if bitLen_int_128(lo, hi) != v.BitLen() || digitLen_int_128(lo, hi) != calcDigitLenOf(v) {
				t.Errorf("bitLen/digitLen %v", v)
			}
		}
	}
}

func TestCalcBitLenDigitLenBitLen_and_digitLen_256(t *testing.T) {
	m := int64(-1)
	for _, x := range [][4]int64{
		{0, 0, 0, 0}, {1, 0, 0, 0}, {m, m, 0, 0}, {m, m, m, 0}, {m, m, m, m},
		{0x0123_4567_89AB_CDEF, 0xCAFE, 0xBEEF, 0x42}, {0, 0, 1, 0}, {0, 0, 0, 1},
	} {
		v := big256(x[0], x[1], x[2], x[3])
		if bitLen_int_256(x[0], x[1], x[2], x[3]) != v.BitLen() || digitLen_int_256(x[0], x[1], x[2], x[3]) != calcDigitLenOf(v) {
			t.Errorf("bitLen/digitLen %v", v)
		}
	}
}
