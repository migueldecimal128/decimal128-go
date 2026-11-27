// The reciprocals of 10^j, j in [1, 34], for the ties-to-even roundings that
// divide a wide value by a power of ten in one multiply (d128_multiply_tte;
// GoUbdKernelGapWorkOrder.md L1).
//
// The division by d = 10^j (1 <= j <= 76) is by a reciprocal. With l = bitLen(d) - 1,
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
// j in [1, 76]; const_RECIP10_128_S[j] is its s. Up to j = 34 the divisor and the
// remainder fit two words; past it (divRemPow10_256) they take four, and the bound
// on m and the proof above are unchanged. Index 0 and the padding are unused;
// every index is masked, so each read is provably in bounds.
var const_RECIP10_128_M = [256]int64{
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
	-8668647847015794858, 187072209578355573, -3245569462870725564, 149657767662684458, 2185786488890659745, 239452428260295134,
	5437978005854438119, 191561942608236107, -3028315224800270151, 153249554086588885, 6222742084545298728, 245199286538542217,
	-2400503961847581664, 196159429230833773, 1768945645263844992, 156927543384667019, -8237733411803578982, 251084069415467230,
	8167208529524778107, 200867255532373784, -8223628435347818807, 160693804425899027, 1599589762411131201, 257110087081438444,
	4969020624670815284, 205688069665150755, 3975216499736652227, 164550455732120604, -4707700044647087406, 263280729171392966,
	7301886408508061045, 210624583337114373, -5226537317419282134, 168499666669691498, -983762078387030767, 269599466671506397,
	6591687966774196032, 215679573337205118, -5794696070806374144, 172543658669764094, 9175230360419352986, 276069853871622551,
	3650835473593572066, 220855883097298041, -768680435867052671, 176684706477838432, -4919237512129194596, 282695530364541492,
	3443307619780464969, 226156424291633194, 6443994910566282299, 180925139433306555, 5155195928453025839, 144740111546645244,
	-2819732958700889627, 231584178474632390, -5945135181702622025, 185267342779705912, 2622589484121723026, 148213874223764730,
	4196143174594756842, 237142198758023568, -7711131904549925496, 189713759006418854, -6168905523639940397, 151771007205135083,
	-2491551208340083988, 242833611528216133, 1696107848069843132, 194266889222572907, -6021811351027946141, 155413511378058325,
	1433148282581017145, 248661618204893321, -2542830188677096607, 198929294563914656, 9033782293284053684, 159143435651131725,
	-3992692404455065722, 254629497041810760, -6883502738305962901, 203703597633448608, -1817453375902859997, 162962878106758886,
	781423413297334328, 260740604970814219, 4314487545379777785, 208592483976651375, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
	0, 0, 0, 0,
}

