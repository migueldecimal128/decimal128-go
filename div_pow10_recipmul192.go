// The 192-bit-dividend (3x3) reciprocal-multiply divPow10 (1..77): x / 10^pow10
// with a classified residue, taken when the stripped dividend A = x>>(pow10-1)
// fits 3 limbs (A < 2^191). A narrowed sibling of divPow10_q128res_256_recipmul
// (DivPow10RecipMul192WorkOrder.md): the reciprocal M also fits 3 limbs, so the
// recip multiply is a 192x192->384 comba -- 9 64x64 macs vs the 4x4's 16. Only
// the multiply narrows; the strip / shift / residue tail reuses the 256 kernel's
// shared helpers. Backs div_256_pow10_q128res when bitLen(x) <= 190 + pow10.
// Tables in div_pow10_recipmul192_tables.go.

package decimal128

// mul384_192x192 returns the full 192x192->384 product A*M as (pLo, pHi): pLo the
// low 192 bits, pHi the high 192 bits (each with a zero top limb, P < 2^383).
// Reuses macCol512's 192-bit column accumulator; 9 macs / 6 columns.
func mul384_192x192(a0, a1, a2, m0, m1, m2 uint64) (pLo, pHi u256) {
	var acc0, acc1, acc2 uint64

	macCol512(a0, m0, &acc0, &acc1, &acc2)
	r0 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a1, m0, &acc0, &acc1, &acc2)
	macCol512(a0, m1, &acc0, &acc1, &acc2)
	r1 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a2, m0, &acc0, &acc1, &acc2)
	macCol512(a1, m1, &acc0, &acc1, &acc2)
	macCol512(a0, m2, &acc0, &acc1, &acc2)
	r2 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a2, m1, &acc0, &acc1, &acc2)
	macCol512(a1, m2, &acc0, &acc1, &acc2)
	r3 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	macCol512(a2, m2, &acc0, &acc1, &acc2)
	r4 := acc0
	acc0, acc1, acc2 = acc1, acc2, 0

	r5 := acc0

	pLo = u256{int64(r0), int64(r1), int64(r2), 0}
	pHi = u256{int64(r3), int64(r4), int64(r5), 0}
	return pLo, pHi
}

// Accessors for the flat reciprocal table: for index k the three dwords of M
// ascend at rmp10_192_m[k*3 + 0..2] (m3 == 0 by the N=191 gate).
func rmp10_192_dw0(k int) int64 { return rmp10_192_m[k*3] }
func rmp10_192_dw1(k int) int64 { return rmp10_192_m[k*3+1] }
func rmp10_192_dw2(k int) int64 { return rmp10_192_m[k*3+2] }

func divPow10_q128res_192_recipmul(x u256, pow10 int) quot128Residue {
	demand("recipmul192: pow10 outside [1, 77]", pow10 > 0 && pow10 <= 77)
	k := pow10
	m0 := rmp10_192_dw0(k)
	m1 := rmp10_192_dw1(k)
	m2 := rmp10_192_dw2(k)
	s192 := rmp10_192_s192[k] // = S192 - 192, in 1..173
	sh := k - 1               // 0..76

	// Strip k-1 low bits (the 2^(k-1) factor of 10^k); track them as sticky.
	a := shr_256_256(x.dw0, x.dw1, x.dw2, x.dw3, sh)
	stickyAlfa := lowBitsNonZero_256(x.dw0, x.dw1, x.dw2, x.dw3, sh)

	// P = A * M  (384-bit), M = m2:m1:m0.
	pLo, pHi := mul384_192x192(uint64(a.dw0), uint64(a.dw1), uint64(a.dw2),
		uint64(m0), uint64(m1), uint64(m2))

	// quotPlusRound = P >> S192 = pHi >> s192  (= 2q + roundBit).
	quotPlusRound := shr_256_256(pHi.dw0, pHi.dw1, pHi.dw2, pHi.dw3, s192)
	q := shr_256_256(quotPlusRound.dw0, quotPlusRound.dw1, quotPlusRound.dw2, quotPlusRound.dw3, 1)
	roundBit := (quotPlusRound.dw0 & 1) != 0

	// stickyBeta = (P mod 2^S192) >= M: low s192 bits of pHi nonzero
	// (fraction >= 2^192 > M), else compare the low 192 bits (pLo) against M.
	highFrac := lowBitsNonZero_256(pHi.dw0, pHi.dw1, pHi.dw2, pHi.dw3, s192)
	pLoGeM := cmp_int_256x256(pLo.dw0, pLo.dw1, pLo.dw2, pLo.dw3, m0, m1, m2, 0) >= 0
	sticky := stickyAlfa || highFrac || pLoGeM

	res := RESIDUE_EXACT
	if roundBit && sticky {
		res = RESIDUE_GT_HALF
	} else if roundBit {
		res = RESIDUE_HALF
	} else if sticky {
		res = RESIDUE_LT_HALF
	}
	return quot128Residue{q.dw0, q.dw1, res}
}
