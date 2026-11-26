// Comparison754 is the Core's compare currency (used by the compare cores and the IEEE
// 754 predicate family). Its values ARE the tri-state boundary codes (LT = 0, EQ = 1,
// GT = 2, UNORDERED = 3), so the cores cross the wrapper/core boundary with the raw
// value -- no mapping table, no conversion. The CMP754_* constants are the single point
// that names those codes for callers of the `d128_comparePartial*` int API, so the two
// can never drift.
//
// In Go: Comparison754 is an int alias, so .rawValue is identity.

package decimal128

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
