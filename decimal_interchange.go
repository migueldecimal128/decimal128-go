// IEEE 754-2019 interchange encodings as 128-bit values (a high/low uint64
// pair). Two encodings are standardized:
//
//   - BID (Binary Integer Decimal): the significand is a binary integer; used by
//     Intel's libbid and its test vectors.
//   - DPD (Densely Packed Decimal): the significand is packed decimal declets;
//     used by IBM/Cowlishaw decNumber and the dectest vectors.
//
// Both round-trip losslessly with the internal representation, including the
// cohort (exponent) and NaN payloads — they are pure re-encodings, never
// rounding or signalling.

package decimal128

// BID128 returns d's BID128 interchange encoding (128-bit IEEE 754) as a
// high/low uint64 pair (hi64 is the most-significant 64 bits).
func (d Decimal128) BID128() (hi64, lo64 uint64) {
	u := d128_to_bid128(d.core)
	return uint64(u.dw1), uint64(u.dw0)
}

// FromBID128 decodes a value from its BID128 interchange encoding.
func FromBID128(hi64, lo64 uint64) Decimal128 {
	return Decimal128{d128_from_bid128(u128{int64(lo64), int64(hi64)})}
}

// DPD128 returns d's DPD128 interchange encoding (128-bit IEEE 754) as a
// high/low uint64 pair (hi64 is the most-significant 64 bits).
func (d Decimal128) DPD128() (hi64, lo64 uint64) {
	u := d128_to_dpd128(d.core)
	return uint64(u.dw1), uint64(u.dw0)
}

// FromDPD128 decodes a value from its DPD128 interchange encoding.
func FromDPD128(hi64, lo64 uint64) Decimal128 {
	return Decimal128{d128_from_dpd128(u128{int64(lo64), int64(hi64)})}
}
