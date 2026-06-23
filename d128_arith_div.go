// Divide. The general body dispatches the finite/zero/infinity/NaN cases; the
// FNZ÷FNZ path scales the numerator to 34 extra digits, runs the core 256÷128
// divide with residue, strips exact trailing zeros toward the preferred
// exponent, then finalizes. (divideInteger and remainder land later — they need
// roundToIntegral and fma.)

package decimal128

func d128_divide_tte(x, y d128) d128            { return d128_divide_rnd_ctxnull(x, y, TIES_TO_EVEN, nil) }
func d128_divide_rnd(x, y d128, rnd round) d128 { return d128_divide_rnd_ctxnull(x, y, rnd, nil) }
func d128_divide_ctx(x, y d128, ctx *context) d128 {
	return d128_divide_rnd_ctxnull(x, y, ctx.round, ctx)
}

func d128_divide_rnd_ctxnull(x, y d128, rounding round, ctxnull *context) d128 {
	resultSign := d128_signFlag(x) != d128_signFlag(y)
	if bothFinite(x, y) {
		if d128_isNotZER(x) && d128_isNotZER(y) {
			return divFnzFnz_rnd_ctx(x, y, resultSign, rounding, ctxnull)
		}
		if d128_isZER(x) && d128_isZER(y) {
			return ctxSignalInvalidOperation(ctxnull, causeDivZeroByZero)
		}
		if d128_isZER(x) {
			return d128_zero(resultSign, d128_qExp(x)-d128_qExp(y))
		}
		return ctxSignalDivByZero(ctxnull, resultSign)
	}
	if d128_isNAN(x) || d128_isNAN(y) {
		return ctxNanOperand2(ctxnull, x, y)
	}
	verify("", d128_isINF(x) || d128_isINF(y))
	if d128_isINF(x) && d128_isINF(y) {
		return ctxSignalInvalidOperation(ctxnull, causeDivInfByInf)
	}
	if d128_isINF(y) {
		return d128_zero(resultSign, -6176)
	}
	return d128_infinity(resultSign)
}

func divFnzFnz_rnd_ctx(x, y d128, resultSign bool, rounding round, ctxnull *context) d128 {
	digitLenDelta := d128_digitLen(x) - d128_digitLen(y)
	numeratorScale := 34 - digitLenDelta
	verify("", numeratorScale > 0)

	scaledNumerator := mulPow10_256_128(x.ubdLo64, d128_hi49(x), numeratorScale)
	qr := u256DivResidue_128(scaledNumerator, d128_u128(y))
	coefficient := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
	res := qr.residue

	qPreferred := d128_qExp(x) - d128_qExp(y)
	qZ := qPreferred - numeratorScale

	if res == RESIDUE_EXACT && qZ < qPreferred {
		ss := strip_s128sc_128(coefficient.dw0, coefficient.dw1, qPreferred-qZ)
		coefficient = u128{ss.strippedDw0, ss.strippedDw1}
		qZ += int(ss.stripCount)
	}
	return finalize_128_ctx(resultSign, qZ, coefficient, res, rounding, ctxnull)
}
