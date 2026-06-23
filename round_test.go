package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

// refNeedsRoundUp is the IEEE rounding decision from first principles, against
// which the packed-table needsRoundUp is checked.
func refNeedsRoundUp(code int, sign, isOdd bool, res int) bool {
	if res == 0 { // EXACT
		return false
	}
	switch code {
	case 0: // tiesToEven
		return res == 3 || (res == 2 && isOdd)
	case 1: // tiesToAway
		return res >= 2
	case 2: // towardZero
		return false
	case 3: // towardPositive: up iff value is positive (away from zero)
		return !sign
	case 4: // towardNegative: up iff value is negative (away from zero)
		return sign
	}
	return false
}

func TestNeedsRoundUp(t *testing.T) {
	codes := []round{TIES_TO_EVEN, TIES_TO_AWAY, TOWARD_ZERO, TOWARD_POSITIVE, TOWARD_NEGATIVE}
	for _, rnd := range codes {
		for _, sign := range []bool{false, true} {
			for _, isOdd := range []bool{false, true} {
				for res := 0; res <= 3; res++ {
					got := needsRoundUp(rnd, sign, isOdd, res)
					want := refNeedsRoundUp(rnd.code, sign, isOdd, res)
					if got != want {
						t.Fatalf("needsRoundUp(code=%d sign=%v odd=%v res=%d) = %v want %v",
							rnd.code, sign, isOdd, res, got, want)
					}
				}
			}
		}
	}
}

func TestRoundingOverflowUnderflow(t *testing.T) {
	codes := []round{TIES_TO_EVEN, TIES_TO_AWAY, TOWARD_ZERO, TOWARD_POSITIVE, TOWARD_NEGATIVE}
	for _, rnd := range codes {
		for _, sign := range []bool{false, true} {
			var wantOvf, wantUnf bool
			switch rnd.code {
			case 0, 1:
				wantOvf, wantUnf = true, true
			case 2:
				wantOvf, wantUnf = false, true
			case 3:
				wantOvf, wantUnf = !sign, sign
			case 4:
				wantOvf, wantUnf = sign, !sign
			}
			if roundingOverflowsToInfinity(rnd, sign) != wantOvf {
				t.Fatalf("overflowsToInfinity(code=%d sign=%v) wrong", rnd.code, sign)
			}
			if roundingUnderflowsToZero(rnd, sign) != wantUnf {
				t.Fatalf("underflowsToZero(code=%d sign=%v) wrong", rnd.code, sign)
			}
		}
	}
}

func TestFlags(t *testing.T) {
	f := newFlags()
	if f.raw() != 0 || f.exceptionLetters() != "" {
		t.Fatalf("fresh flags not empty")
	}
	f.set(INEXACT)
	f.set(INVALID_OPERATION)
	f.set(OVERFLOW)
	if !f.isSet(INEXACT) || !f.isSet(OVERFLOW) || !f.isSet(INVALID_OPERATION) {
		t.Fatalf("set/isSet wrong")
	}
	if f.exceptionLetters() != "xoi" {
		t.Fatalf("letters = %q want xoi", f.exceptionLetters())
	}
	f.clear(OVERFLOW)
	if f.isSet(OVERFLOW) || f.exceptionLetters() != "xi" {
		t.Fatalf("clear wrong: %q", f.exceptionLetters())
	}
	f.clearAll()
	if f.raw() != 0 {
		t.Fatalf("clearAll wrong")
	}
}

func TestResidueMergeInverse(t *testing.T) {
	for left := 0; left <= 3; left++ {
		for right := 0; right <= 3; right++ {
			got := residueMerge(left, right)
			wantRound := left & 2
			wantSticky := (left & 1)
			if right != 0 {
				wantSticky = 1
			}
			want := wantRound | wantSticky
			if got != want {
				t.Fatalf("residueMerge(%d,%d) = %d want %d", left, right, got, want)
			}
		}
	}
	wantInv := []residue{0, 3, 2, 1}
	for res := 0; res <= 3; res++ {
		if got := residueSubtractionInverse(res); got != wantInv[res] {
			t.Fatalf("residueSubtractionInverse(%d) = %d want %d", res, got, wantInv[res])
		}
	}
}

// refResidueVsHalf classifies value relative to half of 10^digitLen.
func refResidueVsHalf(value *big.Int, digitLen int) residue {
	half := new(big.Int).Rsh(pow10big(digitLen), 1) // floor(10^d / 2)
	switch value.Cmp(half) {
	case -1:
		return RESIDUE_LT_HALF
	case 0:
		return RESIDUE_HALF
	default:
		return RESIDUE_GT_HALF
	}
}

func TestResidueDecade(t *testing.T) {
	r := rand.New(rand.NewSource(82))
	for i := 0; i < oracleIters; i++ {
		d := r.Intn(39) // 0..38
		v := new(big.Int).Mod(bigU128(randU128(r)), pow10big(d+1))
		vu := u128FromBig(v)
		if got, want := residueFromDecadeWithDigitLen_128(vu.dw0, vu.dw1, d), refResidueVsHalf(v, d); got != want {
			t.Fatalf("decade_128(v=%s, d=%d) = %s want %s", v, d, residueName(got), residueName(want))
		}

		// 256-bit form. In-contract: high half zero => digitLen 0..39;
		// high half nonzero => digitLen 39..77.
		d2 := r.Intn(78) // 0..77
		v2 := new(big.Int).Mod(bigU256(randU256(r)), pow10big(min(d2+1, 77)))
		v2u := u256FromBig(v2)
		highZero := isZero_bool_128(v2u.dw2, v2u.dw3)
		inContract := false
		if highZero {
			inContract = d2 <= 39
		} else {
			inContract = d2 >= 39 && d2 <= 77
		}
		if !inContract {
			continue
		}
		if got, want := residueFromDecadeWithDigitLen_256(v2u, d2), refResidueVsHalf(v2, d2); got != want {
			t.Fatalf("decade_256(v=%s, d=%d) = %s want %s", v2, d2, residueName(got), residueName(want))
		}
	}
}

func TestResidueRemainderDivisor256(t *testing.T) {
	r := rand.New(rand.NewSource(83))
	for i := 0; i < oracleIters; i++ {
		// divisor < 2^255 so 2*remainder stays in range; remainder < divisor.
		div := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), 1))
		bdiv := bigU256(div)
		if bdiv.Sign() == 0 {
			continue
		}
		brem := new(big.Int).Mod(bigU256(randU256(r)), bdiv)
		rem := u256FromBig(brem)
		if got, want := residueFromRemainderDivisor_256(rem, div), refResidue(brem, bdiv); got != want {
			t.Fatalf("remDiv256 = %s want %s", residueName(got), residueName(want))
		}
	}
}
