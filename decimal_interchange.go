// IEEE 754-2019 interchange encodings as 128-bit patterns (a high/low uint64
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

// BID returns d's 128-bit IEEE 754 BID interchange pattern as a high/low uint64
// pair (hi is the most-significant 64 bits).
func (d Decimal128) BID() (hi, lo uint64) {
	u := d128_to_bid128(d.core)
	return uint64(u.dw1), uint64(u.dw0)
}

// DecimalFromBID decodes a value from its 128-bit BID interchange pattern.
func DecimalFromBID(hi, lo uint64) Decimal128 {
	return Decimal128{d128_from_bid128(u128{int64(lo), int64(hi)})}
}

// DPD returns d's 128-bit IEEE 754 DPD interchange pattern as a high/low uint64
// pair (hi is the most-significant 64 bits).
func (d Decimal128) DPD() (hi, lo uint64) {
	u := d128_to_dpd128(d.core)
	return uint64(u.dw1), uint64(u.dw0)
}

// DecimalFromDPD decodes a value from its 128-bit DPD interchange pattern.
func DecimalFromDPD(hi, lo uint64) Decimal128 {
	return Decimal128{d128_from_dpd128(u128{int64(lo), int64(hi)})}
}
