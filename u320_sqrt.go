// Exact integer floor square root for the D48 sqrt (Swift U320Sqrt.swift; kept
// separate from the shared u256 sqrtFloorScaled, which d128_sqrt keeps using
// unchanged). Scales the coefficient into a 95-96 digit integer R whose exact
// floor root is 48 digits, then classifies the exact remainder. The root is the
// Newton isqrt_320 kernel.

package decimal128

type sqrtFloor48 struct {
	resultQExp int
	s          u320
	residue    residue
}

func sqrtFloorScaled48(coeff u192, qExp, digitLen int) sqrtFloor48 {
	verify("", func() bool { return !coeff.isZero() })
	scaleUp := 96 - digitLen
	if ((scaleUp - qExp) & 1) != 0 {
		scaleUp -= 1 // keep (scaleUp − qExp) even
	}
	r := mulPow10_320(u320FromU192(coeff), scaleUp)

	s := u320FromU192(isqrt_320(r))                                // exact: s² ≤ R < (s+1)²
	rem := sub_320_320x320(r, mul_320_192x192(s.u192(), s.u192())) // 0 ≤ rem ≤ 2s

	// rem == 0 → exact; rem ≤ s → below half; else above half. A HALF tie is
	// impossible: (s+½)² = s²+s+¼ and rem is an integer.
	var res residue
	if rem.isZero() {
		res = RESIDUE_EXACT
	} else if cmp_int_320x320(rem, s) <= 0 {
		res = RESIDUE_LT_HALF
	} else {
		res = RESIDUE_GT_HALF
	}

	resultQExp := (qExp - scaleUp) >> 1 // (qExp − scaleUp) is even ⇒ exact
	return sqrtFloor48{resultQExp: resultQExp, s: s, residue: res}
}
