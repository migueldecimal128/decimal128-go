// IEEE 754-2019 §9.2 rootn(x, n) = x^(1/n) for integer n ≠ 0. When |n| is a
// power of two, take √ repeatedly (then reciprocal for n < 0); otherwise
// |x|^(1/n) via pow, restoring the sign for odd n.

package decimal128

import "math/bits"

var d128RootnOne = d128_fromInt64(1)

func d128_rootn_tte(x d128, n int) d128            { return d128_rootn_rnd_ctxnull(x, n, ROUND_TIES_TO_EVEN, nil) }
func d128_rootn_rnd(x d128, n int, rnd round) d128 { return d128_rootn_rnd_ctxnull(x, n, rnd, nil) }
func d128_rootn_ctx(x d128, n int, ctx *context) d128 {
	return d128_rootn_rnd_ctxnull(x, n, ctx.round, ctx)
}

func d128_rootn_rnd_ctxnull(x d128, n int, rounding round, ctxnull *context) d128 {
	if n == 0 {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeRootnBadArgs)
	}
	xSign := d128_signFlag(x)
	nIsOdd := (n & 1) != 0

	if d128_isFIN(x) {
		if d128_isZER(x) {
			// rootn(±0, n): n < 0 → ±∞ (divideByZero); n > 0 → ±0. Sign carries
			// only for odd n.
			resultSign := nIsOdd && xSign
			if n < 0 {
				return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
			}
			return d128_zero(resultSign, 0)
		}
		// Finite non-zero.
		if xSign && !nIsOdd {
			return ctxSignalInvalidOperation_ctxnull(ctxnull, causeRootnBadArgs) // even root of a negative
		}
		return rootnFNZ(x, n, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	// x = ±∞.
	if xSign && nIsOdd {
		if n < 0 {
			return d128_zero(true, 0)
		}
		return D128_NEG_INFINITY // -∞, odd n
	}
	if xSign {
		return ctxSignalInvalidOperation_ctxnull(ctxnull, causeRootnBadArgs) // -∞, even n
	}
	if n < 0 {
		return d128_zero(false, 0)
	}
	return D128_POS_INFINITY // +∞
}

func rootnFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	// |n| as an unsigned magnitude (handles n == math.MinInt32 without overflow).
	var nAbs uint64
	if n < 0 {
		nAbs = -uint64(int64(n))
	} else {
		nAbs = uint64(int64(n))
	}

	if bits.OnesCount64(nAbs) == 1 {
		// |n| is a power of two: x^(1/n) = √ applied `shifts` times. Intermediate
		// roots are quiet/ties-to-even; the last honors the caller's rounding+flags.
		shifts := bits.TrailingZeros64(nAbs)
		z := x
		for i := 0; i < shifts; i++ {
			if i == shifts-1 {
				z = d128_sqrt_rnd_ctxnull(z, rounding, ctxnull)
			} else {
				z = d128_sqrt_tte(z)
			}
		}
		if n < 0 {
			z = d128_divide_rnd_ctxnull(d128RootnOne, z, rounding, ctxnull)
		}
		return z
	}

	// General case: |x|^(1/n), with the sign restored for odd n.
	absX := d128_abs(x)
	nReciprocal := d128_divide_tte(d128RootnOne, d128_fromInt64(n))
	z := d128_pow_rnd_ctxnull(absX, nReciprocal, rounding, ctxnull)
	if d128_signFlag(x) {
		return d128_negate(z)
	}
	return z
}