var const_RECIP10_128_S = [128]uint8{
	0, 0, 0, 3, 7, 10, 13, 17, 20, 23, 27, 30, 33, 37, 40, 43, 47, 50, 53, 57,
	60, 63, 67, 70, 73, 77, 80, 83, 87, 90, 93, 96, 100, 103, 106, 110, 113, 116, 120, 123,
	126, 130, 133, 136, 140, 143, 146, 150, 153, 156, 160, 163, 166, 170, 173, 176, 180, 183, 186, 189,
	193, 196, 199, 203, 206, 209, 213, 216, 219, 223, 226, 229, 233, 236, 239, 243, 246, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
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

// pow10_4 returns 10^t, t in [0, 77], as four words.
func pow10_4(t int) (w0, w1, w2, w3 uint64) {
	if t <= 38 {
		return uint64(pow10_128_dw0(t)), uint64(pow10_128_dw1(t)), 0, 0
	}
	return uint64(pow10_256_dw0(t - 39)), uint64(pow10_256_dw1(t - 39)), uint64(pow10_256_dw2(t - 39)), uint64(pow10_256_dw3(t - 39))
}

// digitLen_u256 is the number of decimal digits of (p3:p2:p1:p0), at most 77 (0 for
// 0): the estimate from the bit length is the count or one short, and it is short
// exactly when the value less 10^t does not borrow. The bit length is taken by
// selects.
func digitLen_u256(p0, p1, p2, p3 uint64) int {
	n := bits.Len64(p0)
	if p1 != 0 {
		n = 64 + bits.Len64(p1)
	}
	if p2 != 0 {
		n = 128 + bits.Len64(p2)
	}
	if p3 != 0 {
		n = 192 + bits.Len64(p3)
	}
	t := (n * 1233) >> 12
	w0, w1, w2, w3 := pow10_4(t)
	_, b := bits.Sub64(p0, w0, 0)
	_, b = bits.Sub64(p1, w1, b)
	_, b = bits.Sub64(p2, w2, b)
	_, b = bits.Sub64(p3, w3, b)
	return t + 1 - int(b)
}

// divRemPow10_256 returns (p3:p2:p1:p0) / 10^j and the remainder, for j in [1, 76]
// and a quotient below 2^120: the reciprocal above, with the divisor and the
// remainder in four words, and one correction.
func divRemPow10_256(p0, p1, p2, p3 uint64, j int) (q0, q1, r0, r1, r2, r3 uint64) {
	d0, d1, d2, d3 := pow10_4(j)
	// a = p >> s, below 2^127: words w and w + 1 of p shifted by sh (s < 248,
	// so w <= 3; the words past p3 are zero).
	s := uint(const_RECIP10_128_S[j&127])
	w, sh := s>>6, s&63
	u := [6]uint64{p0, p1, p2, p3, 0, 0}
	a0 := u[w]>>sh | u[w+1]<<(64-sh)
	a1 := u[w+1]>>sh | u[w+2]<<(64-sh)
	m0 := uint64(const_RECIP10_128_M[(j<<1)&255])
	m1 := uint64(const_RECIP10_128_M[((j<<1)|1)&255])
	h00, _ := bits.Mul64(a0, m0)
	h01, l01 := bits.Mul64(a0, m1)
	h10, l10 := bits.Mul64(a1, m0)
	h11, l11 := bits.Mul64(a1, m1)
	mid, c1 := bits.Add64(h00, l01, 0)
	_, c2 := bits.Add64(mid, l10, 0)
	q0, c1 = bits.Add64(h01, h10, c1)
	q0, c2 = bits.Add64(q0, l11, c2)
	q1 = h11 + c1 + c2
	// r = p - q * d, modulo 2^256.
	t1, t0 := bits.Mul64(q0, d0)
	g1, f1 := bits.Mul64(q0, d1)
	g2, f2 := bits.Mul64(q0, d2)
	e1, k1 := bits.Mul64(q1, d0)
	e2, k2 := bits.Mul64(q1, d1)
	var c uint64
	x1, c := bits.Add64(t1, f1, 0)
	x2, c3 := bits.Add64(g1, f2, c)
	x3 := g2 + q0*d3 + c3
	x1, c = bits.Add64(x1, k1, 0)
	x2, c3 = bits.Add64(x2, e1, c)
	x3 += c3
	x2, c = bits.Add64(x2, k2, 0)
	x3 += e2 + q1*d2 + c
	r0, c = bits.Sub64(p0, t0, 0)
	r1, c = bits.Sub64(p1, x1, c)
	r2, c = bits.Sub64(p2, x2, c)
	r3, _ = bits.Sub64(p3, x3, c)
	_, lt := bits.Sub64(r0, d0, 0)
	_, lt = bits.Sub64(r1, d1, lt)
	_, lt = bits.Sub64(r2, d2, lt)
	_, lt = bits.Sub64(r3, d3, lt)
	ge := lt ^ 1 // 1 when the remainder reaches d
	q0, c = bits.Add64(q0, ge, 0)
	q1 += c
	r0, c = bits.Sub64(r0, d0&-ge, 0)
	r1, c = bits.Sub64(r1, d1&-ge, c)
	r2, c = bits.Sub64(r2, d2&-ge, c)
	r3, _ = bits.Sub64(r3, d3&-ge, c)
	return
}
