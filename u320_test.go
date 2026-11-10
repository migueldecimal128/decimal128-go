package decimal128

// D48 primitives (D48GoWorkOrder.md Phase 1, G-11; Swift TestU320.swift) against
// math/big: u192/u320 arithmetic, the width-aware power-of-ten helpers for every
// k = 0..96 (including each width-class boundary 19/20, 38/39, 57/58, 77/78),
// the recipmul divPow10-320 kernel and its fixed-k entries, the 320÷160 MG
// divide (with the crafted add-back / all-ones-limb cases) and isqrt_320.
//
// Swift's ten tests with the same SplitMix seeds and draw order; every check
// counts as one Swift #expect, so a mutation's failure count compares across
// ports. Rust's two additions (fixed-k entries, mul_320_320x64) draw after
// Swift's, leaving Swift's counts unchanged.

import (
	"fmt"
	"math/big"
	"testing"
)

// splitMix is the deterministic test RNG (SplitMix64), so a failure reproduces.
type splitMix struct{ state uint64 }

func (r *splitMix) next() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// expectCounter is Swift's #expect semantics: record and continue.
type expectCounter struct {
	t        *testing.T
	failures int
	shown    int
}

func (c *expectCounter) expect(ok bool, what func() string) {
	if ok {
		return
	}
	c.failures++
	if c.shown < 10 {
		c.shown++
		c.t.Errorf("%s", what())
	}
}

func (c *expectCounter) done(name string) {
	c.t.Logf("%s: %d failures", name, c.failures)
	if c.failures != 0 {
		c.t.Errorf("%s: %d failures", name, c.failures)
	}
}

// --- big.Int <-> u320 helpers ---

func bigTen(k int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil) }

func bigOne() *big.Int { return big.NewInt(1) }

func bigPow2(n int) *big.Int { return new(big.Int).Lsh(bigOne(), uint(n)) }

func bigAdd(a, b *big.Int) *big.Int { return new(big.Int).Add(a, b) }

func bigSub(a, b *big.Int) *big.Int { return new(big.Int).Sub(a, b) }

func bigMul(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }

func bigQuo(a, b *big.Int) *big.Int { return new(big.Int).Quo(a, b) }

func bigRem(a, b *big.Int) *big.Int { return new(big.Int).Rem(a, b) }

func bigShl(a *big.Int, n int) *big.Int { return new(big.Int).Lsh(a, uint(n)) }

func bigShr(a *big.Int, n int) *big.Int { return new(big.Int).Rsh(a, uint(n)) }

func bigU64(v uint64) *big.Int { return new(big.Int).SetUint64(v) }

func bigOf320(x u320) *big.Int {
	r := new(big.Int)
	for _, dw := range []int64{x.dw4, x.dw3, x.dw2, x.dw1, x.dw0} {
		r.Lsh(r, 64)
		r.Or(r, bigU64(uint64(dw)))
	}
	return r
}

func bigOf192(x u192) *big.Int { return bigOf320(u320FromU192(x)) }

func u320Of(x *big.Int) u320 {
	if (x.Sign() < 0) || (x.BitLen() > 320) {
		panic("u320Of: out of range")
	}
	var b [40]byte
	x.FillBytes(b[:])
	limb := func(i int) int64 { // limb i, little-endian
		var v uint64
		for j := 0; j < 8; j++ {
			v = (v << 8) | uint64(b[40-8*(i+1)+j])
		}
		return int64(v)
	}
	return u320{limb(0), limb(1), limb(2), limb(3), limb(4)}
}

func u192Of(x *big.Int) u192 { return u320Of(x).u192() }

// randomBits is a uniformly random value of exactly `nbits` bits (nbits >= 1),
// or 0 for nbits 0.
func randomBits(rng *splitMix, nbits int) *big.Int {
	if nbits == 0 {
		return new(big.Int)
	}
	r := new(big.Int)
	have := 0
	for have < nbits {
		r.Lsh(r, 64)
		r.Or(r, bigU64(rng.next()))
		have += 64
	}
	r.Rsh(r, uint(have-nbits))
	top := bigPow2(nbits - 1)
	if r.Cmp(top) < 0 {
		r.Add(r, top)
	}
	return r
}

