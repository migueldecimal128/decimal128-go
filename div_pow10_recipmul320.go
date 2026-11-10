// 320-bit-dividend recipmul divPow10 kernel for the D48 domain
// (D48GoWorkOrder.md Phase 1; Swift DivPow10RecipMul320.swift). The 256-bit
// kernel (D47) one tier up: strip k-1 low bits (the 2^(k-1) factor of 10^k),
// multiply by the 5^k reciprocal M = ceil(2^S / 5^k) (<= 320 bits, five
// dwords) with the explicit 5x5 comba, take P >> S = 2q + roundBit, and
// classify the residue from the product bits. Contract: 1 <= k <= 96, x <
// 10^96 (every D48 intermediate). Tables in div_pow10_recipmul320_tables.go
// (generated).

package decimal128

func divPow10_q320res_320_recipmul(x u320, k int) quot320Residue {
	return recipmul320(x, k)
}

// Fixed-k entry points for d48_finalize_320's common drops: a 95-96-digit
// product loses 47 or 48 digits, a 49-digit quotient 1. Swift's shape (G-7):
// Go does not fold the literal k into recipmul320 (the body is over the
// inliner's budget); Phase 5 decides whether specialized bodies are worth it.

func divPow10_q320res_320_recipmul_48(x u320) quot320Residue { return recipmul320(x, 48) }

func divPow10_q320res_320_recipmul_47(x u320) quot320Residue { return recipmul320(x, 47) }

func divPow10_q320res_320_recipmul_1(x u320) quot320Residue { return recipmul320(x, 1) }

func recipmul320(x u320, k int) quot320Residue {
	verify("", func() bool { return (k >= 1) && (k <= 96) })
	verify("x >= 10^96", func() bool { return cmp_int_320_pow10(x, 96) < 0 })
	kk := k & RMP10_320_BCE
	b := kk * 5
	m := u320{rmp10_320_m[b], rmp10_320_m[b+1], rmp10_320_m[b+2], rmp10_320_m[b+3], rmp10_320_m[b+4]}
	s := int(rmp10_320_s[kk])
	verify("", func() bool { return (s >= 322) && (s <= 447) }) // gen_recipmul320: S - 320 in 2..127

	// Strip k-1 low bits (the 2^(k-1) factor of 10^k); track them as sticky.
	a := shr_320_320(x, k-1)
	stickyAlfa := lowBitsNonZero_320(x, k-1)

	// P = A * M (640-bit), P >> S = pHi >> (S - 320) = 2q + roundBit: q is pHi
	// shifted one bit further, and the round bit is bit S - 320 of pHi (S - 320
	// is 2..127, so it is in limb 0 or 1).
	pLo, pHi := mulWide_640_320x320(a, m)
	sh := s - 320
	q := shr_320_320(pHi, sh+1)
	roundLimb := uint64(pHi.dw1)
	if sh < 64 {
		roundLimb = uint64(pHi.dw0)
	}
	roundBit := ((roundLimb >> uint(sh&63)) & 1) != 0

	// stickyBeta = (P mod 2^S) >= M, without a 640-bit compare: high fraction
	// bits (pHi mod 2^(S-320)) nonzero -> >= 2^320 > M; else compare pLo with M.
	stickyBeta := lowBitsNonZero_320(pHi, sh) || !lt_320(pLo, m)
	sticky := stickyAlfa || stickyBeta

	var res residue
	if roundBit && sticky {
		res = RESIDUE_GT_HALF
	} else if roundBit {
		res = RESIDUE_HALF
	} else if sticky {
		res = RESIDUE_LT_HALF
	} else {
		res = RESIDUE_EXACT
	}
	return quot320Residue{q.dw0, q.dw1, q.dw2, q.dw3, q.dw4, res}
}

// divPow10_q320res is x / 10^k with the residue of the discarded digits,
// 0 <= k <= 96, x < 10^96.
func divPow10_q320res(x u320, k int) quot320Residue {
	if k == 0 {
		return quot320Residue{x.dw0, x.dw1, x.dw2, x.dw3, x.dw4, RESIDUE_EXACT}
	}
	return divPow10_q320res_320_recipmul(x, k)
}

// divPow10_checked_q320res is divPow10_q320res for any k >= 0: beyond 96 the
// quotient is 0 and x (< 10^96 <= 10^(k-1)) is under half a unit unless it is
// zero.
func divPow10_checked_q320res(x u320, k int) quot320Residue {
	verify("", func() bool { return k >= 0 })
	if k <= 96 {
		return divPow10_q320res(x, k)
	}
	if x.isZero() {
		return quot320Residue{residue: RESIDUE_EXACT}
	}
	return quot320Residue{residue: RESIDUE_LT_HALF}
}
