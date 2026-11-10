// D48 square root (round-half-even) via the exact scaled integer floor root
// (Swift D48Sqrt.swift).

package decimal128

func d48_sqrt(x d48) d48 {
	if d48_isZero(x) {
		// sqrt(±0) = +0 with halved exponent; the internal domain has no −0.
		return d48_fromSignQExp_160(false, d48_qExp(x)>>1, u192{})
	}
	verify("", func() bool { return !d48_signFlag(x) })
	f := sqrtFloorScaled48(d48_u160(x), d48_qExp(x), d48_digitLen(x))
	return d48_finalize_320(false, f.resultQExp, f.s, f.residue)
}