// randomBelow is a random value below max (max > 0).
func randomBelow(rng *splitMix, max *big.Int) *big.Int {
	return bigRem(randomBits(rng, max.BitLen()+64), max)
}

func expectedResidue(r, d *big.Int) residue {
	if r.Sign() == 0 {
		return RESIDUE_EXACT
	}
	switch bigShl(r, 1).Cmp(d) {
	case -1:
		return RESIDUE_LT_HALF
	case 0:
		return RESIDUE_HALF
	default:
		return RESIDUE_GT_HALF
	}
}

func u320Two320() *big.Int { return bigPow2(320) }

// --- arithmetic ---

func TestU320AddSubShiftsCompare(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{1}
	two320 := u320Two320()
	for it := 0; it < 5000; it++ {
		a := randomBits(rng, int(rng.next()%320))
		b := randomBits(rng, int(rng.next()%320))
		hi, lo := a, b
		if a.Cmp(b) < 0 {
			hi, lo = b, a
		}
		if bigAdd(hi, lo).Cmp(two320) < 0 {
			c.expect(bigOf320(add_320_320x320(u320Of(hi), u320Of(lo))).Cmp(bigAdd(hi, lo)) == 0,
				func() string { return fmt.Sprintf("add %v %v", hi, lo) })
		}
		c.expect(bigOf320(sub_320_320x320(u320Of(hi), u320Of(lo))).Cmp(bigSub(hi, lo)) == 0,
			func() string { return fmt.Sprintf("sub %v %v", hi, lo) })
		c.expect(cmp_int_320x320(u320Of(a), u320Of(b)) == a.Cmp(b),
			func() string { return fmt.Sprintf("cmp %v %v", a, b) })
		c.expect(bitLen_int_320(u320Of(a)) == a.BitLen(),
			func() string { return fmt.Sprintf("bitLen %v", a) })
		n := int(rng.next() % 320)
		c.expect(bigOf320(shr_320_320(u320Of(a), n)).Cmp(bigShr(a, n)) == 0,
			func() string { return fmt.Sprintf("shr %v %d", a, n) })
		if a.BitLen()+n <= 320 {
			c.expect(bigOf320(shl_320_320(u320Of(a), n)).Cmp(bigShl(a, n)) == 0,
				func() string { return fmt.Sprintf("shl %v %d", a, n) })
		}
		low := int(rng.next() % 321)
		want := (bigRem(a, bigPow2(low)).Sign() != 0) && (low > 0)
		c.expect(lowBitsNonZero_320(u320Of(a), low) == want,
			func() string { return fmt.Sprintf("lowBitsNonZero %v n=%d", a, low) })
	}
	c.done("addSubShiftsCompare")
}

