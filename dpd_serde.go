// DPD (Densely Packed Decimal) encode/decode for decimal128. On-the-wire format:
// 1 sign bit + 17-bit combination field G + 110-bit trailing significand T
// encoded as 11 declets of 10 bits. The declet codec is Cowlishaw's branchless
// dpd2bcd/bcd2dpd.

package decimal128

const (
	eighteenNines int64 = 999_999_999_999_999_999 // 10^18 - 1
	tenPow15U64   int64 = 1_000_000_000_000_000
	tenPow18U64   int64 = 1_000_000_000_000_000_000
)

func dpd128Decode(dpdHi, dpdLo int64) d128 {
	const bias = 6176
	const w = 12  // exponent continuation width
	const w5 = 17 // combination field width
	const t = 110 // trailing field width

	sign := ushr_64_64(dpdHi, 63) != 0
	combination := int(ushr_64_64(dpdHi, t-64)) & 0x1FFFF
	expContinuation := combination & 0x0FFF

	var d0 int64
	var qExp int
	isNaN := false

	top2 := combination >> (w5 - 2)
	top4 := combination >> (w5 - 4)
	top5 := combination >> (w5 - 5)

	if top2 != 0b11 {
		expMSB := top2
		biasedExponent := (expMSB << w) | expContinuation
		qExp = biasedExponent - bias
		d0 = int64((combination >> w) & 0b111)
	} else if top4 != 0b1111 {
		expMSB := (combination >> (w5 - 4)) & 0b11
		biasedExponent := (expMSB << w) | expContinuation
		qExp = biasedExponent - bias
		d0 = 8 + int64((combination>>w)&0b1)
	} else if top5 == 0b11110 {
		return d128_infinity(sign)
	} else {
		verify("", func() bool { return top5 == 0b11111 })
		isNaN = true
		d0 = 0
		qExp = 0
	}

	decletsHi6 := (d0 << 50) | ((dpdHi & 0x3FFF_FFFF_FFFF) << 4) | ushr_64_64(dpdLo, 60)
	decletsLo6 := dpdLo & 0x0FFF_FFFF_FFFF_FFFF

	binHi := binFromDecletsUpTo7(decletsHi6)
	binLo := binFromDecletsUpTo7(decletsLo6)

	var coeffLo, coeffHi int64
	if binHi == 0 {
		coeffLo = binLo
		coeffHi = 0
	} else {
		prodHi := mul_128_64x64(binHi, tenPow18U64)
		sum := add_128_128x128(prodHi.dw0, prodHi.dw1, binLo, 0)
		coeffLo = sum.dw0
		coeffHi = sum.dw1
	}

	if isNaN {
		isSignaling := ((combination >> (w5 - 6)) & 1) != 0
		return d128_nan_64x64(sign, isSignaling, coeffLo, coeffHi)
	}
	if isZero_bool_128(coeffLo, coeffHi) {
		return d128_zero(sign, qExp)
	}
	return decimalFNZ_64x64(sign, qExp, coeffLo, coeffHi)
}

func dpd128Encode(d d128) u128 {
	var coeff u128
	if d128_isFIN(d) {
		coeff = d128_u128(d)
	} else if d128_isNAN(d) {
		coeff = d128_nanPayload(d)
	} else {
		verify("", func() bool { return d128_isINF(d) })
		coeff = u128{0, 0}
	}
	verify("", func() bool { return isLt_pow10_34_bool_128(coeff.dw0, coeff.dw1) })

	dlen := digitLen_int_128(coeff.dw0, coeff.dw1)
	verify("", func() bool { return dlen <= 34 })

	mostSigBcd4 := 0
	var declets5Hi int64 = 0
	var binLo int64

	if dlen > 18 {
		qr := divRem_q128r64_128x64(coeff.dw0, coeff.dw1, tenPow18U64)
		binLo = qr.remDw0
		verify("", func() bool { return qr.quotDw1 == 0 })
		binHi := qr.quotDw0
		if ucmp_int_64x64(binHi, tenPow15U64) >= 0 {
			tt := udiv_64_64x64(binHi, tenPow15U64)
			binHi = binHi - (tt * tenPow15U64)
			mostSigBcd4 = int(tt)
			verify("", func() bool { return mostSigBcd4 >= 0 && mostSigBcd4 <= 9 })
		}
		verify("", func() bool { return ucmp_int_64x64(binHi, tenPow15U64) < 0 })
		declets5Hi = declets6FromBin(binHi)
	} else {
		verify("", func() bool { return coeff.dw1 == 0 })
		binLo = coeff.dw0
	}

	declets6Lo := declets6FromBin(binLo)
	signCombo := encodeSignAndGCombinationFieldDpd128(d, mostSigBcd4)
	dpdLo := (declets5Hi << 60) | declets6Lo
	dpdHi := (int64(signCombo) << 46) | ushr_64_64(declets5Hi, 4)
	return fromLoHi_128_64x64(dpdLo, dpdHi)
}

