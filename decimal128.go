// Package decimal128 implements IEEE 754-2019 decimal128 floating point.
//
// Decimal128 is the public, Go-idiomatic face of the library; it wraps the
// package-internal d128 engine and adapts it to Go conventions — value methods,
// error-returning parsing, and idiomatic enums for rounding and formatting. The
// d128 / d128_* engine boundary is deliberately unexported: callers work only
// with Decimal128.
//
// The bare arithmetic methods (Add, Sub, Mul, Quo) round ties-to-even and track
// no flags, the way float64 arithmetic does. For an explicit rounding direction
// use the Rounding receiver methods (e.g. rnd.Add); for IEEE 754 exception
// flags use a Context.
package decimal128

import "strconv"

// Decimal128 is an IEEE 754-2019 decimal128 floating-point value.
type Decimal128 struct {
	core d128
}

// --- construction ---

// FromInt returns the Decimal128 value exactly equal to n.
func FromInt(n int) Decimal128 {
	return Decimal128{d128_fromInt(n)}
}

// Parse parses a decimal string (e.g. "3.14", "-1E+10", "Infinity", "NaN"),
// rounding ties-to-even. It returns an error if the text is not a valid
// decimal128 string.
func Parse(text string) (Decimal128, error) {
	if parsed, ok := d128_parse(text); ok {
		return Decimal128{parsed}, nil
	}
	return Decimal128{}, &ParseError{text}
}

// MustParse is like Parse but panics if the text is not a valid decimal128
// string. It is intended for package-level values and tests where the input is
// author-controlled.
func MustParse(text string) Decimal128 {
	d, err := Parse(text)
	if err != nil {
		panic(err)
	}
	return d
}

// ParseOrNaN parses a decimal string, returning a quiet NaN instead of an error
// when the text is not a valid decimal128 string. It rounds ties-to-even and
// raises no flags.
func ParseOrNaN(text string) Decimal128 {
	return Decimal128{d128_parseOrNaN(text)}
}

// ParseError reports a string that could not be parsed as a decimal128 value.
type ParseError struct{ Text string }

func (e *ParseError) Error() string {
	return "decimal128: cannot parse " + strconv.Quote(e.Text)
}

// --- well-known values ---

// Zero returns positive zero with a quantum exponent of 0.
func Zero() Decimal128 { return Decimal128{d128_zero(false, 0)} }

// NaN returns a quiet NaN with no payload.
func NaN() Decimal128 { return Decimal128{D128_QNAN0} }

// SignalingNaN returns a signaling NaN with no payload.
func SignalingNaN() Decimal128 { return Decimal128{D128_SNAN0} }

// Inf returns positive infinity if sign >= 0, negative infinity otherwise.
func Inf(sign int) Decimal128 {
	if sign < 0 {
		return Decimal128{D128_NEG_INFINITY}
	}
	return Decimal128{D128_POS_INFINITY}
}

// --- classification ---

// IsNaN reports whether d is a quiet or signaling NaN.
func (d Decimal128) IsNaN() bool { return d128_isNAN(d.core) }

// IsSignalingNaN reports whether d is a signaling NaN.
func (d Decimal128) IsSignalingNaN() bool { return d128_isSNAN(d.core) }

// IsFinite reports whether d is finite (neither infinite nor NaN).
func (d Decimal128) IsFinite() bool { return d128_isFinite(d.core) }

// IsInf reports whether d is positive or negative infinity.
func (d Decimal128) IsInf() bool { return d128_isINF(d.core) }

// IsZero reports whether d is positive or negative zero.
func (d Decimal128) IsZero() bool { return d128_isZER(d.core) }

// Signbit reports whether d carries a negative sign (including −0 and −NaN).
func (d Decimal128) Signbit() bool { return d128_isNegative(d.core) }

// --- formatting ---

// String returns the GDAS "to-scientific-string" rendering of d (plain when the
// adjusted exponent allows, scientific otherwise) — the form BigDecimal.toString
// and Python's decimal.Decimal produce. It implements fmt.Stringer.
func (d Decimal128) String() string { return d128_format(d.core, FORMAT_AUTO) }

// --- arithmetic (ties-to-even, no flags) ---

// Add returns d + other, rounded ties-to-even.
func (d Decimal128) Add(other Decimal128) Decimal128 {
	return Decimal128{d128_add_tte(d.core, other.core)}
}

// Sub returns d − other, rounded ties-to-even.
func (d Decimal128) Sub(other Decimal128) Decimal128 {
	return Decimal128{d128_subtract_tte(d.core, other.core)}
}

// Mul returns d × other, rounded ties-to-even.
func (d Decimal128) Mul(other Decimal128) Decimal128 {
	return Decimal128{d128_multiply_tte(d.core, other.core)}
}

// Quo returns d ÷ other, rounded ties-to-even.
func (d Decimal128) Quo(other Decimal128) Decimal128 {
	return Decimal128{d128_divide_tte(d.core, other.core)}
}

// Neg returns −d.
func (d Decimal128) Neg() Decimal128 { return Decimal128{d128_negate(d.core)} }

// Abs returns the magnitude (absolute value) of d.
func (d Decimal128) Abs() Decimal128 { return Decimal128{d128_abs(d.core)} }

// --- comparison (IEEE semantics) ---

// Equal reports whether d and other are numerically equal under IEEE 754
// semantics: +0 equals −0, and a NaN equals nothing, itself included.
func (d Decimal128) Equal(other Decimal128) bool {
	return d128_isEqual(d.core, other.core)
}

// Less reports whether d is numerically less than other under IEEE 754
// semantics (any NaN operand yields false).
func (d Decimal128) Less(other Decimal128) bool {
	return d128_isLess(d.core, other.core)
}
