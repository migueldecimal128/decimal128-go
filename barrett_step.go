// Single Barrett-reduction step for a 64-bit dividend chunk (D10). Given dividend
// dw, denominator denom, and precomputed reciprocal mu = floor(2^64 / denom),
// returns (q, r) with dw = q*denom + r and 0 <= r < denom. qHat = (dw*mu) >> 64
// is never above the true quotient and at most 1 below, so a single conditional
// subtract corrects exactly. Requires dw < 2^64 and a contract-satisfying
// mu/denom pair (e.g. a tabulated 5^k denominator).

package decimal128

import "math/bits"

func divRem_q64r64_64x64x64_mu_barrettStep(dw, denom, mu int64) quot64Rem64 {
	qHat := uint64(umulHi_64_64x64(dw, mu))
	d := uint64(denom)
	rHat := uint64(dw) - qHat*d
	rMinusDenom, borrow := bits.Sub64(rHat, d, 0)
	if borrow != 0 { // rHat < denom: estimate already exact
		return quot64Rem64{int64(qHat), int64(rHat)}
	}
	return quot64Rem64{int64(qHat + 1), int64(rMinusDenom)}
}
