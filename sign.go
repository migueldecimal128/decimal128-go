// The bool → 0/1 conversions (CrossPlatformArchitecture.md §4.5: signFlag is a
// bool, signBit a 0/1 int). Swift writes `(b) ? 1 : 0`; Go has no conditional
// expression and no bool-to-int conversion, so the two spellings live here.

package decimal128

func signBitOf(signFlag bool) int {
	if signFlag {
		return 1
	}
	return 0
}

// boolToInt64 is the int64 spelling of signBitOf, used when packing a sign into
// a dword field.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
