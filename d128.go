// The D128 value type — the library's internal UBD representation (UBD.md §3.2),
// stored as two int64 dword limbs. Layout: top 14 bits qExp, next 1 bit sign,
// low 49 bits the coefficient high half (ubdLo64 holds the low 64). Non-finite
// values use special qExp ranges with coefficient >= 10^34. coeff == 0 implies a
// zero cohort member.
//
// In Go the public Decimal128 (Phase 5) wraps this unexported d128.

package decimal128

type d128 struct {
	ubdLo64 int64
	ubdHi64 int64
}

// d128Pack builds a finite d128 from sign + quantum exponent + coefficient dwords
// (the init(sign,qExp,coeff) mechanic; construction is platform-local).
func d128Pack(sign bool, qExp int, coeffDw0, coeffDw1 int64) d128 {
	verify("", qExp >= -6176 && qExp <= 6111)
	verify("", isLt_pow10_34_bool_128(coeffDw0, coeffDw1))
	qExpAndSignBit := (int64(qExp) << 50) | (boolToInt64(sign) << 49)
	return d128{ubdLo64: coeffDw0, ubdHi64: qExpAndSignBit | coeffDw1}
}

// d128FromQExpSignBit builds a finite d128 from a pre-packed qExpAndSignBit and
// coefficient (the init(qExpAndSignBit, coeff) mechanic).
func d128FromQExpSignBit(qExpAndSignBit, coeffDw0, coeffDw1 int64) d128 {
	verify("", ushr_64_64(qExpAndSignBit, 15) == 0)
	verify("", isLt_pow10_34_bool_128(coeffDw0, coeffDw1))
	return d128{ubdLo64: coeffDw0, ubdHi64: (qExpAndSignBit << 49) | coeffDw1}
}

// --- field accessors ---

func d128_hi49(x d128) int64 { return x.ubdHi64 & ((1 << 49) - 1) }

func d128_signBit(x d128) int { return int(ushr_64_64(x.ubdHi64, 49)) & 1 }

func d128_signFlag(x d128) bool { return d128_signBit(x) != 0 }

func d128_signMask(x d128) int { return int((x.ubdHi64 << 14) >> 63) }

func d128_qExp(x d128) int { return int(x.ubdHi64 >> 50) }

func d128_qExpAndSignBit(x d128) int64 { return ushr_64_64(x.ubdHi64, 49) }

func d128_u128(x d128) u128 { return fromLoHi_128_64x64(x.ubdLo64, d128_hi49(x)) }

// --- lengths and scientific exponent ---

func d128_bitLen(x d128) int { return bitLen_int_128(x.ubdLo64, d128_hi49(x)) }

func d128_digitLen(x d128) int { return digitLen_int_128(x.ubdLo64, d128_hi49(x)) }

func d128_sciExpWithDigitLen(x d128, digitLen int) int {
	bitLen := bitLen_int_128(x.ubdLo64, d128_hi49(x))
	digitLen = digitLen_int_128_withBitLen(x.ubdLo64, d128_hi49(x), bitLen)
	return d128_qExp(x) + digitLen + (-bitLen >> 63)
}

func d128_sciExp(x d128) int {
	digitLen := d128_digitLen(x)
	return d128_qExp(x) + digitLen + (-digitLen >> 63)
}

// --- predicates ---

func d128_isNegative(x d128) bool { return d128_signFlag(x) }

func d128_isNonFinite(x d128) bool { return ushr_64_64(x.ubdHi64, 60) == 7 }

func d128_isFinite(x d128) bool { return ushr_64_64(x.ubdHi64, 60) != 7 }

func d128_isFIN(x d128) bool { return ushr_64_64(x.ubdHi64, 60) != 7 }

func d128_isZER(x d128) bool { return ((x.ubdHi64 << 15) | x.ubdLo64) == 0 }

func d128_isNotZER(x d128) bool { return ((x.ubdHi64 << 15) | x.ubdLo64) != 0 }

func d128_isFNZ(x d128) bool { return d128_isFinite(x) && !d128_isZER(x) }

func d128_isINF(x d128) bool { return ushr_64_64(x.ubdHi64, 59) == 14 }

func d128_isNAN(x d128) bool { return ushr_64_64(x.ubdHi64, 59) == 15 }

func d128_isNotNAN(x d128) bool { return ushr_64_64(x.ubdHi64, 59) != 15 }

func d128_isQNAN(x d128) bool { return ushr_64_64(x.ubdHi64, 58) == 30 }

func d128_isSNAN(x d128) bool { return ushr_64_64(x.ubdHi64, 58) == 31 }

func d128_isNotSNAN(x d128) bool { return ushr_64_64(x.ubdHi64, 58) != 31 }

func d128_isSubnormal(x d128) bool {
	return d128_isFNZ(x) && d128_sciExpWithDigitLen(x, d128_digitLen(x)) < -6143
}

func d128_isNormal(x d128) bool {
	return d128_isFNZ(x) && d128_sciExpWithDigitLen(x, d128_digitLen(x)) >= -6143
}

func d128_isCanonical(x d128) bool { return true }

func bothFinite(x, y d128) bool { return d128_isFIN(x) && d128_isFIN(y) }

// --- IEEE 754-2019 §5.7.2 class codes (matching Intel bid128_class 0-9) ---

