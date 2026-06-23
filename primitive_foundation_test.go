package decimal128

import (
	"math/big"
	"math/rand"
	"testing"
)

// math/big oracle tests for the Phase 1 primitive foundation. Each limb op is
// checked against an arbitrary-precision reference over many random inputs.

var (
	bigMask64  = new(big.Int).SetUint64(^uint64(0))
	bigMask128 = newMask(128)
	bigMask256 = newMask(256)
)

func newMask(bits uint) *big.Int {
	m := new(big.Int).Lsh(big.NewInt(1), bits)
	return m.Sub(m, big.NewInt(1))
}

func bigU128(x u128) *big.Int {
	b := new(big.Int).SetUint64(uint64(x.dw1))
	b.Lsh(b, 64)
	return b.Or(b, new(big.Int).SetUint64(uint64(x.dw0)))
}

func bigU256(x u256) *big.Int {
	b := new(big.Int)
	for _, dw := range []int64{x.dw3, x.dw2, x.dw1, x.dw0} {
		b.Lsh(b, 64)
		b.Or(b, new(big.Int).SetUint64(uint64(dw)))
	}
	return b
}

func u128FromBig(b *big.Int) u128 {
	v := new(big.Int).And(b, bigMask128)
	lo := new(big.Int).And(v, bigMask64)
	hi := new(big.Int).Rsh(v, 64)
	return u128FromU64(lo.Uint64(), hi.Uint64())
}

func u256FromBig(b *big.Int) u256 {
	v := new(big.Int).And(b, bigMask256)
	var limbs [4]int64
	for i := 0; i < 4; i++ {
		limbs[i] = int64(new(big.Int).And(v, bigMask64).Uint64())
		v.Rsh(v, 64)
	}
	return u256{limbs[0], limbs[1], limbs[2], limbs[3]}
}

func randU128(r *rand.Rand) u128 { return u128{int64(r.Uint64()), int64(r.Uint64())} }
func randU256(r *rand.Rand) u256 {
	return u256{int64(r.Uint64()), int64(r.Uint64()), int64(r.Uint64()), int64(r.Uint64())}
}

const oracleIters = 4000

func TestTier0(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < oracleIters; i++ {
		a := int64(r.Uint64())
		b := int64(r.Uint64())
		if b == 0 {
			b = 1
		}
		ba := new(big.Int).SetUint64(uint64(a))
		bb := new(big.Int).SetUint64(uint64(b))

		wantHi := new(big.Int).Rsh(new(big.Int).Mul(ba, bb), 64)
		if got := uint64(umulHi_64_64x64(a, b)); got != wantHi.Uint64() {
			t.Fatalf("umulHi(%d,%d)=%d want %d", a, b, got, wantHi.Uint64())
		}
		if got := uint64(udiv_64_64x64(a, b)); got != uint64(a)/uint64(b) {
			t.Fatalf("udiv mismatch")
		}
		if got := uint64(urem_64_64x64(a, b)); got != uint64(a)%uint64(b) {
			t.Fatalf("urem mismatch")
		}
		wantCmp := ba.Cmp(bb)
		if got := ucmp_int_64x64(a, b); got != wantCmp {
			t.Fatalf("ucmp(%d,%d)=%d want %d", a, b, got, wantCmp)
		}
		n := int(r.Intn(64))
		if got := uint64(ushr_64_64(a, n)); got != uint64(a)>>uint(n) {
			t.Fatalf("ushr mismatch")
		}
	}
	if clz_int_64(0) != 64 {
		t.Fatalf("clz(0)=%d want 64", clz_int_64(0))
	}
	if clz_int_64(1) != 63 || clz_int_64(-1) != 0 {
		t.Fatalf("clz edge cases wrong")
	}
	if uint64(hi32_64(-1)) != 0xFFFF_FFFF || uint64(lo32_64(-1)) != 0xFFFF_FFFF {
		t.Fatalf("hi32/lo32 wrong")
	}
}

