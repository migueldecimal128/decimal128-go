// Möller & Granlund 2011 ("Improved division by invariant integers") reciprocal
// division primitives (tier-1). Each division STEP estimates a quotient word by
// multiplying the numerator by a precomputed reciprocal v, then applies ≤2
// compare-subtract corrections — no hardware divide in the step. One 128÷64
// divide remains in the reciprocal build.
//
// Algorithm numbering (M&G): Alg. 3 = single-word reciprocal; Alg. 4 = DIV2BY1
// (2-word ÷ 1-word); Alg. 5 = DIV3BY2 (3-word ÷ 2-word). Swift uses native
// UInt128 for the exact 128-bit intermediates; Go synthesizes them from uint64
// limbs via math/bits.

package decimal128

import "math/bits"

// vReciprocal_64_64_mg returns the single-word reciprocal v of a normalized
// 1-word divisor d (top bit set): v = low 64 bits of floor((2^128 - 1) / d),
// i.e. floor((2^128-1)/d) - 2^64. One 128÷64 divide. Precondition: d >= 2^63.
func vReciprocal_64_64_mg(d int64) int64 {
	dd := uint64(d)
	x := (0 - dd) - 1
	qLo, _ := bits.Div64(x, ^uint64(0), dd)
	return int64(qLo)
}

// vReciprocal_64_128_mg returns the single-word reciprocal v of the normalized
// 2-word divisor (d1:d0) (d1's top bit set). One 128÷64 divide. Precondition:
// d1 >= 2^63.
func vReciprocal_64_128_mg(d1, d0 int64) int64 {
	D1 := uint64(d1)
	D0 := uint64(d0)
	v := uint64(vReciprocal_64_64_mg(int64(D1)))
	p := D1*v + D0
	if p < D0 {
		v = v - 1
		if p >= D1 {
			v = v - 1
			p = p - D1
		}
		p = p - D1
	}
	t1, t0 := bits.Mul64(D0, v)
	p = p + t1
	if p < t1 {
		v = v - 1
		if (p >= D1) && ((p > D1) || (t0 >= D0)) {
			v = v - 1
		}
	}
	return int64(v)
}

// divRem_q64r64_128x64x64_v_mg divides the 2-word numerator (u1:u0) by the
// normalized 1-word divisor d (top bit set) using the precomputed reciprocal v.
// Requires u1 < d. Returns the 64-bit quotient and the < d remainder. M&G Alg. 4.
func divRem_q64r64_128x64x64_v_mg(u1, u0, d, v int64) quot64Rem64 {
	U1 := uint64(u1)
	U0 := uint64(u0)
	D := uint64(d)
	V := uint64(v)

	// (q : q0) = v*u1 + (u1 : u0); then q += 1.
	pHi, pLo := bits.Mul64(U1, V)
	qLo, c := bits.Add64(pLo, U0, 0)
	qHi, _ := bits.Add64(pHi, U1, c)
	q := qHi
	q0 := qLo
	q = q + 1
	r := U0 - q*D

	if r > q0 {
		q = q - 1
		r = r + D
	}
	if r >= D {
		q = q + 1
		r = r - D
	}
	return quot64Rem64{int64(q), int64(r)}
}

// divRem_q64r128_192x128x64_v_mg divides the 3-word numerator (n2:n1:n0) by the
// normalized 2-word divisor (d1:d0) using the precomputed reciprocal v. Requires
// (n2:n1) < (d1:d0). Returns the 64-bit quotient and the < d 128-bit remainder.
// M&G Alg. 5.
func divRem_q64r128_192x128x64_v_mg(n2, n1, n0, d1, d0, v int64) quot64Rem128 {
	N2 := uint64(n2)
	N1 := uint64(n1)
	N0 := uint64(n0)
	D1 := uint64(d1)
	D0 := uint64(d0)
	V := uint64(v)

	// (q : q0) = n2*v + (n2 : n1)
	pHi, pLo := bits.Mul64(N2, V)
	qLo, c := bits.Add64(pLo, N1, 0)
	qHi, _ := bits.Add64(pHi, N2, c)
	q := qHi
	q0 := qLo

	// rem = (n1 : n0) - q*d - d   (low 128 bits exact; true rem < d)
	pdHi, pdLo := bits.Mul64(q, D0)
	pdHi = pdHi + q*D1
	remLo, b := bits.Sub64(N0, pdLo, 0)
	remHi, _ := bits.Sub64(N1, pdHi, b)
	remLo, b = bits.Sub64(remLo, D0, 0)
	remHi, _ = bits.Sub64(remHi, D1, b)
	q = q + 1

	// Correction 1: estimate was one too large.
	if remHi >= q0 {
		q = q - 1
		remLo, c = bits.Add64(remLo, D0, 0)
		remHi, _ = bits.Add64(remHi, D1, c)
	}
	// Correction 2: still one short (rare). rem >= d (128-bit compare).
	if remHi > D1 || (remHi == D1 && remLo >= D0) {
		q = q + 1
		remLo, b = bits.Sub64(remLo, D0, 0)
		remHi, _ = bits.Sub64(remHi, D1, b)
	}
	return quot64Rem128{int64(q), int64(remLo), int64(remHi)}
}