const (
	CLASS_SIGNALING_NAN      = 0
	CLASS_QUIET_NAN          = 1
	CLASS_NEGATIVE_INFINITY  = 2
	CLASS_NEGATIVE_NORMAL    = 3
	CLASS_NEGATIVE_SUBNORMAL = 4
	CLASS_NEGATIVE_ZERO      = 5
	CLASS_POSITIVE_ZERO      = 6
	CLASS_POSITIVE_SUBNORMAL = 7
	CLASS_POSITIVE_NORMAL    = 8
	CLASS_POSITIVE_INFINITY  = 9
)

func d128_ieeeClass(x d128) int {
	if d128_isSNAN(x) {
		return CLASS_SIGNALING_NAN
	}
	if d128_isQNAN(x) {
		return CLASS_QUIET_NAN
	}
	if d128_isINF(x) {
		if d128_isNegative(x) {
			return CLASS_NEGATIVE_INFINITY
		}
		return CLASS_POSITIVE_INFINITY
	}
	if d128_isZER(x) {
		if d128_isNegative(x) {
			return CLASS_NEGATIVE_ZERO
		}
		return CLASS_POSITIVE_ZERO
	}
	if d128_isSubnormal(x) {
		if d128_isNegative(x) {
			return CLASS_NEGATIVE_SUBNORMAL
		}
		return CLASS_POSITIVE_SUBNORMAL
	}
	if d128_isNegative(x) {
		return CLASS_NEGATIVE_NORMAL
	}
	return CLASS_POSITIVE_NORMAL
}

// --- non-finite factories ---

const (
	d128NonFiniteBias int64 = 0x1_F000_0000_0000
	d128InfHi64       int64 = 14 << 59
	d128NanHi64       int64 = 15 << 59
)

func d128_infinity(sign bool) d128 {
	signBit := boolToInt64(sign) << 49
	return d128{ubdLo64: 0, ubdHi64: d128InfHi64 | signBit | d128NonFiniteBias}
}

func d128_nan_64x64(sign, signaling bool, payloadDw0, payloadDw1 int64) d128 {
	var payload110Lo, payload110Hi int64
	if cmp_int_128x128_pow10(payloadDw0, payloadDw1, 33) >= 0 {
		payload110Lo = 0
		payload110Hi = 0
	} else {
		payload110Lo = payloadDw0
		payload110Hi = payloadDw1
	}
	payloadLo64 := payload110Lo
	payloadHi46 := payload110Hi
	payloadHi2 := ushr_64_64(payloadHi46, 44)
	payloadMid44 := payloadHi46 & ((1 << 44) - 1)
	payloadMid44Biased := d128NonFiniteBias + payloadMid44
	verify("", payloadMid44Biased < (1<<49))
	signBit := boolToInt64(sign) << 49
	payloadQBits := payloadHi2 << 50
	snanBit := boolToInt64(signaling) << 58
	ubdHi := d128NanHi64 | snanBit | payloadQBits | signBit | payloadMid44Biased
	return d128{ubdLo64: payloadLo64, ubdHi64: ubdHi}
}

var (
	D128_QNAN0        = d128{ubdLo64: 0, ubdHi64: d128NanHi64 | d128NonFiniteBias}
	D128_SNAN0        = d128{ubdLo64: 0, ubdHi64: d128NanHi64 | (1 << 58) | d128NonFiniteBias}
	D128_POS_INFINITY = d128_infinity(false)
	D128_NEG_INFINITY = d128_infinity(true)
)

func d128_nanPayload(x d128) u128 {
	if !d128_isNAN(x) {
		return u128{0, 0}
	}
	ubdHi := x.ubdHi64
	payloadMid44Biased := ubdHi & ((1 << 49) - 1)
	verify("", payloadMid44Biased >= d128NonFiniteBias)
	payloadMid44 := payloadMid44Biased - d128NonFiniteBias
	payloadHi2 := ushr_64_64(ubdHi, 50) & 0x03
	payloadHi46 := (payloadHi2 << 44) | payloadMid44
	return fromLoHi_128_64x64(x.ubdLo64, payloadHi46)
}

func d128_quieted(x d128) d128 {
	verify("", d128_isNAN(x))
	return d128{ubdLo64: x.ubdLo64, ubdHi64: x.ubdHi64 &^ (int64(1) << 58)}
}

// --- finite factories ---

func d128_zero(sign bool, qExp int) d128 {
	qClamped := min(max(qExp, -6176), 6111)
	return d128Pack(sign, qClamped, 0, 0)
}

func decimalFNZ_128(sign bool, qExp int, coeffDw0, coeffDw1 int64) d128 {
	verify("", qExp >= -6176 && qExp <= 6111)
	verify("", !isZero_bool_128(coeffDw0, coeffDw1))
	return d128Pack(sign, qExp, coeffDw0, coeffDw1)
}

func decimalFNZ_64x64(sign bool, qExp int, dw0, dw1 int64) d128 {
	verify("", qExp >= -6176 && qExp <= 6111)
	verify("", !isZero_bool_128(dw0, dw1))
	verify("", isLt_pow10_34_bool_128(dw0, dw1))
	qExpAndSignBit := (int64(qExp) << 50) | (boolToInt64(sign) << 49)
	return d128{ubdLo64: dw0, ubdHi64: qExpAndSignBit | dw1}
}

func d128_bitwiseEQ(x, y d128) bool {
	return x.ubdLo64 == y.ubdLo64 && x.ubdHi64 == y.ubdHi64
}

func d128_fromInt(n int) d128 {
	mask := int64(n) >> 63
	abs := (int64(n) ^ mask) - mask
	signBit := mask & (1 << 49)
	return d128{ubdLo64: abs, ubdHi64: signBit}
}

func d128_fromUInt(u uint) d128 {
	return d128{ubdLo64: int64(u), ubdHi64: 0}
}