func TestU128Arith(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		y := randU128(r)
		bx, by := bigU128(x), bigU128(y)

		check128(t, "add", add_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1), new(big.Int).Add(bx, by))
		check128(t, "sub", sub_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1), new(big.Int).Sub(bx, by))
		check128(t, "mul", mul_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1), new(big.Int).Mul(bx, by))
		check128(t, "and", and_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1), new(big.Int).And(bx, by))
		check128(t, "or", or_128_128x128(x.dw0, x.dw1, y.dw0, y.dw1), new(big.Int).Or(bx, by))
		check128(t, "not", not_128_128(x.dw0, x.dw1), new(big.Int).Not(bx))

		if got, want := cmp_int_128x128(x.dw0, x.dw1, y.dw0, y.dw1), bx.Cmp(by); got != want {
			t.Fatalf("cmp128=%d want %d", got, want)
		}
		if got := isZero_bool_128(x.dw0, x.dw1); got != (bx.Sign() == 0) {
			t.Fatalf("isZero128 wrong")
		}

		n := int(r.Intn(128))
		check128(t, "shl", shl_128_128(x.dw0, x.dw1, n), new(big.Int).Lsh(bx, uint(n)))
		check128(t, "shr", shr_128_128(x.dw0, x.dw1, n), new(big.Int).Rsh(bx, uint(n)))

		a, b := int64(r.Uint64()), int64(r.Uint64())
		full := new(big.Int).Mul(new(big.Int).SetUint64(uint64(a)), new(big.Int).SetUint64(uint64(b)))
		check128(t, "mul64", mul_128_64x64(a, b), full)
	}
}

func check128(t *testing.T, op string, got u128, want *big.Int) {
	t.Helper()
	// big.Int.And uses two's-complement semantics, so this yields the low 128
	// bits even when want is negative (from Sub/Not).
	w := new(big.Int).And(want, bigMask128)
	if bigU128(got).Cmp(w) != 0 {
		t.Fatalf("%s: got %s want %s", op, bigU128(got).String(), w.String())
	}
}

func TestU256AddSub(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for i := 0; i < oracleIters; i++ {
		// Bound operands so the trapping add/sub families stay in range.
		x := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), 8)) // < 2^248
		y := u256FromBig(new(big.Int).Rsh(bigU256(randU256(r)), 8))
		bx, by := bigU256(x), bigU256(y)

		check256(t, "add256x256", add_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3), new(big.Int).Add(bx, by))
		check256(t, "add256x128", add_256_256x128(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1), new(big.Int).Add(bx, bigU128(u128{y.dw0, y.dw1})))
		check256(t, "add256x64", add_256_256x64(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0), new(big.Int).Add(bx, new(big.Int).SetUint64(uint64(y.dw0))))

		lo := u128{x.dw0, x.dw1}
		lo2 := u128{y.dw0, y.dw1}
		check256(t, "add128x128", add_256_128x128(lo.dw0, lo.dw1, lo2.dw0, lo2.dw1), new(big.Int).Add(bigU128(lo), bigU128(lo2)))

		// Order so x >= y for the trapping subtract.
		if bx.Cmp(by) < 0 {
			x, y, bx, by = y, x, by, bx
		}
		check256(t, "sub256x256", sub_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3), new(big.Int).Sub(bx, by))
	}
}

func TestSubAbs(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for i := 0; i < oracleIters; i++ {
		x := randU128(r)
		y := randU128(r)
		bx, by := bigU128(x), bigU128(y)
		d := subAbs_d128swap_128x128(x.dw0, x.dw1, y.dw0, y.dw1)
		wantAbs := new(big.Int).Abs(new(big.Int).Sub(bx, by))
		if bigU128(u128{d.diffDw0, d.diffDw1}).Cmp(wantAbs) != 0 {
			t.Fatalf("subAbs128 magnitude wrong")
		}
		if d.swap != (by.Cmp(bx) > 0) {
			t.Fatalf("subAbs128 swap flag wrong")
		}

		X := randU256(r)
		Y := randU256(r)
		bX, bY := bigU256(X), bigU256(Y)
		D := subAbs_d256swap_256x256(X.dw0, X.dw1, X.dw2, X.dw3, Y.dw0, Y.dw1, Y.dw2, Y.dw3)
		wantAbs256 := new(big.Int).Abs(new(big.Int).Sub(bX, bY))
		if bigU256(u256{D.diffDw0, D.diffDw1, D.diffDw2, D.diffDw3}).Cmp(wantAbs256) != 0 {
			t.Fatalf("subAbs256 magnitude wrong")
		}
		if D.swap != (bY.Cmp(bX) > 0) {
			t.Fatalf("subAbs256 swap flag wrong")
		}
	}
}

