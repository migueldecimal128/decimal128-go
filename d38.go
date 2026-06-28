// The D38 extended-precision intermediate: a 128-bit coefficient with a packed
// (qExp << 1 | sign) field, used to carry 38-digit intermediates before
// finalization. Field order matches the C aggregate literal (coeff limbs first,
// qExpAndSignBit last — §4.8). The arithmetic forms (multiply/square/format) that
// need the finalize/print engine land in later increments.

package decimal128

type d38 struct {
	coeffLo64      int64
	coeffHi64      int64
	qExpAndSignBit int32
}

// d38Pack packs sign + qExp + coefficient limbs (the init(sign,qExp,lo,hi) mechanic).
func d38Pack(signFlag bool, qExp int, coeffLo64, coeffHi64 int64) d38 {
	return d38{coeffLo64, coeffHi64, int32((qExp << 1) | signBitOf(signFlag))}
}

// d38Raw is the raw-limb literal constructor (pre-packed qExpAndSignBit).
func d38Raw(qExpAndSignBit int32, coeffLo64, coeffHi64 int64) d38 {
	return d38{coeffLo64, coeffHi64, qExpAndSignBit}
}

var D38_ZERO = d38{0, 0, 0}

func d38_qExp(x d38) int { return int(x.qExpAndSignBit >> 1) }

func d38_signBit(x d38) int { return int(x.qExpAndSignBit & 1) }

func d38_signFlag(x d38) bool { return d38_signBit(x) != 0 }

func d38_isZero(x d38) bool { return isZero_bool_128(x.coeffLo64, x.coeffHi64) }

func d38_negated(x d38) d38 { return d38{x.coeffLo64, x.coeffHi64, x.qExpAndSignBit ^ 1} }

func d38_digitLen(x d38) int { return digitLen_int_128(x.coeffLo64, x.coeffHi64) }

func d38_u128(x d38) u128 { return fromLoHi_128_64x64(x.coeffLo64, x.coeffHi64) }

func d38_multiply(x, y d38) d38 {
	zSign := d38_signFlag(x) != d38_signFlag(y)
	zExp := d38_qExp(x) + d38_qExp(y)
	prod := mul_256_128x128(x.coeffLo64, x.coeffHi64, y.coeffLo64, y.coeffHi64)
	return finalize38_256(zSign, zExp, prod, RESIDUE_EXACT)
}

func d38_square(x d38) d38 {
	zExp := d38_qExp(x) + d38_qExp(x)
	prod := mul_256_128x128(x.coeffLo64, x.coeffHi64, x.coeffLo64, x.coeffHi64)
	return finalize38_256(false, zExp, prod, RESIDUE_EXACT)
}

func d38_format(x d38) string {
	return toFiniteString(d38_signFlag(x), d38_qExp(x), d38_u128(x), FORMAT_AUTO)
}
