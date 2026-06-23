// Exact integer floor square root over U256, shared by the D128 squareRoot and
// (later) the D38 sqrt. Scales the coefficient to a 75–76 digit integer R (parity
// chosen so (scaleUp − qExp) is even, keeping the exponent halving exact), takes
// its exact floor root s = floor(√R) (always 38 digits), and derives the round
// residue from the exact remainder.

package decimal128

import "math"

type sqrtFloor struct {
	resultQExp int
	s          u256
	residue    residue
}

func sqrtFloorScaled(coeff u128, qExp, digitLen int) sqrtFloor {
	scaleUp := 76 - digitLen
	if ((scaleUp - qExp) & 1) != 0 {
		scaleUp -= 1 // keep (scaleUp − qExp) even
	}
	R := mulPow10_256_128(coeff.dw0, coeff.dw1, scaleUp)

	s := isqrtFloor(R)
	rem := subU(R, sqrU(s)) // exact remainder, 0 <= rem <= 2s

	// rem==0 exact; rem<=s below half; else above half. A HALF tie is impossible.
	var res residue
	if rem == (u256{}) {
		res = RESIDUE_EXACT
	} else if !s.less(rem) { // rem <= s
		res = RESIDUE_LT_HALF
	} else {
		res = RESIDUE_GT_HALF
	}

	resultQExp := (qExp - scaleUp) >> 1 // (qExp − scaleUp) even ⇒ exact
	return sqrtFloor{resultQExp, s, res}
}

// isqrtFloor returns the unique s with s² <= R < (s+1)². Double-seeded integer
// Newton (one unconditional step makes the iterate an overestimate), then a
// bidirectional ±1 pin guarantees the exact floor.
func isqrtFloor(R u256) u256 {
	one := u256{1, 0, 0, 0}
	seed := u256FromU128(u128FromDouble(math.Sqrt(u256ToFloorDouble(R))))
	x := seed
	if seed.less(one) {
		x = one
	}
	x = shr1U(addU(x, divU(R, x))) // now x >= floor(√R)
	for {
		y := shr1U(addU(x, divU(R, x)))
		if !y.less(x) { // y >= x
			break
		}
		x = y
	}
	for R.less(sqrU(x)) { // sqrU(x) > R
		x = subU(x, one)
	}
	for !R.less(sqrU(addU(x, one))) { // sqrU(x+1) <= R
		x = addU(x, one)
	}
	return x
}

func u256FromU128(u u128) u256 { return u256{u.dw0, u.dw1, 0, 0} }

func sqrU(x u256) u256 {
	return mul_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, x.dw0, x.dw1, x.dw2, x.dw3)
}
func addU(x, y u256) u256 {
	return add_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3)
}
func subU(x, y u256) u256 {
	return sub_256_256x256(x.dw0, x.dw1, x.dw2, x.dw3, y.dw0, y.dw1, y.dw2, y.dw3)
}
func shr1U(x u256) u256 {
	return shr1_256_256(x.dw0, x.dw1, x.dw2, x.dw3)
}

// divU computes R / x where x = floor(√R) is <= 38 digits < 2^127, so it always
// fits U128 and takes the 256÷128 path.
func divU(R, x u256) u256 {
	verify("", isZero_bool_128(x.dw2, x.dw3))
	qr := u256DivRem128(R, x.lo())
	return u256{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3}
}
