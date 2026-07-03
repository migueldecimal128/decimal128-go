// Correctly-rounded square root in the 38-digit D38 domain, used by the ln /
// log10 Padé argument reduction. Same exact-integer floor-root method as the
// D128 squareRoot, lowered to 38 digits: scale the coefficient to a 75–76 digit
// integer R whose exact floor root is exactly 38 digits, then round half-even
// from the exact remainder. No preferred-exponent reduction.

package decimal128

func d38_sqrt(x d38) d38 {
	if d38_isZero(x) {
		// √(±0) = +0 with halved exponent; internal domain has no signed zero.
		return d38Pack(false, d38_qExp(x)>>1, 0, 0)
	}
	verify("", func() bool { return !d38_signFlag(x) })

	f := sqrtFloorScaled(d38_u128(x), d38_qExp(x), d38_digitLen(x))
	return finalize38_256(false, f.resultQExp, f.s, f.residue)
}
