// IEEE 754-2019 §9.2 pown(x, n): x raised to a signed integer power. Left-to-
// right binary exponentiation in the 48-digit D48 domain with a single final
// round (n >= 3 and n <= −2; n <= −2 first tries the exact 2^a / 5^b rule),
// exact fast paths for n ∈ {0, ±1, 2} and |x| ∈ {1, 10}, plus the special
// classes (zero / infinity / NaN). Correctly rounded except in the documented
// 1-ulp gap for bases very close to 1 in directed modes
// (CrossPlatformArchitecture.md §12, "Transcendental accuracy").

package decimal128

import "math/bits"

var d128PownOne = d128_fromInt64(1)
var d128PownTen = d128_fromInt64(10)

// --- ladder ---

func d128_pown_tte(x d128, n int) d128            { return d128_pown_rnd_ctxnull(x, n, ROUND_TIES_TO_EVEN, nil) }
func d128_pown_rnd(x d128, n int, rnd round) d128 { return d128_pown_rnd_ctxnull(x, n, rnd, nil) }
func d128_pown_ctx(x d128, n int, ctx *context) d128 {
	return d128_pown_rnd_ctxnull(x, n, ctx.round, ctx)
}

func d128_pown_rnd_ctxnull(x d128, n int, rounding round, ctxnull *context) d128 {
	if d128_isFIN(x) {
		if d128_isZER(x) {
			return pownZER(x, n, ctxnull)
		}
		return pownFNZ(x, n, rounding, ctxnull)
	}
	if d128_isNAN(x) {
		// pown(x, 0) is 1 unless x is a signaling NaN (§9.2).
		if n == 0 && d128_isQNAN(x) {
			return d128_fromInt64(1)
		}
		return ctxNanOperand1_ctxnull(ctxnull, x)
	}
	verify("", func() bool { return d128_isINF(x) })
	return pownINF(x, n)
}

// --- special classes ---

func pownINF(x d128, n int) d128 {
	resultSign := d128_signFlag(x) && (n&1) != 0
	if n > 0 {
		return d128_infinity(resultSign)
	}
	if n < 0 {
		return d128_zero(resultSign, 0)
	}
	return d128_fromInt64(1)
}

func pownZER(x d128, n int, ctxnull *context) d128 {
	resultSign := d128_signFlag(x) && (n&1) != 0
	if n > 0 {
		// ±0, exponent scaled by n; clamp the multiplier so the product stays
		// well inside int (d128_zero clamps the exponent into range).
		pClamp := min(n, 9999)
		return d128_zero(resultSign, d128_qExp(x)*pClamp)
	}
	if n < 0 {
		return ctxSignalDivByZero_ctxnull(ctxnull, resultSign)
	}
	return d128_fromInt64(1)
}

// --- finite non-zero magnitude ---
//
// n is an int (64-bit); |n| is taken as a uint64 (unsigned negation, so
// math.MinInt is safe) and every product with n or |n| is bounded or saturated
// (D48GoWorkOrder.md G-9; Rust R-9, Zig Z-8). Swift traps at Int.min instead.