func TestU320Multiplies(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{2}
	two320 := u320Two320()
	all1 := bigSub(bigPow2(160), bigOne())
	all1w := bigSub(two320, bigOne())
	pairs := [][2]*big.Int{{all1, all1}, {new(big.Int), all1}, {bigOne(), all1}}
	for i := 0; i < 4000; i++ {
		a := randomBits(rng, int(rng.next()%161))
		b := randomBits(rng, int(rng.next()%161))
		pairs = append(pairs, [2]*big.Int{a, b})
	}
	for _, p := range pairs {
		a, b := p[0], p[1]
		c.expect(bigOf320(mul_320_192x192(u192Of(a), u192Of(b))).Cmp(bigMul(a, b)) == 0,
			func() string { return fmt.Sprintf("mul192 %v %v", a, b) })
	}
	wide := [][2]*big.Int{{all1w, all1w}, {all1w, bigOne()}, {new(big.Int), all1w}}
	for i := 0; i < 4000; i++ {
		a := randomBits(rng, int(rng.next()%321))
		b := randomBits(rng, int(rng.next()%321))
		wide = append(wide, [2]*big.Int{a, b})
	}
	for _, p := range wide {
		a, b := p[0], p[1]
		lo, hi := mulWide_640_320x320(u320Of(a), u320Of(b))
		c.expect(bigAdd(bigShl(bigOf320(hi), 320), bigOf320(lo)).Cmp(bigMul(a, b)) == 0,
			func() string { return fmt.Sprintf("mulWide %v %v", a, b) })
	}
	// mul_320_320x64 (exp's n * ln10): random 320-bit x, one-limb y, fitting
	// (Rust's addition).
	for i := 0; i < 4000; i++ {
		v := rng.next()
		y := v >> (rng.next() % 64)
		x := randomBits(rng, int(rng.next()%321))
		if bigMul(x, bigU64(y)).Cmp(two320) < 0 {
			c.expect(bigOf320(mul_320_320x64(u320Of(x), int64(y))).Cmp(bigMul(x, bigU64(y))) == 0,
				func() string { return fmt.Sprintf("mul64 %v %d", x, y) })
		}
	}
	c.done("multiplies")
}

// --- powers of ten, every k ---

func TestU320MulPow10EveryKAndWidthClass(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{3}
	two320 := u320Two320()
	for k := 0; k <= 96; k++ {
		p := bigTen(k)
		c.expect(bigOf320(mulPow10_320(u320FromDw0(1), k)).Cmp(p) == 0,
			func() string { return fmt.Sprintf("10^%d", k) })
		// The largest x whose product still fits, and random x below it, of
		// every limb length that fits.
		maxX := bigQuo(bigSub(two320, bigOne()), p)
		xs := []*big.Int{new(big.Int), bigOne(), maxX, bigQuo(maxX, big.NewInt(10))}
		for i := 0; i < 60; i++ {
			xs = append(xs, randomBelow(rng, bigAdd(maxX, bigOne())))
		}
		for _, nb := range []int{1, 63, 64, 65, 127, 128, 129, 160, 191, 192, 193, 255, 256, 257} {
			if nb > maxX.BitLen() {
				continue
			}
			x := randomBits(rng, nb)
			if x.Cmp(maxX) <= 0 {
				xs = append(xs, x)
			}
		}
		for _, x := range xs {
			c.expect(bigOf320(mulPow10_320(u320Of(x), k)).Cmp(bigMul(x, p)) == 0,
				func() string { return fmt.Sprintf("x=%v k=%d", x, k) })
		}
	}
	c.done("mulPow10EveryKAndWidthClass")
}

func TestU320DigitLenAndComparesEveryK(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{4}
	c.expect(digitLen_320(u320{}) == 0, func() string { return "digitLen(0)" })
	for k := 0; k <= 96; k++ {
		p := bigTen(k)
		c.expect(digitLen_320(u320Of(p)) == k+1, func() string { return fmt.Sprintf("10^%d", k) })
		if k > 0 {
			c.expect(digitLen_320(u320Of(bigSub(p, bigOne()))) == k, func() string { return fmt.Sprintf("10^%d-1", k) })
		}
		c.expect(cmp_int_320_pow10(u320Of(p), k) == 0, func() string { return fmt.Sprintf("cmp 10^%d", k) })
		c.expect(cmp_int_320_pow10(u320Of(bigAdd(p, bigOne())), k) == 1, func() string { return fmt.Sprintf("cmp 10^%d+1", k) })
		if k > 0 {
			c.expect(cmp_int_320_pow10(u320Of(bigSub(p, bigOne())), k) == -1, func() string { return fmt.Sprintf("cmp 10^%d-1", k) })
		}
		// A value above 10^k whose extra width sits in limbs 10^k doesn't use.
		for _, nb := range []int{65, 129, 193, 257, 320} {
			if nb <= p.BitLen() {
				continue
			}
			x := randomBits(rng, nb)
			c.expect(cmp_int_320_pow10(u320Of(x), k) == 1, func() string { return fmt.Sprintf("cmp %v 10^%d", x, k) })
		}
	}
	for i := 0; i < 5000; i++ {
		x := randomBits(rng, int(rng.next()%321))
		want := 0
		if x.Sign() != 0 {
			want = len(x.String())
		}
		c.expect(digitLen_320(u320Of(x)) == want, func() string { return fmt.Sprintf("digitLen %v", x) })
	}
	c.expect(digitLen_320(u320Of(bigSub(u320Two320(), bigOne()))) == 97, func() string { return "digitLen 2^320-1" })
	c.done("digitLenAndComparesEveryK")
}

