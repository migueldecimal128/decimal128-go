// The cold route for GDAS round-05up. The rounding engine (round.go, finalize.go)
// never sees const_ROUND_05_UP. Instead, each operation's rounding entry recognizes
// it and re-enters itself with const_ROUND_TOWARD_ZERO and a local flag sink, then
// finishes here: when the truncated result is inexact and its last digit is 0 or 5,
// add one unit in the last place.
//
// That gives the round-05up result because the two directions agree on everything
// but that one digit:
//   - Round-05up truncates, then moves away from zero only when the discarded digits
//     are nonzero and the kept last digit is 0 or 5.
//   - Adding one to a last digit of 0 or 5 never carries, so the exponent and the
//     digit count are those of the truncated result.
//   - On overflow, toward-zero gives the largest finite value, whose last digit is 9,
//     so it is unchanged; round-05up also gives the largest finite value.
//   - A result that toward-zero collapses to zero is inexact with last digit 0, so it
//     becomes one unit at the result's exponent (for underflow, the smallest
//     subnormal), the round-05up result.
//   - The flags are the same: the increment never crosses into or out of the
//     subnormal range, and never reaches the overflow threshold.
//
// In Go: no closures in the core (CrossPlatformArchitecture.md Section 4), so each
// entry spells the re-entry itself:
//
//	if rounding == const_ROUND_05_UP {
//		tz := newContextRound(const_ROUND_TOWARD_ZERO)
//		return round05Up_finish_ctxnull(op_rnd_ctxnull(..., const_ROUND_TOWARD_ZERO, tz),
//			tz.flags, ctxnull)
//	}

package decimal128

// round05Up_fromTowardZero is round-05up from the toward-zero result: it adds one
// unit in the last place when inexact is set and the finite result's last digit is
// 0 or 5; otherwise it returns truncated unchanged.
func round05Up_fromTowardZero(truncated d128, inexact bool) d128 {
	if !inexact || !d128_isFIN(truncated) {
		return truncated
	}
	// A last digit of 0 or 5 is a coefficient that is a multiple of 5.
	coeff := d128_u128(truncated)
	if !isMultipleOf5_128(coeff.dw0, coeff.dw1) {
		return truncated
	}
	// The coefficient ends in 0 or 5, so it is at most 10^34 - 5: one more stays
	// canonical, and the carry out of the low word lands in the coefficient bits
	// of the high word, leaving the sign and exponent alone.
	lo := truncated.ubdLo64 + 1
	hi := truncated.ubdHi64
	if lo == 0 {
		hi++
	}
	r := d128{ubdLo64: lo, ubdHi64: hi}
	verify("", func() bool { return d128_qExp(r) == d128_qExp(truncated) })
	return r
}

// round05Up_mergeFlags_ctxnull adds every flag in local to ctxnull's sink (no-op
// for a nil context).
func round05Up_mergeFlags_ctxnull(local *flags, ctxnull *context) {
	if ctxnull != nil {
		ctxnull.flags.bits |= local.bits
	}
}

// round05Up_finish_ctxnull finishes the route: it merges the local flags into
// ctxnull and applies round05Up_fromTowardZero with the local inexact flag.
func round05Up_finish_ctxnull(truncated d128, local *flags, ctxnull *context) d128 {
	round05Up_mergeFlags_ctxnull(local, ctxnull)
	return round05Up_fromTowardZero(truncated, local.isSet(const_INEXACT))
}

// isMultipleOf5_128 reports whether the 128-bit unsigned value (dw1:dw0) is a
// multiple of 5. Because 2^32 = 1 (mod 5), the value is congruent to the sum of
// its four 32-bit pieces; that sum is below 2^34, so one signed remainder
// finishes it.
func isMultipleOf5_128(dw0, dw1 int64) bool {
	const mask int64 = 0xFFFF_FFFF
	sum := (dw0 & mask) + ((dw0 >> 32) & mask) + (dw1 & mask) + ((dw1 >> 32) & mask)
	return sum%5 == 0
}