func encodeSignAndGCombinationFieldDpd128(d d128, mostSigBcd4 int) int {
	sBit := d128_signBit(d)
	var gField int

	if d128_isFIN(d) {
		verify("", func() bool { return mostSigBcd4 >= 0 && mostSigBcd4 <= 9 })
		biasedQExp := d128_qExp(d) + 6176
		expMSB := (biasedQExp >> 12) & 0x3
		expLSB := biasedQExp & 0x0FFF

		var gPrefix int
		if mostSigBcd4 <= 7 {
			gPrefix = (expMSB << 3) | (mostSigBcd4 & 0x7)
		} else {
			gPrefix = 0b11000 | (expMSB << 1) | (mostSigBcd4 & 0x1)
		}
		gField = (gPrefix << 12) | expLSB
	} else if d128_isINF(d) {
		gField = 0b11110 << 12
	} else {
		verify("", func() bool { return d128_isNAN(d) })
		isSignaling := 0
		if d128_isSNAN(d) {
			isSignaling = 1
		}
		gField = (0b11111 << 12) | (isSignaling << 11)
	}
	return (sBit << 17) | gField
}

// --- declet packing helpers ---

func binFromDecletsUpTo7(declets7 int64) int64 {
	var bin int64 = 0
	decletsT := declets7
	var multiplier int64 = 1
	for decletsT != 0 {
		declet := decletsT & 0x3FF
		bin1 := dpdDecodeDeclet(declet)
		bin = bin + (bin1 * multiplier)
		multiplier = multiplier * 1000
		decletsT = ushr_64_64(decletsT, 10)
	}
	return bin
}

func declets6FromBin(bin int64) int64 {
	verify("", func() bool { return ucmp_int_64x64(bin, eighteenNines) <= 0 })
	t := bin
	var declets6 int64 = 0
	shift := 0
	for t != 0 {
		q := ushr_64_64(umulHi_64_64x64(t, 0x020C49BA5E353F7D), 3)
		r := t - (q * 1000)
		declet := dpdEncodeDeclet(r)
		declets6 |= declet << uint(shift)
		shift += 10
		t = q
	}
	return declets6
}

// --- declet codec (Cowlishaw's tables, branchless) ---

func dpdDecodeDeclet(declet int64) int64 {
	verify("", func() bool { return ucmp_int_64x64(declet, 1023) <= 0 })
	if declet == 0 {
		return 0
	}
	p := ushr_64_64(declet, 9) & 1
	q := ushr_64_64(declet, 8) & 1
	r := ushr_64_64(declet, 7) & 1
	s := ushr_64_64(declet, 6) & 1
	t := ushr_64_64(declet, 5) & 1
	u := ushr_64_64(declet, 4) & 1
	v := ushr_64_64(declet, 3) & 1
	w := ushr_64_64(declet, 2) & 1
	x := ushr_64_64(declet, 1) & 1
	y := declet & 1

	ns := s ^ 1
	nx := x ^ 1
	nv := v ^ 1
	nw := w ^ 1
	nt := t ^ 1

	a := (v & w) & (ns | t | nx)
	b := p & (nv | nw | (s & (t ^ 1) & x))
	c := q & (nv | nw | (s & (t ^ 1) & x))
	d := r

	e := v & ((nw & x) | ((t ^ 1) & x) | (s & x))
	f := (s & (nv | nx)) | (p & ns & t & v & w & x)
	g := (t & (nv | nx)) | (q & ns & t & w)
	h := u

	i := v & ((nw & nx) | (w & x & (s | t)))
	j := (nv & w) | (s & v & nw & x) | (p & w & (nx | (ns & (t ^ 1))))
	k := (nv & x) | (t & nw & x) | (q & v & w & (nx | (ns & nt)))
	m := y

	d2 := (a << 3) | (b << 2) | (c << 1) | d
	d1 := (e << 3) | (f << 2) | (g << 1) | h
	d0 := (i << 3) | (j << 2) | (k << 1) | m

	verify("", func() bool { return d2 <= 9 })
	verify("", func() bool { return d1 <= 9 })
	verify("", func() bool { return d0 <= 9 })
	return d2*100 + d1*10 + d0
}