func TestU320ResiduesAgainstPow10AndDivisor(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{5}
	for p := 0; p <= 48; p++ {
		d := bigTen(p)
		rems := []*big.Int{new(big.Int), bigSub(d, bigOne())}
		if p > 0 {
			half := bigQuo(d, big.NewInt(2))
			rems = append(rems, bigOne(), bigSub(half, bigOne()), half, bigAdd(half, bigOne()))
		}
		for i := 0; i < 50; i++ {
			rems = append(rems, randomBelow(rng, d))
		}
		for _, r := range rems {
			if r.Cmp(d) >= 0 {
				continue
			}
			c.expect(residueFromRemainderPow10_320(u320Of(r), p) == expectedResidue(r, d),
				func() string { return fmt.Sprintf("p=%d r=%v", p, r) })
		}
	}
	for i := 0; i < 5000; i++ {
		d := randomBits(rng, int(rng.next()%318)+1)
		r := randomBelow(rng, d)
		if rng.next()%4 == 0 {
			r = bigQuo(d, big.NewInt(2))
		}
		c.expect(residueFromRemainderDivisor_320(u320Of(r), u320Of(d)) == expectedResidue(r, d),
			func() string { return fmt.Sprintf("d=%v r=%v", d, r) })
	}
	c.done("residuesAgainstPow10AndDivisor")
}

// --- recipmul divPow10-320 ---

func checkDivPow10_320(c *expectCounter, x *big.Int, k int) {
	g := divPow10_q320res_320_recipmul(u320Of(x), k)
	d := bigTen(k)
	c.expect(bigOf320(g.quot()).Cmp(bigQuo(x, d)) == 0, func() string { return fmt.Sprintf("quot x=%v k=%d", x, k) })
	c.expect(g.residue == expectedResidue(bigRem(x, d), d), func() string { return fmt.Sprintf("residue x=%v k=%d", x, k) })
	// The fixed-k entries equal the general kernel (Rust's addition).
	var f quot320Residue
	switch k {
	case 48:
		f = divPow10_q320res_320_recipmul_48(u320Of(x))
	case 47:
		f = divPow10_q320res_320_recipmul_47(u320Of(x))
	case 1:
		f = divPow10_q320res_320_recipmul_1(u320Of(x))
	default:
		return
	}
	c.expect(f == g, func() string { return fmt.Sprintf("fixed-k %d x=%v", k, x) })
}

func TestU320RecipmulBoundaryAndRandomEveryK(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{6}
	pow10_96 := bigTen(96)
	xHi := bigSub(pow10_96, bigOne())
	for k := 1; k <= 96; k++ {
		d := bigTen(k)
		half := bigQuo(d, big.NewInt(2))
		xs := []*big.Int{new(big.Int), bigOne(), bigSub(d, bigOne()), d, bigAdd(d, bigOne()), half, bigAdd(half, bigOne()),
			xHi, bigSub(xHi, bigOne()), bigSub(xHi, half)}
		for i := 0; i < 40; i++ {
			b := randomBelow(rng, bigTen(96-k))
			bd := bigMul(b, d)
			xs = append(xs, bd, bigAdd(bd, half), bigSub(bigAdd(bd, half), bigOne()), bigAdd(bigAdd(bd, half), bigOne()),
				bigSub(bigAdd(bd, d), bigOne()))
		}
		for i := 0; i < 200; i++ {
			xs = append(xs, randomBelow(rng, pow10_96))
		}
		for _, x := range xs {
			if (x.Sign() >= 0) && (x.Cmp(xHi) <= 0) {
				checkDivPow10_320(c, x, k)
			}
		}
	}
	c.done("recipmulBoundaryAndRandomEveryK")
}