func pownFNZ(x d128, n int, rounding round, ctxnull *context) d128 {
	if n == 0 {
		return d128_fromInt64(1)
	}
	if n == 1 {
		return x
	}
	if n == 2 {
		return d128_square_rnd_ctxnull(x, rounding, ctxnull)
	}

	negative := d128_signFlag(x) && (n&1) != 0

	// |x| == 1: result is ±1 carrying the preferred exponent n·Q(x), clamped to
	// [-33, 0] (the product saturated: n may be math.MinInt).
	if cmpMagFnzFnz(x, d128PownOne) == 0 {
		resultQExp := max(min(satMulSigned(n, d128_qExp(x)), 0), -33)
		return d128_oneWithQExp(negative, resultQExp)
	}

	// |x| == 10: an exact power of ten, carrying the preferred cohort exponent.
	if cmpMagFnzFnz(x, d128PownTen) == 0 {
		if n > 6144 {
			return finalizeOverflow_ctxnull(negative, rounding, ctxnull)
		}
		if n < -6176 {
			return finalizeUnderflowFullySwamped(negative, rounding, ctxnull)
		}
		preferredQExpLong := int64(n) * int64(d128_qExp(x)) // |n| <= 6176 here
		minPreferredQExp := minI64(maxI64(int64(n)-33, -32), 0)
		preferredQExp := int(maxI64(preferredQExpLong, minPreferredQExp))
		return tenPowN(negative, n, preferredQExp, rounding, ctxnull)
	}

	// n <= −2: |x|^|n| and the reciprocal in D48, rounded once
	// (pownFNZ_negative_d48). n = −1 is a single correctly rounded divide, so it
	// keeps the D128 path below.
	if n < -1 {
		return pownFNZ_negative_d48(x, -uint64(n), rounding, ctxnull)
	}
	// n = −1: raise to |n| then reciprocate. Choosing raise-first vs
	// reciprocate-first by the estimated result exponent avoids a spurious
	// subnormal underflow on the intermediate.
	if n < 0 {
		absPow := -n // n == -1
		estimatedResultExp := int64(d128_sciExp(x)) * int64(absPow)
		if estimatedResultExp < 6100 {
			raised := pownFNZ(x, absPow, rounding, ctxnull)
			return d128_divide_rnd_ctxnull(d128PownOne, raised, rounding, ctxnull)
		}
		reciprocal := d128_divide_rnd_ctxnull(d128PownOne, x, rounding, ctxnull)
		return pownFNZ(reciprocal, absPow, rounding, ctxnull)
	}

	return pownFNZ_powGE3(x, uint64(n), rounding, ctxnull)
}

// Quantum-exponent guard band on the running D48 ladder value. D48 is a bare
// finite carrier -- it has no infinity -- so the ladder guards qExp instead of
// relying on saturation. A D48 coefficient is at most 48 digits, so a value
// with qExp > OVER has magnitude ≥ 10^6200 and one with qExp < UNDER has
// magnitude < 10^-6203 -- both unrepresentable in every decimal128 cohort --
// hence clamping here never drops a representable result. The ladder magnitude
// is monotonic, so the first band crossing is terminal → the correctly-signed
// overflow / underflow.
const (
	POWN_QEXP_OVER  = 6200
	POWN_QEXP_UNDER = -6250
)

// ladderStartBit is the bit below |n|'s top bit (n >= 2): the left-to-right
// ladder starts from x and walks the remaining bits.
func ladderStartBit(n uint64) uint64 {
	return uint64(1) << (62 - bits.LeadingZeros64(n))
}

func pownFNZ_powGE3(x d128, n uint64, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return n >= 3 })
	return pown_d48Positive(d48_fromD128(x, d128_signFlag(x)), n, pownPositiveExact(x, n), rounding, ctxnull)
}

// --- D48-base bodies (shared with compound, CompoundD48OnePlusXWorkOrder.md CX-2) ---

// pown_d48Ladder is left-to-right binary exponentiation m^n (n >= 1; n = 1
// returns m) in the 48-digit D48 domain. It stops at the first guard-band
// crossing and returns that value: the caller tests d48_qExp against
// POWN_QEXP_OVER / POWN_QEXP_UNDER.
func pown_d48Ladder(m d48, n uint64) d48 {
	if n == 1 {
		return m // the top-bit shift needs n >= 2
	}
	z := m
	for bit := ladderStartBit(n); bit != 0; bit >>= 1 {
		z = d48_square(z)
		if (n & bit) != 0 {
			z = d48_multiply(z, m)
		}
		if (d48_qExp(z) > POWN_QEXP_OVER) || (d48_qExp(z) < POWN_QEXP_UNDER) {
			return z
		}
	}
	return z
}

