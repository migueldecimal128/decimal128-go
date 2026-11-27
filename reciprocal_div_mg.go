// Moller & Granlund 2011 ("Improved division by invariant integers") reciprocal
// division primitives (tier-1, like barrettStep). Each division STEP estimates a
// quotient word by multiplying the numerator by a precomputed reciprocal v, then
// applies <= 2 compare-subtract corrections -- no hardware divide in the step. One
// 128/64 divide remains in the reciprocal build (vReciprocal_*); a table+Newton build
// would remove even that (MgDivideProductionMigrationWorkOrder.md Section 9).
// Algorithm numbering (M&G): Alg. 3 = single-word reciprocal; Alg. 4 = DIV2BY1 (2-word
// / 1-word); Alg. 5 = DIV3BY2 (3-word / 2-word). GMP symbols: invert_limb, invert_pi1,
// udiv_qrnnd_preinv, udiv_qr_3by2.
//
// In Go: the 128-bit intermediates are synthesized from uint64 limbs via math/bits.

package decimal128

import "math/bits"

// Single-word reciprocal `v` of a normalized 1-word divisor `d` (top bit set):
// `v = floor((2^128 - 1) / d) - 2^64`. The value DIV2BY1 (Alg. 4) consumes.
// One 128/64 divide. Precondition: `d` normalized (`d >= 2^63`).
func vReciprocal_64_64_mg(d int64) int64 {
	dd := uint64(d)
	x := (0 - dd) - 1
	qLo, _ := bits.Div64(x, ^uint64(0), dd)
	return int64(qLo)
}

// Single-word reciprocal `v` of the normalized 2-word divisor `(d1:d0)` (d1's
// top bit set): extends the Alg. 3 single-word reciprocal of `d1` to the
// two-word divisor. Returns the value DIV3BY2 (Alg. 5) consumes. One 128/64
// divide. Precondition: `d1 >= 2^63`.
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

// Divide the 2-word numerator `(u1:u0)` by the normalized 1-word divisor `d`
// (top bit set) using the precomputed reciprocal `v`. Requires `u1 < d`.
// Returns the 64-bit quotient and the `< d` remainder. No hardware divide.
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

// Divide the 3-word numerator `(n2:n1:n0)` by the normalized 2-word divisor
// `(d1:d0)` using the precomputed reciprocal `v`. Requires `(n2:n1) < (d1:d0)`.
// Returns the 64-bit quotient and the `< d` 128-bit remainder. No hardware divide.
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

// const_RECIP_V0[i] = floor((2^19 - 3*2^8) / (256 + i)): the 11-bit first
// approximation of Moller & Granlund Alg. 2, indexed by the top nine bits of a
// normalized divisor less 256.
var const_RECIP_V0 = [256]uint16{
	2045, 2037, 2029, 2021, 2013, 2005, 1998, 1990, 1983, 1975, 1968, 1960, 1953, 1946, 1938, 1931,
	1924, 1917, 1910, 1903, 1896, 1889, 1883, 1876, 1869, 1863, 1856, 1849, 1843, 1836, 1830, 1824,
	1817, 1811, 1805, 1799, 1792, 1786, 1780, 1774, 1768, 1762, 1756, 1750, 1745, 1739, 1733, 1727,
	1722, 1716, 1710, 1705, 1699, 1694, 1688, 1683, 1677, 1672, 1667, 1661, 1656, 1651, 1646, 1641,
	1636, 1630, 1625, 1620, 1615, 1610, 1605, 1600, 1596, 1591, 1586, 1581, 1576, 1572, 1567, 1562,
	1558, 1553, 1548, 1544, 1539, 1535, 1530, 1526, 1521, 1517, 1513, 1508, 1504, 1500, 1495, 1491,
	1487, 1483, 1478, 1474, 1470, 1466, 1462, 1458, 1454, 1450, 1446, 1442, 1438, 1434, 1430, 1426,
	1422, 1418, 1414, 1411, 1407, 1403, 1399, 1396, 1392, 1388, 1384, 1381, 1377, 1374, 1370, 1366,
	1363, 1359, 1356, 1352, 1349, 1345, 1342, 1338, 1335, 1332, 1328, 1325, 1322, 1318, 1315, 1312,
	1308, 1305, 1302, 1299, 1295, 1292, 1289, 1286, 1283, 1280, 1276, 1273, 1270, 1267, 1264, 1261,
	1258, 1255, 1252, 1249, 1246, 1243, 1240, 1237, 1234, 1231, 1228, 1226, 1223, 1220, 1217, 1214,
	1211, 1209, 1206, 1203, 1200, 1197, 1195, 1192, 1189, 1187, 1184, 1181, 1179, 1176, 1173, 1171,
	1168, 1165, 1163, 1160, 1158, 1155, 1153, 1150, 1148, 1145, 1143, 1140, 1138, 1135, 1133, 1130,
	1128, 1125, 1123, 1121, 1118, 1116, 1113, 1111, 1109, 1106, 1104, 1102, 1099, 1097, 1095, 1092,
	1090, 1088, 1086, 1083, 1081, 1079, 1077, 1074, 1072, 1070, 1068, 1066, 1064, 1061, 1059, 1057,
	1055, 1053, 1051, 1049, 1047, 1044, 1042, 1040, 1038, 1036, 1034, 1032, 1030, 1028, 1026, 1024,
}

// Single-word reciprocal `v = floor((2^128 - 1) / d) - 2^64` of a normalized 1-word
// divisor `d` (top bit set), without a divide: Moller & Granlund 2011, Alg. 2 (a
// table seed and three Newton steps, then one adjustment). The same value as
// vReciprocal_64_64_mg, whose 128/64 divide is a software routine on arm64.
func vReciprocal_64_64_mgNewton(d uint64) uint64 {
	d0 := d & 1
	d40 := d>>24 + 1
	d63 := d>>1 + d0
	v0 := uint64(const_RECIP_V0[(d>>55)&255])
	v1 := v0<<11 - v0*v0*d40>>40 - 1
	v2 := v1<<13 + v1*(1<<60-v1*d40)>>47
	e := (v2>>1)&-d0 - v2*d63 // modulo 2^64
	h, _ := bits.Mul64(v2, e)
	v3 := v2<<31 + h>>1
	h, l := bits.Mul64(v3, d)
	_, c := bits.Add64(l, d, 0)
	return v3 - (h + d + c)
}