func dpdEncodeDeclet(n int64) int64 {
	verify("", func() bool { return ucmp_int_64x64(n, 999) <= 0 })
	if n == 0 {
		return 0
	}
	q10 := ushr_64_64(n*205, 11) // n / 10
	q100 := ushr_64_64(n*41, 12) // n / 100

	d2 := q100
	d1 := q10 - (q100 * 10)
	d0 := n - (q10 * 10)

	a := ushr_64_64(d2, 3) & 1
	b := ushr_64_64(d2, 2) & 1
	c := ushr_64_64(d2, 1) & 1
	d := d2 & 1

	e := ushr_64_64(d1, 3) & 1
	f := ushr_64_64(d1, 2) & 1
	g := ushr_64_64(d1, 1) & 1
	h := d1 & 1

	i := ushr_64_64(d0, 3) & 1
	j := ushr_64_64(d0, 2) & 1
	k := ushr_64_64(d0, 1) & 1
	m := d0 & 1

	na := a ^ 1
	ne := e ^ 1
	ni := i ^ 1

	p := b | ((a & j) | ((a & f) & i))
	q := c | ((a & k) | ((a & g) & i))
	r := d
	s := (f & (na | ni)) | ((na & (e & j)) | (e & i))
	t := g | ((na & (e & k)) | (a & i))
	u := h
	v := a | (e | i)
	w := a | ((e & i) | (ne & j))
	x := e | ((a & i) | (na & k))
	y := m

	return (p << 9) | (q << 8) | (r << 7) | (s << 6) |
		(t << 5) | (u << 4) | (v << 3) | (w << 2) |
		(x << 1) | y
}

func d128_from_dpd128(dpd u128) d128 { return dpd128Decode(dpd.dw1, dpd.dw0) }
func d128_to_dpd128(x d128) u128     { return dpd128Encode(x) }

// dpd64Decode decodes a d128 from a 64-bit DPD-encoded decimal64 value
// (IEEE 754-2019 §3.5.2). Every decimal64 value is a strict subset of decimal128,
// so this decode is exact and lossless — no rounding, no context, no signalling.
// The coefficient (≤ 16 digits) and NaN payload (≤ 15 digits) each fit one word,
// so unlike dpd128Decode this is entirely single-word. The 50-bit trailing field T
// holds 5 declets; the most-significant digit d0 comes from the combination field
// and is folded into the 6th declet slot (dpdDecodeDeclet(d)==d for a lone digit),
// exactly as dpd128Decode folds in ITS leading digit. DPD declets always decode to
// ≤ 999, so no canonicalization checks are needed (unlike bid64Decode).
func dpd64Decode(dpd64 int64) d128 {
	const bias = 398
	const w = 8   // exponent continuation width
	const w5 = 13 // combination field width
	const t = 50  // trailing significand width

	sign := ushr_64_64(dpd64, 63) != 0
	combination := int(ushr_64_64(dpd64, t)) & 0x1FFF
	coeffT := dpd64 & ((int64(1) << t) - 1) // low 50 bits: 5 declets
	expContinuation := combination & 0xFF

	var d0 int64
	var qExp int
	isNaN := false

	top2 := combination >> (w5 - 2)
	top4 := combination >> (w5 - 4)
	top5 := combination >> (w5 - 5)

	if top2 != 0b11 {
		expMSB := top2
		biasedExponent := (expMSB << w) | expContinuation
		qExp = biasedExponent - bias
		d0 = int64((combination >> w) & 0b111)
	} else if top4 != 0b1111 {
		expMSB := (combination >> (w5 - 4)) & 0b11
		biasedExponent := (expMSB << w) | expContinuation
		qExp = biasedExponent - bias
		d0 = 8 + int64((combination>>w)&0b1)
	} else if top5 == 0b11110 {
		return d128_infinity(sign)
	} else {
		verify("", func() bool { return top5 == 0b11111 })
		isNaN = true
		d0 = 0
		qExp = 0
	}

	// Fold d0 into the 6th declet slot; binFromDecletsUpTo7 folds it in at 10^15.
	coeff := binFromDecletsUpTo7((d0 << 50) | coeffT)

	if isNaN {
		isSignaling := ((combination >> (w5 - 6)) & 1) != 0
		return d128_nan_64x64(sign, isSignaling, coeff, 0)
	}
	if coeff == 0 {
		return d128_zero(sign, qExp)
	}
	return decimalFNZ_64x64(sign, qExp, coeff, 0)
}

func d128_from_dpd64(dpd64 int64) d128 { return dpd64Decode(dpd64) }