func TestU320DivPow10Wrappers(t *testing.T) {
	c := &expectCounter{t: t}
	x := u320Of(bigSub(bigTen(96), bigOne()))
	c.expect(divPow10_q320res(x, 0).quot() == x, func() string { return "k=0 quot" })
	c.expect(divPow10_q320res(x, 0).residue == RESIDUE_EXACT, func() string { return "k=0 residue" })
	far := divPow10_checked_q320res(x, 200)
	c.expect(far.quot().isZero() && (far.residue == RESIDUE_LT_HALF), func() string { return "k=200" })
	c.expect(divPow10_checked_q320res(u320{}, 200).residue == RESIDUE_EXACT, func() string { return "0, k=200" })
	c.expect(divPow10_checked_q320res(x, 96).residue == RESIDUE_GT_HALF, func() string { return "k=96" })
	c.expect(divPow10_q320res(u320FromDw0(1250), 2).residue == RESIDUE_HALF, func() string { return "1250, k=2" })
	c.done("divPow10Wrappers")
}

// --- 320 ÷ 160 MG divide ---

func checkDiv320(c *expectCounter, x, y *big.Int) {
	g := divRem_320_160_mg(u320Of(x), u192Of(y))
	c.expect(bigOf320(g.quot()).Cmp(bigQuo(x, y)) == 0, func() string { return fmt.Sprintf("quot x=%v y=%v", x, y) })
	c.expect(bigOf192(g.rem()).Cmp(bigRem(x, y)) == 0, func() string { return fmt.Sprintf("rem x=%v y=%v", x, y) })
}

func TestU320DivRem320_160Random(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{160}
	two320 := u320Two320()
	ones := []uint64{0, 1, ^uint64(0), ^uint64(0) - 1, 1 << 63}
	for it := 0; it < 6000; it++ {
		nb := int(rng.next()%160) + 1
		// randomBits sets the top bit, so every y is exactly nb wide; nb a
		// multiple of 64 gives a top limb with its top bit set.
		y := randomBits(rng, nb)
		// Quotients with all-ones / boundary limbs drive the add-back and the
		// n2:n1 == d1:d0 paths.
		qv := new(big.Int)
		for i := 0; i < 5; i++ {
			var l uint64
			if rng.next()&1 == 0 {
				l = ones[rng.next()%uint64(len(ones))]
			} else {
				l = rng.next()
			}
			qv = bigAdd(bigShl(qv, 64), bigU64(l))
		}
		r := randomBelow(rng, y)
		prod := bigAdd(bigMul(qv, y), r)
		if prod.Cmp(two320) < 0 {
			checkDiv320(c, prod, y)
		}
		checkDiv320(c, randomBits(rng, int(rng.next()%320)+1), y)
		exact := bigMul(y, randomBits(rng, int(rng.next()%160)+1))
		if exact.Cmp(two320) < 0 {
			checkDiv320(c, exact, y)
		}
	}
	checkDiv320(c, bigSub(two320, bigOne()), bigSub(bigPow2(160), bigOne()))
	checkDiv320(c, bigSub(two320, bigOne()), bigOne())
	checkDiv320(c, new(big.Int), big.NewInt(7))
	c.done("divRem320_160Random")
}

