// D48: the 48-digit internal extended-precision domain (D48GoWorkOrder.md G-1;
// Swift D48.swift). Same contract as D38: round-half-even, no flags, no
// rounding modes, finite only, no −0, unbounded exponent; 48 digits instead of
// 38, with 96-digit (u320) exact intermediates.

package decimal128

// d48 is the coefficient as three dword limbs (lo first, D23; < 10^48 < 2^160)
// and qExpAndSignBit, the signed quantum exponent over the sign bit, last (the
// small field last, as d38 and Swift's D48).
type d48 struct {
	coeffDw0       int64
	coeffDw1       int64
	coeffDw2       int64
	qExpAndSignBit int32
}

// d48Pack packs sign + qExp + coefficient dword limbs (lo first); Swift's
// init(_:_:_:_:_:).
func d48Pack(signFlag bool, qExp int, coeffDw0, coeffDw1, coeffDw2 int64) d48 {
	return d48{coeffDw0, coeffDw1, coeffDw2, int32((qExp << 1) | signBitOf(signFlag))}
}

// d48Raw is the raw-limb literal constructor: the pre-packed (qExp << 1 | sign)
// field first, then the coefficient dwords.
func d48Raw(qExpAndSignBit int32, coeffDw0, coeffDw1, coeffDw2 int64) d48 {
	return d48{coeffDw0, coeffDw1, coeffDw2, qExpAndSignBit}
}

const D48_DIGITS = 48

var d48_ZERO = d48{0, 0, 0, 0}

// d48_fromSignQExp_160 is from an explicit sign, qExp, and a coefficient below
// 10^48 (< 2^160, held as a u192).
func d48_fromSignQExp_160(sign bool, qExp int, coeff u192) d48 {
	return d48Pack(sign, qExp, coeff.dw0, coeff.dw1, coeff.dw2)
}

// d48_fromSignQExp_64x64x64 is from an explicit sign, qExp, and raw coefficient
// dwords (lo first).
func d48_fromSignQExp_64x64x64(sign bool, qExp int, dw0, dw1, dw2 int64) d48 {
	return d48Pack(sign, qExp, dw0, dw1, dw2)
}

// d48_fromQExpAndSignBit_160 is from a packed qExpAndSignBit field and a
// coefficient below 10^48.
func d48_fromQExpAndSignBit_160(qExpAndSignBit int32, coeff u192) d48 {
	return d48{coeff.dw0, coeff.dw1, coeff.dw2, qExpAndSignBit}
}

func d48_qExp(x d48) int { return int(x.qExpAndSignBit >> 1) }

func d48_signBit(x d48) int { return int(x.qExpAndSignBit & 1) }

func d48_signFlag(x d48) bool { return d48_signBit(x) != 0 }

func d48_isZero(x d48) bool { return (x.coeffDw0 | x.coeffDw1 | x.coeffDw2) == 0 }

func d48_negated(x d48) d48 { return d48{x.coeffDw0, x.coeffDw1, x.coeffDw2, x.qExpAndSignBit ^ 1} }

// d48_u160 / d48_u320 are the coefficient as a u192 / u320 value.
func d48_u160(x d48) u192 { return u192{x.coeffDw0, x.coeffDw1, x.coeffDw2} }

func d48_u320(x d48) u320 { return u320{x.coeffDw0, x.coeffDw1, x.coeffDw2, 0, 0} }

func d48_digitLen(x d48) int { return digitLen_320(d48_u320(x)) }

func d48_sciExp(x d48) int { return d48_qExp(x) + d48_digitLen(x) - 1 }

func d48_multiply(x, y d48) d48 {
	zSign := d48_signFlag(x) != d48_signFlag(y)
	zExp := d48_qExp(x) + d48_qExp(y)
	p := mul_320_192x192(d48_u160(x), d48_u160(y))
	return d48_finalize_320(zSign, zExp, p, RESIDUE_EXACT)
}

func d48_square(x d48) d48 {
	zExp := d48_qExp(x) + d48_qExp(x)
	p := mul_320_192x192(d48_u160(x), d48_u160(x))
	return d48_finalize_320(false, zExp, p, RESIDUE_EXACT)
}

// d48_fromInt32 is n as a D48 (|n| taken as unsigned, so the minimum is safe).
func d48_fromInt32(n int) d48 {
	mag := uint64(n)
	if n < 0 {
		mag = -mag
	}
	return d48Pack(n < 0, 0, int64(mag), 0, 0)
}
