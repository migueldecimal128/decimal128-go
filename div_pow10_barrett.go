// The 128- and 256-bit exact-remainder Barrett pow10 unit (D20). Divides a u128
// or u256 by 10^k (k in 1..barrettPow10Max) returning the full quotient and the
// exact 64-bit remainder. 10^k = 2^k · 5^k: the 2^k part is a shift, the 5^k part
// is Barrett reduction over the tabulated reciprocals. The residue-classifying
// entries (the rrmp10 shape) wrap the exact-remainder forms.

package decimal128

// barrettPow10Max is the largest pow10/pow5 covered by the mu table. (Swift
// BARRETT_POW10_MAX; lowercased to stay unexported.)
const barrettPow10Max = 13

// barrettPow5Mu[i] = floor(2^64 / 5^i) for i in 1..13. Index 0 is unused.
var barrettPow5Mu = [14]uint64{
	0,
	3_689_348_814_741_910_323,
	737_869_762_948_382_064,
	147_573_952_589_676_412,
	29_514_790_517_935_282,
	5_902_958_103_587_056,
	1_180_591_620_717_411,
	236_118_324_143_482,
	47_223_664_828_696,
	9_444_732_965_739,
	1_888_946_593_147,
	377_789_318_629,
	75_557_863_725,
	15_111_572_745,
}

// --- 128-bit entry ---

// divPow10_q128r64_128_barrett computes x / 10^pow10 with exact remainder.
func divPow10_q128r64_128_barrett(xDw0, xDw1 int64, pow10 int) quot128Rem64 {
	verify("", pow10 >= 1)
	verify("", pow10 <= barrettPow10Max)
	loBits := uint64(xDw0) & ((uint64(1) << uint(pow10)) - 1)
	xShifted := shr_128_128(xDw0, xDw1, pow10)
	qr := barrett_128_pow5_q128r64(xShifted.dw0, xShifted.dw1, pow10)
	remainder5 := uint64(qr.remDw0)
	remainder10 := (remainder5 << uint(pow10)) + loBits
	return quot128Rem64{qr.quotDw0, qr.quotDw1, int64(remainder10)}
}

func barrett_128_pow5_q128r64(xDw0, xDw1 int64, pow5 int) quot128Rem64 {
	verify("", pow5 >= 1 && pow5 <= barrettPow10Max)
	denom := pow5_64(pow5)
	mu := int64(barrettPow5Mu[pow5])
	return barrett_128_32_mu_q128r64(xDw0, xDw1, denom, mu)
}

// barrett_128_32_mu_q128r64 is the 32-bit chunk kernel (shared with the 256-bit
// side).
func barrett_128_32_mu_q128r64(xDw0, xDw1, denom, mu int64) quot128Rem64 {
	dwA := lo32_64(xDw0)
	dwB := hi32_64(xDw0)
	dwG := xDw1

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	rG := qrG.remDw0
	qrB := divRem_q64r64_64x64x64_mu_barrettStep((rG<<32)|dwB, denom, mu)
	rB := qrB.remDw0
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((rB<<32)|dwA, denom, mu)
	rA := qrA.remDw0

	q1 := qrG.quotDw0
	q0 := (qrB.quotDw0 << 32) | qrA.quotDw0
	return quot128Rem64{q0, q1, rA}
}

// --- residue-classifying entries (D42) ---

func divPow10_q128res_128_barrett(xDw0, xDw1 int64, pow10 int) quot128Residue {
	qr := divPow10_q128r64_128_barrett(xDw0, xDw1, pow10)
	res := residueFromRemainderPow10_64(qr.remDw0, pow10)
	return quot128Residue{qr.quotDw0, qr.quotDw1, res}
}

func divPow10_q128res_256_barrett(xDw0, xDw1, xDw2, xDw3 int64, pow10 int) quot128Residue {
	qr := divPow10_q256r64_256_barrett(xDw0, xDw1, xDw2, xDw3, pow10)
	if !isZero_bool_128(qr.quotDw2, qr.quotDw3) {
		impossible("divPow10_q128res_256_barrett: quotient overflows U128")
	}
	res := residueFromRemainderPow10_64(qr.remDw0, pow10)
	return quot128Residue{qr.quotDw0, qr.quotDw1, res}
}

func divPow10_q256res_256_barrett(xDw0, xDw1, xDw2, xDw3 int64, pow10 int) quot256Residue {
	qr := divPow10_q256r64_256_barrett(xDw0, xDw1, xDw2, xDw3, pow10)
	res := residueFromRemainderPow10_64(qr.remDw0, pow10)
	return quot256Residue{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3, res}
}

// --- 256-bit entry, width dispatch, and chunk ladder ---

