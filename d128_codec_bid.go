// BID (Binary Integer Decimal) encode/decode for decimal128 (IEEE 754-2019
// §3.5.2). The coefficient is a plain binary integer in the trailing significand
// T, with the top 3 coefficient bits in the G combination field (no declets).
//
// 128-bit layout: bit 127 sign; bits 126..110 the 17-bit G field; bits 109..0
// the 110-bit trailing significand T.

package decimal128

const (
	qtinyNeg6176 = -6176
	qmax6111     = 6111

	coeffMaxHi   int64 = 0x0001_ED09_BEAD_87C0 // 10^34 - 1, hi half
	coeffMaxLo   int64 = 0x378D_8E63_FFFF_FFFF // 10^34 - 1, lo half
	payloadMaxHi int64 = 0x0000_314D_C644_8D93 // 10^33 - 1, hi half
	payloadMaxLo int64 = 0x38C1_5B09_FFFF_FFFF // 10^33 - 1, lo half

	coeff64Max   int64 = 9_999_999_999_999_999 // 10^16 - 1, decimal64 coeff max (one word)
	payload64Max int64 = 999_999_999_999_999   // 10^15 - 1, decimal64 NaN payload max
)

// bid128Encode encodes a d128 into its 128-bit BID representation.
func bid128Encode(d d128) u128 {
	var coeff u128
	if d128_isFIN(d) {
		coeff = d128_u128(d)
	} else if d128_isNAN(d) {
		coeff = d128_nanPayload(d)
	} else {
		verify("", func() bool { return d128_isINF(d) })
		coeff = u128{0, 0}
	}

	coeffHi64 := coeff.dw1
	mostSigBits3 := int(ushr_64_64(coeffHi64, 110-64)) & 0x07

	signCombo := encodeSignAndGCombinationFieldBid128(d, mostSigBits3)

	significandHi46 := coeffHi64 & ((int64(1) << (110 - 64)) - 1)
	bidHi := signCombo | significandHi46
	bidLo := coeff.dw0
	return fromLoHi_128_64x64(bidLo, bidHi)
}

func encodeSignAndGCombinationFieldBid128(d d128, mostSigBits3 int) int64 {
	signBitLong := int64(d128_signBit(d)) << 63
	var gCombinationField int

	if d128_isFIN(d) {
		qExp := d128_qExp(d)
		verify("", func() bool { return qExp >= qtinyNeg6176 && qExp <= qmax6111 })
		biasedQExp := qExp - qtinyNeg6176
		verify("", func() bool { return (biasedQExp & 0x3000) != 0x3000 })
		gCombinationField = (biasedQExp << 3) | mostSigBits3
	} else if d128_isINF(d) {
		gCombinationField = 0b11110 << 12
	} else {
		verify("", func() bool { return d128_isNAN(d) })
		if d128_isSNAN(d) {
			gCombinationField = 0b111111 << 11
		} else {
			gCombinationField = 0b111110 << 11
		}
	}
	return signBitLong | (int64(gCombinationField) << 46)
}

