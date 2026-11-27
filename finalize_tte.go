// The reciprocals of 10^j, j in [1, 34], for the ties-to-even roundings that
// divide a wide value by a power of ten in one multiply (d128_multiply_tte;
// GoUbdKernelGapWorkOrder.md L1).
//
// The division by d = 10^j (1 <= j <= 34) is by a reciprocal. With l = bitLen(d) - 1,
// s = max(l - 6, 0) and m = floor(2^(128+s) / d), for a dividend p < d * 2^120 and
// a = floor(p / 2^s), the estimate qe = floor(a * m / 2^128) is q = floor(p / d) or
// q - 1:
//   - a < 2^127: for s = l - 6, a < d * 2^(120-s) < 2^(l+1+120-l+6); for s = 0,
//     d <= 100 < 2^7 and a = p < 2^127.
//   - m < 2^125, since m <= 2^(128+s) / 2^l.
//   - qe <= p / d, since a <= p / 2^s and m <= 2^(128+s) / d.
//   - p / d - a * m / 2^128 < (a + m + 1) / 2^128 < 1, so q < qe + 2.
// The remainder p - qe * d is then in [0, 2d), below 2^128 (2 * 10^34 < 2^115), so it
// is exact from the low 128 bits alone, and one comparison with d corrects qe. The
// quotients here are below 10^34 + 1 < 2^120, inside the bound.

package decimal128

import "math/bits"

// const_RECIP10_128_M[(j<<1)+0..1] is m = floor(2^(128+s) / 10^j), lo/hi dwords, for
// j in [1, 34]; const_RECIP10_128_S[j] is its s. Index 0 and the padding are unused;
// every index is masked, so each read is provably in bounds.
var const_RECIP10_128_M = [128]int64{
	0, 0, -7378697629483820647, 1844674407370955161, 2951479051793528258, 184467440737095516,
	-1328165573307087717, 147573952589676412, -5814413732033250670, 236118324143482260, -8340879800368510859, 188894659314785808,
	-2983355025552898364, 151115727451828646, -1084019226142727059, 241785163922925834, -867215380914181648, 193428131138340667,
	6684925324752475328, 154742504910672534, -372165924621770444, 247588007857076054, -297732739697416355, 198070406285660843,
	3451162622983977239, 158456325028528675, 5521860196774363582, 253530120045645880, 4417488157419490866, 202824096036516704,
	7223339340677503016, 162259276829213363, 7867994130342094502, 259614842926741381, 2605046489531765279, 207691874341393105,
	2084037191625412223, 166153499473114484, -7733586937625071413, 265845599156983174, -6186869550100057130, 212676479325586539,
	-4949495640080045704, 170141183460469231, -4229844209386162804, 272225893536750770, -7073224182250840566, 217780714829400616,
	5409467098425058517, 174224571863520493, 4965798542738183304, 278759314981632789, 7661987648932456966, 223007451985306231,
	2440241304404055249, 178405961588244985, 3904386087046488399, 285449538541191976, -565839945104719604, 228359630832953580,
	-4142020770825686006, 182687704666362864, -3313616616660548805, 146150163733090291, -1612437771914967765, 233840261972944466,
	-8668647847015794858, 187072209578355573, -3245569462870725564, 149657767662684458,
}

var const_RECIP10_128_S = [64]uint8{
	0, 0, 0, 3, 7, 10, 13, 17, 20, 23, 27, 30, 33, 37, 40, 43, 47, 50, 53, 57, 60, 63, 67, 70, 73, 77, 80, 83, 87, 90, 93, 96, 100, 103, 106,
}

// divRemPow10_recip128 returns (hi:lo) / 10^j and its remainder, for j in [1, 34]
// and a quotient below 2^120, by the reciprocal above and one correction.
func divRemPow10_recip128(lo, hi uint64, j int) (q0, q1, r0, r1 uint64) {
	d0, d1 := uint64(pow10_128_dw0(j)), uint64(pow10_128_dw1(j))
	s := uint(const_RECIP10_128_S[j&63])
	sh := s & 63
	a0, a1 := lo>>sh|hi<<(64-sh), hi>>sh
	if s >= 64 {
		a0, a1 = hi>>sh, 0
	}
	m0 := uint64(const_RECIP10_128_M[(j<<1)&127])
	m1 := uint64(const_RECIP10_128_M[((j<<1)|1)&127])
	h00, _ := bits.Mul64(a0, m0)
	h01, l01 := bits.Mul64(a0, m1)
	h10, l10 := bits.Mul64(a1, m0)
	h11, l11 := bits.Mul64(a1, m1)
	mid, c1 := bits.Add64(h00, l01, 0)
	_, c2 := bits.Add64(mid, l10, 0)
	q0, c1 = bits.Add64(h01, h10, c1)
	q0, c2 = bits.Add64(q0, l11, c2)
	q1 = h11 + c1 + c2
	rh, rl := bits.Mul64(q0, d0)
	rh += q0*d1 + q1*d0
	var c uint64
	r0, c = bits.Sub64(lo, rl, 0)
	r1, _ = bits.Sub64(hi, rh, c)
	_, lt := bits.Sub64(r0, d0, 0)
	_, lt = bits.Sub64(r1, d1, lt)
	ge := lt ^ 1 // 1 when the remainder reaches d
	q0, c = bits.Add64(q0, ge, 0)
	q1 += c
	r0, c = bits.Sub64(r0, d0&-ge, 0)
	r1, _ = bits.Sub64(r1, d1&-ge, c)
	return
}