func divPow10_q256r64_256_barrett(xDw0, xDw1, xDw2, xDw3 int64, pow10 int) quot256Rem64 {
	verify("", pow10 >= 1 && pow10 <= barrettPow10Max)
	loBits := xDw0 & ((int64(1) << uint(pow10)) - 1)
	xShifted := shr_256_256(xDw0, xDw1, xDw2, xDw3, pow10)
	qr := barrett_256_pow5_q256r64(xShifted, pow10)
	remainder10 := (qr.remDw0 << uint(pow10)) + loBits
	return quot256Rem64{qr.quotDw0, qr.quotDw1, qr.quotDw2, qr.quotDw3, remainder10}
}

func barrett_256_pow5_q256r64(x u256, pow5 int) quot256Rem64 {
	verify("", pow5 >= 1 && pow5 <= barrettPow10Max)
	denom := pow5_64(pow5)
	mu := int64(barrettPow5Mu[pow5])
	xBitLen := bitLen_int_256(x.dw0, x.dw1, x.dw2, x.dw3)

	if xBitLen <= 64 {
		return barrett_64_64_mu_q256r64(x.dw0, denom, mu)
	}
	if pow5 <= 4 {
		// 5^4 = 625 leaves headroom in 64 bits for the 54-bit (rem<<54)|chunk.
		if xBitLen <= 118 {
			return barrett_118_54_mu_q256r64(x, denom, mu)
		}
		if xBitLen <= 172 {
			return barrett_172_54_mu_q256r64(x, denom, mu)
		}
		if xBitLen <= 226 {
			return barrett_226_54_mu_q256r64(x, denom, mu)
		}
		return barrett_256_54_mu_q256r64(x, denom, mu)
	}
	// pow5 >= 5: denom needs more bits, so shrink to 32-bit chunks.
	if xBitLen <= 128 {
		return barrett_128_32_mu_q256r64(x.dw0, x.dw1, denom, mu)
	}
	if xBitLen <= 192 {
		return barrett_192_32_mu_q256r64(x, denom, mu)
	}
	return barrett_256_32_mu_q256r64(x, denom, mu)
}

func barrett_64_64_mu_q256r64(dw0, denom, mu int64) quot256Rem64 {
	qr0 := divRem_q64r64_64x64x64_mu_barrettStep(dw0, denom, mu)
	return quot256Rem64{qr0.quotDw0, 0, 0, 0, qr0.remDw0}
}

func barrett_128_32_mu_q256r64(xDw0, xDw1, denom, mu int64) quot256Rem64 {
	qr := barrett_128_32_mu_q128r64(xDw0, xDw1, denom, mu)
	return quot256Rem64{qr.quotDw0, qr.quotDw1, 0, 0, qr.remDw0}
}

func barrett_192_32_mu_q256r64(x u256, denom, mu int64) quot256Rem64 {
	dwA := lo32_64(x.dw0)
	dwB := hi32_64(x.dw0)
	dwC := lo32_64(x.dw1)
	dwD := hi32_64(x.dw1)
	dwG := x.dw2

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	qrD := divRem_q64r64_64x64x64_mu_barrettStep((qrG.remDw0<<32)|dwD, denom, mu)
	qrC := divRem_q64r64_64x64x64_mu_barrettStep((qrD.remDw0<<32)|dwC, denom, mu)
	qrB := divRem_q64r64_64x64x64_mu_barrettStep((qrC.remDw0<<32)|dwB, denom, mu)
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((qrB.remDw0<<32)|dwA, denom, mu)

	q2 := qrG.quotDw0
	q1 := (qrD.quotDw0 << 32) | qrC.quotDw0
	q0 := (qrB.quotDw0 << 32) | qrA.quotDw0
	return quot256Rem64{q0, q1, q2, 0, qrA.remDw0}
}

func barrett_256_32_mu_q256r64(x u256, denom, mu int64) quot256Rem64 {
	dwA := lo32_64(x.dw0)
	dwB := hi32_64(x.dw0)
	dwC := lo32_64(x.dw1)
	dwD := hi32_64(x.dw1)
	dwE := lo32_64(x.dw2)
	dwF := hi32_64(x.dw2)
	dwG := x.dw3

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	qrF := divRem_q64r64_64x64x64_mu_barrettStep((qrG.remDw0<<32)|dwF, denom, mu)
	qrE := divRem_q64r64_64x64x64_mu_barrettStep((qrF.remDw0<<32)|dwE, denom, mu)
	qrD := divRem_q64r64_64x64x64_mu_barrettStep((qrE.remDw0<<32)|dwD, denom, mu)
	qrC := divRem_q64r64_64x64x64_mu_barrettStep((qrD.remDw0<<32)|dwC, denom, mu)
	qrB := divRem_q64r64_64x64x64_mu_barrettStep((qrC.remDw0<<32)|dwB, denom, mu)
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((qrB.remDw0<<32)|dwA, denom, mu)

	q3 := qrG.quotDw0
	q2 := (qrF.quotDw0 << 32) | qrE.quotDw0
	q1 := (qrD.quotDw0 << 32) | qrC.quotDw0
	q0 := (qrB.quotDw0 << 32) | qrA.quotDw0
	return quot256Rem64{q0, q1, q2, q3, qrA.remDw0}
}

