// The five IEEE 754-2019 exceptions that may be signaled by decimal floating-point
// operations. Each one's value is a fixed bit position in the Flags word. The values
// are part of the on-the-wire flag encoding (CrossPlatformArchitecture.md Section 6.4);
// do not reorder.
//
// In Go: the names are const_<NAME>, unexported (D-G7a).

package decimal128

type exception754 = int

const (
	const_INVALID_OPERATION exception754 = 0
	const_DIVISION_BY_ZERO  exception754 = 1
	const_OVERFLOW          exception754 = 2
	const_UNDERFLOW         exception754 = 3
	const_INEXACT           exception754 = 4
)

func exc754Mask(e exception754) int64 { return int64(1) << uint(e) }