func TestU256Shifts(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	for i := 0; i < oracleIters; i++ {
		x := randU256(r)
		bx := bigU256(x)
		check256(t, "shl1", shl1_256_256(x.dw0, x.dw1, x.dw2, x.dw3), new(big.Int).Lsh(bx, 1))
		check256(t, "shr1", shr1_256_256(x.dw0, x.dw1, x.dw2, x.dw3), new(big.Int).Rsh(bx, 1))
		n := int(r.Intn(256))
		check256(t, "shr_n", shr_256_256(x.dw0, x.dw1, x.dw2, x.dw3, n), new(big.Int).Rsh(bx, uint(n)))
	}
}

func check256(t *testing.T, op string, got u256, want *big.Int) {
	t.Helper()
	w := bigU256(u256FromBig(want))
	if bigU256(got).Cmp(w) != 0 {
		t.Fatalf("%s: got %s want %s", op, bigU256(got).String(), w.String())
	}
}

func TestResidue(t *testing.T) {
	wantDigit := []residue{
		RESIDUE_EXACT,
		RESIDUE_LT_HALF, RESIDUE_LT_HALF, RESIDUE_LT_HALF, RESIDUE_LT_HALF,
		RESIDUE_HALF,
		RESIDUE_GT_HALF, RESIDUE_GT_HALF, RESIDUE_GT_HALF, RESIDUE_GT_HALF,
	}
	for d := 0; d < 10; d++ {
		if got := residueFromDecimalDigit(d); got != wantDigit[d] {
			t.Fatalf("residueFromDecimalDigit(%d)=%s want %s", d, residueName(got), residueName(wantDigit[d]))
		}
	}

	r := rand.New(rand.NewSource(6))
	for i := 0; i < oracleIters; i++ {
		// 64-bit divisor form: divisor in [1, 2^63), remainder in [0, divisor).
		div := int64(r.Uint64()>>1) | 1
		rem := int64(r.Uint64() % uint64(div))
		if got, want := residueFromRemainderDivisor_64(rem, div), refResidue(big.NewInt(rem), big.NewInt(div)); got != want {
			t.Fatalf("residue64(rem=%d,div=%d)=%s want %s", rem, div, residueName(got), residueName(want))
		}

		// 128-bit divisor form: divisor < 2^127 so the remainder's top bit is 0.
		dv := u128{int64(r.Uint64()), int64(r.Uint64()>>1) | 1}
		bdv := bigU128(dv)
		brem := new(big.Int).Mod(bigU128(randU128(r)), bdv)
		rm := u128FromBig(brem)
		if got, want := residueFromRemainderDivisor_128(rm.dw0, rm.dw1, dv.dw0, dv.dw1), refResidue(brem, bdv); got != want {
			t.Fatalf("residue128 mismatch: got %s want %s", residueName(got), residueName(want))
		}
	}
}

// refResidue is the reference classification of remainder vs divisor.
func refResidue(rem, div *big.Int) residue {
	if rem.Sign() == 0 {
		return RESIDUE_EXACT
	}
	twice := new(big.Int).Lsh(rem, 1)
	switch twice.Cmp(div) {
	case -1:
		return RESIDUE_LT_HALF
	case 0:
		return RESIDUE_HALF
	default:
		return RESIDUE_GT_HALF
	}
}