// The add-back and the n2:n1 == d1:d0 paths need a three-limb divisor (129-160
// bits) and numerators random inputs reach with probability ~2^-63, so build
// them in normalized space: yn has its top bit set and `shift` zero low bits.
func TestU320DivRem320_160AddBackAndAllOnesLimb(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{3}
	two320 := u320Two320()
	two64 := bigPow2(64)
	addBack, allOnes := 0, 0
	for it := 0; it < 20000; it++ {
		nb := 129 + int(rng.next()%32)
		shift := 192 - nb // 32..63
		yn := randomBits(rng, 192)
		yn = bigSub(yn, bigRem(yn, bigPow2(shift))) // clear the low `shift` bits
		y := bigShr(yn, shift)
		top := bigSub(yn, bigRem(yn, two64)) // the two limbs DIV3BY2 sees
		var xn *big.Int
		if it%2 == 0 {
			// Exact multiple of the top limbs + a small tail: the estimate is one
			// too large once the low limb is subtracted.
			qb := randomBits(rng, 128)
			q := bigShr(qb, int(rng.next()%64))
			xn = bigAdd(bigMul(q, top), bigU64(rng.next()))
			addBack++
		} else {
			// Remainder = yn - eps (eps below the low limb) brought down one
			// limb: its top two limbs equal the divisor's.
			lowLimb := bigRem(yn, two64)
			epsRange := bigShr(lowLimb, shift)
			eps := new(big.Int)
			if epsRange.Sign() > 0 {
				eps = randomBelow(rng, epsRange)
			}
			eps = bigShl(eps, shift)
			t1 := bigShl(bigSub(yn, eps), 64)
			t2 := bigU64(rng.next())
			t3 := bigMul(bigU64(rng.next()&0xFF), bigShl(yn, 128))
			xn = bigAdd(bigAdd(t1, t2), t3)
			allOnes++
		}
		xn = bigSub(xn, bigRem(xn, bigPow2(shift))) // clear the low `shift` bits
		xs := bigShr(xn, shift)
		if xs.Cmp(two320) >= 0 {
			continue
		}
		checkDiv320(c, xs, y)
	}
	c.expect((addBack > 0) && (allOnes > 0), func() string { return "no add-back / all-ones cases" })
	c.done("divRem320_160AddBackAndAllOnesLimb")
}

// --- isqrt_320 ---

func checkSqrt320(c *expectCounter, r *big.Int) {
	s := bigOf192(isqrt_320(u320Of(r)))
	c.expect(bigMul(s, s).Cmp(r) <= 0, func() string { return fmt.Sprintf("s^2 > r for r=%v", r) })
	s1 := bigAdd(s, bigOne())
	c.expect(bigMul(s1, s1).Cmp(r) > 0, func() string { return fmt.Sprintf("(s+1)^2 <= r for r=%v", r) })
}

func TestU320Isqrt320(t *testing.T) {
	c := &expectCounter{t: t}
	rng := &splitMix{96}
	pow10_96 := bigTen(96)
	two320 := u320Two320()
	for i := 0; i < 5000; i++ {
		checkSqrt320(c, randomBits(rng, int(rng.next()%320)+1))
		// Perfect squares and their neighbours.
		s := bigShr(randomBits(rng, 160), i%160)
		sq := bigMul(s, s)
		c.expect(bigOf192(isqrt_320(u320Of(sq))).Cmp(s) == 0, func() string { return fmt.Sprintf("isqrt(%v^2)", s) })
		if sq.Sign() > 0 {
			c.expect(bigOf192(isqrt_320(u320Of(bigSub(sq, bigOne())))).Cmp(bigSub(s, bigOne())) == 0,
				func() string { return fmt.Sprintf("isqrt(%v^2-1)", s) })
		}
		c.expect(bigOf192(isqrt_320(u320Of(bigAdd(sq, bigShl(s, 1))))).Cmp(s) == 0,
			func() string { return fmt.Sprintf("isqrt(%v^2+2s)", s) })
	}
	for _, r := range []*big.Int{new(big.Int), bigOne(), big.NewInt(2), big.NewInt(3), big.NewInt(4),
		bigSub(pow10_96, bigOne()), bigSub(two320, bigOne()), bigSub(bigPow2(128), bigOne()), bigPow2(128)} {
		checkSqrt320(c, r)
	}
	c.done("isqrt320")
}