// bid128Decode decodes a d128 from its 128-bit BID encoding.
func bid128Decode(bidHi, bidLo int64) d128 {
	const bias = 6176
	const w5 = 17 // combination field width
	const t = 110 // trailing significand width

	sign := ushr_64_64(bidHi, 63) != 0
	combination := int(ushr_64_64(bidHi, t-64)) & 0x1FFFF
	coeffTHi := bidHi & ((int64(1) << (t - 64)) - 1)

	top2 := combination >> (w5 - 2)
	top4 := combination >> (w5 - 4)
	top5 := combination >> (w5 - 5)

	var coeffHiPrefix int64
	var qExp int

	if top2 != 0b11 {
		// Case A: biased exponent in top 14 bits, coeff prefix in low 3 bits.
		biasedExponentE := combination >> 3
		qExp = biasedExponentE - bias
		coeffHiPrefix = int64(combination & 0x07)
	} else if top4 != 0b1111 {
		// Case B: biased exponent in middle 14 bits, coeff prefix = 8 + low bit.
		biasedExponentE := (combination >> 1) & ((1 << 14) - 1)
		qExp = biasedExponentE - bias
		coeffHiPrefix = 8 + int64(combination&1)
	} else if top5 == 0b11110 {
		return d128_infinity(sign)
	} else {
		verify("", func() bool { return top5 == 0b11111 })
		isSignaling := ((combination >> (w5 - 6)) & 1) != 0
		var payloadCanonLo, payloadCanonHi int64
		if ucmp_int_64x64(coeffTHi, payloadMaxHi) > 0 ||
			(coeffTHi == payloadMaxHi && ucmp_int_64x64(bidLo, payloadMaxLo) > 0) {
			payloadCanonLo = 0
			payloadCanonHi = 0
		} else {
			payloadCanonLo = bidLo
			payloadCanonHi = coeffTHi
		}
		return d128_nan_64x64(sign, isSignaling, payloadCanonLo, payloadCanonHi)
	}

	// Finite: assemble the 113-bit coefficient from the prefix and T.
	dw1Bits := (coeffHiPrefix << (t - 64)) | coeffTHi
	dw0Bits := bidLo

	// Non-canonical finite coefficient (> 10^34 - 1) is treated as zero.
	if isZero_bool_128(dw0Bits, dw1Bits) ||
		ucmp_int_64x64(dw1Bits, coeffMaxHi) > 0 ||
		(dw1Bits == coeffMaxHi && ucmp_int_64x64(dw0Bits, coeffMaxLo) > 0) {
		return d128_zero(sign, qExp)
	}
	return decimalFNZ_64x64(sign, qExp, dw0Bits, dw1Bits)
}

func d128_from_bid128(bid u128) d128 { return bid128Decode(bid.dw1, bid.dw0) }
func d128_to_bid128(x d128) u128     { return bid128Encode(x) }

// bid64Decode decodes a d128 from a 64-bit BID-encoded decimal64 value
// (IEEE 754-2019 §3.5.2). Every decimal64 value is a strict subset of decimal128,
// so this decode is exact and lossless — no rounding, no context, no signalling.
// The coefficient (≤ 16 digits, < 2^54) and NaN payload (≤ 15 digits) each fit
// one word, so unlike bid128Decode this is entirely single-word. Non-canonical
// finite coefficients (> 10^16 - 1) are treated as zero; non-canonical NaN
// payloads (≥ 10^15) are canonicalized to zero.
func bid64Decode(bid64 int64) d128 {
	const bias = 398
	const w5 = 13 // combination field width
	const t = 50  // trailing significand width

	sign := ushr_64_64(bid64, 63) != 0
	combination := int(ushr_64_64(bid64, t)) & 0x1FFF
	coeffT := bid64 & ((int64(1) << t) - 1) // lower 50 bits of T

	top2 := combination >> (w5 - 2)
	top4 := combination >> (w5 - 4)
	top5 := combination >> (w5 - 5)

	var coeffPrefix int64
	var qExp int

	if top2 != 0b11 {
		// Case A: biased exponent in top 10 bits, coeff prefix in low 3 bits.
		biasedExponentE := combination >> 3
		qExp = biasedExponentE - bias
		coeffPrefix = int64(combination & 0x07)
	} else if top4 != 0b1111 {
		// Case B: biased exponent in middle 10 bits, coeff prefix = 8 + low bit.
		biasedExponentE := (combination >> 1) & ((1 << 10) - 1)
		qExp = biasedExponentE - bias
		coeffPrefix = 8 + int64(combination&1)
	} else if top5 == 0b11110 {
		return d128_infinity(sign)
	} else {
		verify("", func() bool { return top5 == 0b11111 })
		isSignaling := ((combination >> (w5 - 6)) & 1) != 0
		var payloadCanon int64
		if ucmp_int_64x64(coeffT, payload64Max) > 0 {
			payloadCanon = 0
		} else {
			payloadCanon = coeffT
		}
		return d128_nan_64x64(sign, isSignaling, payloadCanon, 0)
	}

	// Finite: coefficient = prefix (bits 53..50) | T; fits one word, upper word 0.
	coeff := (coeffPrefix << t) | coeffT

	// Non-canonical finite coefficient (> 10^16 - 1) is treated as zero.
	if coeff == 0 || ucmp_int_64x64(coeff, coeff64Max) > 0 {
		return d128_zero(sign, qExp)
	}
	return decimalFNZ_64x64(sign, qExp, coeff, 0)
}

func d128_from_bid64(bid64 int64) d128 { return bid64Decode(bid64) }