// pown_d48Positive is m^n for n >= 2 over D48, rounded once into decimal128.
// exact says the true power is exact in decimal128 (pownPositiveExact): no
// exact rule gates the ladder (an exact power never leaves 48 digits, so the
// ladder never rounded it), so z is rounded as is -- the documented 1-ulp gap
// near 1. Its tail can still fall below 48 digits and land z on a decimal128
// point: Inexact comes from the operands, not z.
func pown_d48Positive(m d48, n uint64, exact bool, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return n >= 2 })
	resultSign := d48_signFlag(m) && (n&1) != 0
	z := pown_d48Ladder(m, n)
	if d48_qExp(z) > POWN_QEXP_OVER {
		return finalizeOverflow_ctxnull(resultSign, rounding, ctxnull)
	}
	if d48_qExp(z) < POWN_QEXP_UNDER {
		return finalizeUnderflowFullySwamped(resultSign, rounding, ctxnull)
	}
	r := finalizeD48_ctxnull(z, rounding, ctxnull)
	if exact {
		return r
	}
	return d48_forceInexact(r, ctxnull)
}

// pown_d48Negative is 1/m^absPow (absPow >= 1) over D48: ladder on absPow, one
// D48 divide 1/z, one final rounding. The caller has already run the exact
// rule, so the result is inexact. The guard bands are inverted: a huge z means
// 1/z fully underflows, a tiny z means 1/z overflows.
func pown_d48Negative(m d48, absPow uint64, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return absPow >= 1 })
	resultSign := d48_signFlag(m) && (absPow&1) != 0
	z := pown_d48Ladder(m, absPow)
	if d48_qExp(z) > POWN_QEXP_OVER {
		return finalizeUnderflowFullySwamped(resultSign, rounding, ctxnull)
	}
	if d48_qExp(z) < POWN_QEXP_UNDER {
		return finalizeOverflow_ctxnull(resultSign, rounding, ctxnull)
	}
	r := finalizeD48_ctxnull(d48_divide(d48_fromInt32(1), z), rounding, ctxnull)
	return d48_forceInexact(r, ctxnull)
}

// pownPositiveExact is true if x^n (n >= 2) has at most 34 significant digits,
// i.e. is exact in decimal128 before range limits: c'^n < 10^34 with c' the
// coefficient of x stripped of trailing zeros (c' not a multiple of 10, so
// neither is c'^n).
func pownPositiveExact(x d128, n uint64) bool {
	return pown_positiveExactCoeff(d128_u128(x), n)
}

// pown_positiveExactCoeff is pownPositiveExact on a coefficient: c^n has at
// most 34 significant digits.
func pown_positiveExactCoeff(cIn u128, n uint64) bool {
	c, _ := stripTens(cIn, 0)
	if c == (u128{1, 0}) {
		return true
	}
	p := u128{1, 0}
	for i := uint64(0); i < n; i++ { // c >= 2: at most 113 rounds before p >= 10^34
		t, ok := u128MulChecked(p, c)
		if !ok || !isLt_pow10_34_bool_128(t.dw0, t.dw1) {
			return false
		}
		p = t
	}
	return true
}

// saturate clamps an exponent to ±10^9 (finalize turns it into the overflow /
// underflow it is).
func saturate(v int) int {
	return max(-1_000_000_000, min(1_000_000_000, v))
}

// satMul is a·m clamped to ±2·10^9 (m unsigned, so |math.MinInt| is exact):
// wide enough that saturate of a sum with a small term is unchanged.
func satMul(a int, m uint64) int {
	const lim = 2_000_000_000
	ua := uint64(a)
	if a < 0 {
		ua = -ua
	}
	hi, lo := bits.Mul64(ua, m)
	if (hi != 0) || (lo > lim) {
		lo = lim
	}
	if a < 0 {
		return -int(lo)
	}
	return int(lo)
}

// satMulSigned is n·q clamped to ±2·10^9, for any int n.
func satMulSigned(n, q int) int {
	if n < 0 {
		return -satMul(q, -uint64(n))
	}
	return satMul(q, uint64(n))
}