// --- 54-bit chunk kernels (small pow5) ---

func barrett_118_54_mu_q256r64(x u256, denom, mu int64) quot256Rem64 {
	dwA := x.dw0 & mask54L
	dwG := (x.dw1 << 10) | ushr_64_64(x.dw0, 54)

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((qrG.remDw0<<54)|dwA, denom, mu)

	q1 := ushr_64_64(qrG.quotDw0, 10)
	q0 := (qrG.quotDw0 << 54) | qrA.quotDw0
	return quot256Rem64{q0, q1, 0, 0, qrA.remDw0}
}

func barrett_172_54_mu_q256r64(x u256, denom, mu int64) quot256Rem64 {
	dwA := x.dw0 & mask54L
	dwB := ((x.dw1 << 10) | ushr_64_64(x.dw0, 54)) & mask54L
	dwG := (x.dw2 << 20) | ushr_64_64(x.dw1, 44)

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	qrB := divRem_q64r64_64x64x64_mu_barrettStep((qrG.remDw0<<54)|dwB, denom, mu)
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((qrB.remDw0<<54)|dwA, denom, mu)

	q2 := ushr_64_64(qrG.quotDw0, 20)
	q1 := (qrG.quotDw0 << 44) | ushr_64_64(qrB.quotDw0, 10)
	q0 := (qrB.quotDw0 << 54) | qrA.quotDw0
	return quot256Rem64{q0, q1, q2, 0, qrA.remDw0}
}

func barrett_226_54_mu_q256r64(x u256, denom, mu int64) quot256Rem64 {
	dwA := x.dw0 & mask54L
	dwB := ((x.dw1 << 10) | ushr_64_64(x.dw0, 54)) & mask54L
	dwC := ((x.dw2 << 20) | ushr_64_64(x.dw1, 44)) & mask54L
	dwG := (x.dw3 << 30) | ushr_64_64(x.dw2, 34)

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	qrC := divRem_q64r64_64x64x64_mu_barrettStep((qrG.remDw0<<54)|dwC, denom, mu)
	qrB := divRem_q64r64_64x64x64_mu_barrettStep((qrC.remDw0<<54)|dwB, denom, mu)
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((qrB.remDw0<<54)|dwA, denom, mu)

	q3 := ushr_64_64(qrG.quotDw0, 30)
	q2 := (qrG.quotDw0 << 34) | ushr_64_64(qrC.quotDw0, 20)
	q1 := (qrC.quotDw0 << 44) | ushr_64_64(qrB.quotDw0, 10)
	q0 := (qrB.quotDw0 << 54) | qrA.quotDw0
	return quot256Rem64{q0, q1, q2, q3, qrA.remDw0}
}

func barrett_256_54_mu_q256r64(x u256, denom, mu int64) quot256Rem64 {
	dwA := x.dw0 & mask54L
	dwB := ((x.dw1 << 10) | ushr_64_64(x.dw0, 54)) & mask54L
	dwC := ((x.dw2 << 20) | ushr_64_64(x.dw1, 44)) & mask54L
	dwD := ((x.dw3 << 30) | ushr_64_64(x.dw2, 34)) & mask54L
	dwG := ushr_64_64(x.dw3, 24)

	qrG := divRem_q64r64_64x64x64_mu_barrettStep(dwG, denom, mu)
	qrD := divRem_q64r64_64x64x64_mu_barrettStep((qrG.remDw0<<54)|dwD, denom, mu)
	qrC := divRem_q64r64_64x64x64_mu_barrettStep((qrD.remDw0<<54)|dwC, denom, mu)
	qrB := divRem_q64r64_64x64x64_mu_barrettStep((qrC.remDw0<<54)|dwB, denom, mu)
	qrA := divRem_q64r64_64x64x64_mu_barrettStep((qrB.remDw0<<54)|dwA, denom, mu)

	q3 := (qrG.quotDw0 << 24) | ushr_64_64(qrD.quotDw0, 30)
	q2 := (qrD.quotDw0 << 34) | ushr_64_64(qrC.quotDw0, 20)
	q1 := (qrC.quotDw0 << 44) | ushr_64_64(qrB.quotDw0, 10)
	q0 := (qrB.quotDw0 << 54) | qrA.quotDw0
	return quot256Rem64{q0, q1, q2, q3, qrA.remDw0}
}
