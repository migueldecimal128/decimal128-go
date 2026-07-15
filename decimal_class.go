// The IEEE 754 class of a value.

package decimal128

// DecimalClass is the IEEE 754-2019 §5.7.2 class of a Decimal128 value: one of
// ten mutually exclusive categories. The values match the core class ordering.
type DecimalClass int

const (
	ClassSignalingNaN DecimalClass = iota
	ClassQuietNaN
	ClassNegativeInfinity
	ClassNegativeNormal
	ClassNegativeSubnormal
	ClassNegativeZero
	ClassPositiveZero
	ClassPositiveSubnormal
	ClassPositiveNormal
	ClassPositiveInfinity
)

// String returns a readable name for the class.
func (c DecimalClass) String() string {
	switch c {
	case ClassSignalingNaN:
		return "SignalingNaN"
	case ClassQuietNaN:
		return "QuietNaN"
	case ClassNegativeInfinity:
		return "NegativeInfinity"
	case ClassNegativeNormal:
		return "NegativeNormal"
	case ClassNegativeSubnormal:
		return "NegativeSubnormal"
	case ClassNegativeZero:
		return "NegativeZero"
	case ClassPositiveZero:
		return "PositiveZero"
	case ClassPositiveSubnormal:
		return "PositiveSubnormal"
	case ClassPositiveNormal:
		return "PositiveNormal"
	case ClassPositiveInfinity:
		return "PositiveInfinity"
	default:
		return "DecimalClass(?)"
	}
}

// DecimalClass returns the IEEE 754 class of d.
func (d Decimal128) DecimalClass() DecimalClass { return DecimalClass(d128_ieeeClass(d.core)) }
