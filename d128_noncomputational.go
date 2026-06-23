// Non-computational sign operations (IEEE 754-2019 §5.5.1): negate/abs/copy/
// copySign. Bit-level, no rounding, no flags.

package decimal128

func d128_negate(x d128) d128 {
	return d128{x.ubdLo64, x.ubdHi64 ^ (int64(1) << 49)}
}

func d128_negateIf(x d128, performNegation bool) d128 {
	return d128{x.ubdLo64, x.ubdHi64 ^ (boolToInt64(performNegation) << 49)}
}

func d128_abs(x d128) d128 {
	return d128{x.ubdLo64, x.ubdHi64 &^ (int64(1) << 49)}
}

func d128_copy(x d128) d128 { return x }

func d128_copySign(x, signDonor d128) d128 {
	signBit := int64(1) << 49
	return d128{x.ubdLo64, (x.ubdHi64 &^ signBit) | (signDonor.ubdHi64 & signBit)}
}
