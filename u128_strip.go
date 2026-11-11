// Trailing-zero decimal-digit stripping (D18, relocated by D20). Bulk removal
// goes through the exact-remainder Barrett pow10 unit in chunks of
// const_BARRETT_POW10_MAX; a nonzero remainder ends the bulk phase and the dword tail
// helper finishes the job.

package decimal128

import "math/bits"

// strip_s128sc_128 strips up to maxToStrip trailing zero decimal digits from c.
func strip_s128sc_128(cDw0, cDw1 int64, maxToStrip int) stripped128StripCount {
	verify("", func() bool { return (cDw0 | cDw1) != 0 })
	stripped := fromLoHi_128_64x64(cDw0, cDw1)
	remaining := maxToStrip
	stripCount128 := 0
	for {
		// trailingZeros of dw0 is 64 when dw0 == 0; both exceed const_BARRETT_POW10_MAX,
		// so the chunk computation below is unaffected.
		ntzBits := bits.TrailingZeros64(uint64(stripped.dw0))
		if ntzBits == 0 || remaining == 0 {
			return stripped128StripCount{stripped.dw0, stripped.dw1, int64(stripCount128)}
		}
		chunk := min(ntzBits, remaining, const_BARRETT_POW10_MAX)
		qr := divPow10_q128r64_128_barrett(stripped.dw0, stripped.dw1, chunk)
		q := fromLoHi_128_64x64(qr.quotDw0, qr.quotDw1)
		rem := uint64(qr.remDw0)
		if rem != 0 {
			stripped64, stripCount64 := u64StripTrailingZeroDigits(rem, remaining)
			scaled := mulPow10_128_128(q.dw0, q.dw1, chunk-stripCount64)
			strippedFinal := add_128_128x128(scaled.dw0, scaled.dw1, int64(stripped64), 0)
			return stripped128StripCount{strippedFinal.dw0, strippedFinal.dw1, int64(stripCount128 + stripCount64)}
		}
		stripped = q
		stripCount128 += chunk
		remaining -= chunk
	}
}

func u64StripTrailingZeroDigits(dw uint64, maxToStrip int) (uint64, int) {
	verify("", func() bool { return dw != 0 })
	verify("", func() bool { return dw < uint64(pow10_128_dw0(14)) })
	t := dw
	ntzBits := bits.TrailingZeros64(dw)
	limit := maxToStrip
	ntzd := 0

	if ntzBits >= 8 && limit >= 8 && t >= 100000000 && t%100000000 == 0 {
		t /= 100000000
		ntzBits = bits.TrailingZeros64(t)
		limit -= 8
		ntzd += 8
	}
	if ntzBits >= 4 && limit >= 4 && t >= 10000 && t%10000 == 0 {
		t /= 10000
		ntzBits = bits.TrailingZeros64(t)
		limit -= 4
		ntzd += 4
	}
	if ntzBits >= 2 && limit >= 2 && t >= 100 && t%100 == 0 {
		t /= 100
		ntzBits = bits.TrailingZeros64(t)
		limit -= 2
		ntzd += 2
	}
	if ntzBits >= 1 && limit >= 1 && t >= 10 && t%10 == 0 {
		t /= 10
		ntzd += 1
	}
	return t, ntzd
}