// pownNegativeExact is the exact 1/x^m (m >= 1) when it exists in decimal128,
// else ok = false. Strip x's trailing zeros: x = c'·10^e. 1/x^m is a
// terminating decimal iff c' = 2^a or 5^b (or 1), and fits 34 digits iff
// a·m <= 48 (5^48 < 10^34) or b·m <= 112 (2^112 < 10^34). The result takes the
// exponent closest to the preferred n·Q(x).
func pownNegativeExact(x d128, m uint64, rounding round, ctxnull *context) (d128, bool) {
	return pown_negativeExactCoeff(d128_signFlag(x), d128_u128(x), d128_qExp(x), d128_qExp(x), m, rounding, ctxnull)
}

// pown_negativeExactCoeff is pownNegativeExact on a value ±c·10^qExp (m >= 1);
// an exact result takes the exponent nearest −m·preferredBaseQExp (the base's
// quantum exponent before any trailing zeros were stripped).
func pown_negativeExactCoeff(xSign bool, c u128, qExp, preferredBaseQExp int, m uint64,
	rounding round, ctxnull *context) (d128, bool) {
	p2 := u128TrailingZeros(c)
	odd := shrU128(c, p2)
	q5 := 0
	for {
		q, r := u128DivRemSmall(odd, 5)
		if r != 0 {
			break
		}
		odd = q
		q5++
	}
	if odd != (u128{1, 0}) {
		return d128{}, false // c' has a prime factor other than 2 and 5
	}
	t := min(p2, q5)
	a := p2 - t // power of two left after stripping tens
	b := q5 - t // power of five left (a and b are not both > 0)
	e := qExp + t
	coeff := u128{1, 0}
	k := 0 // 1/x^m = coeff · 10^(−k − e·m)
	if a > 0 {
		if m > 48 || uint64(a)*m > 48 { // a >= 1; the first test keeps a·m from wrapping
			return d128{}, false
		}
		k = a * int(m)
		for i := 0; i < k; i++ {
			coeff = mul_128_128x128(coeff.dw0, coeff.dw1, 5, 0)
		}
	} else if b > 0 {
		if m > 112 || uint64(b)*m > 112 {
			return d128{}, false
		}
		k = b * int(m)
		coeff = shlU128(coeff, k)
	}
	sign := xSign && ((m & 1) != 0)
	return d48_exactWithPreferred(sign, coeff, saturate(-k-satMul(e, m)), saturate(-satMul(preferredBaseQExp, m)),
		rounding, ctxnull), true
}

// pownFNZ_negative_d48 is pown for n <= −2 (m = |n|): the exact rule, else
// pown_d48Negative.
func pownFNZ_negative_d48(x d128, m uint64, rounding round, ctxnull *context) d128 {
	verify("", func() bool { return m >= 2 })
	if exact, ok := pownNegativeExact(x, m, rounding, ctxnull); ok {
		return exact
	}
	return pown_d48Negative(d48_fromD128(x, d128_signFlag(x)), m, rounding, ctxnull)
}

// --- exact builders ---

// d128_oneWithQExp builds ±1 carrying quantum exponent qExp (in [-33, 0]); the
// coefficient is 10^(-qExp).
func d128_oneWithQExp(sign bool, qExp int) d128 {
	p := -qExp
	return decimalFNZ_128(sign, qExp, pow10_128_dw0(p), pow10_128_dw1(p))
}

// tenPowN builds ±10^n with the preferred cohort exponent preferredQExp;
// finalize clamps the exponent into range / overflows in the large-n branch.
func tenPowN(sign bool, n, preferredQExp int, rounding round, ctxnull *context) d128 {
	coeffExp := n - preferredQExp
	if coeffExp <= 0 || coeffExp > 33 {
		// coefficient 1, qExp = n
		return finalize_128_ctxnull(sign, n, u128{1, 0}, RESIDUE_EXACT, rounding, ctxnull)
	}
	coeff := fromLoHi_128_64x64(pow10_128_dw0(coeffExp), pow10_128_dw1(coeffExp))
	return finalize_128_ctxnull(sign, preferredQExp, coeff, RESIDUE_EXACT, rounding, ctxnull)
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
