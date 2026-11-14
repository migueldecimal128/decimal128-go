// The four-way IEEE 754 comparison result (CrossPlatformArchitecture.md §6).

package decimal128

// Comparison754 tri-state codes (the wrapper/core boundary value). Comparison754
// is an int alias so .rawValue is identity.
type comparison754 = int

const (
	const_CMP754_LT        comparison754 = 0
	const_CMP754_EQ        comparison754 = 1
	const_CMP754_GT        comparison754 = 2
	const_CMP754_UNORDERED comparison754 = 3
)

func comparison754FromSignum(signum int) comparison754 {
	verify("", func() bool { return signum >= -1 && signum <= 1 })
	return signum + 1
}
