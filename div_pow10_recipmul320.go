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
	// shr_320_320 and lowBitsNonZero_320 written out (they don't fit Go's inline
	// budget; D48GoWorkOrder.md Phase 5, option 1): k-1 is 0..95, so the shift
	// moves whole limbs by 0 or 1.
	x0, x1, x2, x3, x4 := uint64(x.dw0), uint64(x.dw1), uint64(x.dw2), uint64(x.dw3), uint64(x.dw4)
	na := uint(k - 1)
	sa := na & 63
	ta := 63 - sa
	maskA := (uint64(1) << sa) - 1
	var a u320
	var stickyAlfa bool
	if na < 64 {
		a = u320{
			int64((x0 >> sa) | ((x1 << 1) << ta)),
			int64((x1 >> sa) | ((x2 << 1) << ta)),
			int64((x2 >> sa) | ((x3 << 1) << ta)),
			int64((x3 >> sa) | ((x4 << 1) << ta)),
			int64(x4 >> sa),
		}
		stickyAlfa = (x0 & maskA) != 0
	} else {
		a = u320{
			int64((x1 >> sa) | ((x2 << 1) << ta)),
			int64((x2 >> sa) | ((x3 << 1) << ta)),
			int64((x3 >> sa) | ((x4 << 1) << ta)),
			int64(x4 >> sa),
			0,
		}
		stickyAlfa = (x0 | (x1 & maskA)) != 0
	}

	// P = A * M (640-bit), P >> S = pHi >> (S - 320) = 2q + roundBit: q is pHi
	// shifted one bit further, and the round bit is bit S - 320 of pHi (S - 320
	// is 2..127, so it is in limb 0 or 1).
	pLo, pHi := mulWide_640_320x320(a, m)
	sh := uint(s - 320)
	h0, h1, h2, h3, h4 := uint64(pHi.dw0), uint64(pHi.dw1), uint64(pHi.dw2), uint64(pHi.dw3), uint64(pHi.dw4)

	// q = pHi >> (sh + 1), written out: sh + 1 is 3..128, a whole-limb move of
	// 0, 1 or 2.
	nq := sh + 1
	q0, q1, q2, q3, q4 := h0, h1, h2, h3, h4
	if nq >= 128 {
		q0, q1, q2, q3, q4 = h2, h3, h4, 0, 0
	} else if nq >= 64 {
		q0, q1, q2, q3, q4 = h1, h2, h3, h4, 0
	}
	sq := nq & 63
	tq := 63 - sq
	q := u320{
		int64((q0 >> sq) | ((q1 << 1) << tq)),
		int64((q1 >> sq) | ((q2 << 1) << tq)),
		int64((q2 >> sq) | ((q3 << 1) << tq)),
		int64((q3 >> sq) | ((q4 << 1) << tq)),
		int64(q4 >> sq),
	}

	// The round bit (bit sh of pHi) and the fraction bits below it (pHi mod
	// 2^sh), from limb 0 or 1.
	maskB := (uint64(1) << (sh & 63)) - 1
	var roundLimb uint64
	var lowFraction bool
	if sh < 64 {
		roundLimb = h0
		lowFraction = (h0 & maskB) != 0
	} else {
		roundLimb = h1
		lowFraction = (h0 | (h1 & maskB)) != 0
	}
	roundBit := ((roundLimb >> (sh & 63)) & 1) != 0

	// stickyBeta = (P mod 2^S) >= M, without a 640-bit compare: high fraction
	// bits (pHi mod 2^(S-320)) nonzero -> >= 2^320 > M; else compare pLo with M.
	stickyBeta := lowFraction || !lt_320(pLo, m)
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
