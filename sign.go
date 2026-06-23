// The sign representations (CrossPlatformArchitecture.md §4.5): signFlag (bool),
// signBit (0/1 int), signMask (0/-1 int), with named conversions so each
// platform can emit its best branchless lowering. The bare term `sign` is not
// used.

package decimal128

func signBitOf(signFlag bool) int {
	if signFlag {
		return 1
	}
	return 0
}

func signFlagOf(signBit int) bool { return signBit != 0 }

func signMaskOf(signBit int) int { return -signBit }

// boolToInt64 is the int64 spelling of signBitOf, used when packing a sign into
// a dword field.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
