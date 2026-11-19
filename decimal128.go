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
//
// `Decimal128` is the public face of the library: it wraps the internal engine
// value and adapts it to the language's conventions.
//
// The bare arithmetic operations (`+ - * /`) round ties-to-even with no flag
// tracking. For an explicit rounding direction use the `Rounding` methods; to
// also record IEEE 754 exception flags use the `Context` methods.
type Decimal128 struct {
	core d128
}

// --- construction ---

// FromInt64 creates a value exactly equal to the given integer.
func FromInt64(n int64) Decimal128 {
	return Decimal128{d128_fromInt64(int(n))}
}

// FromUInt64 returns the Decimal128 value exactly equal to n.
func FromUInt64(n uint64) Decimal128 {
	return Decimal128{d128_fromUInt64(uint(n))}
}

// Parse parses a decimal string (e.g. `"3.14"`, `"-1E+10"`, `"Infinity"`,
// `"NaN"`). Returns an error if the text is not a valid decimal128 string.
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
// when the text is not a valid decimal128 string. Rounds ties-to-even and
// raises no flags -- the only difference from `Parse` is that failure is
// reported in-band as a NaN rather than as an error. For IEEE/GDAS parse
// semantics (rounding + flags) use `Context.ParseOrNaN`.
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
func NaN() Decimal128 { return Decimal128{const_D128_QNAN0} }

// SignalingNaN returns a signaling NaN with no payload.
func SignalingNaN() Decimal128 { return Decimal128{const_D128_SNAN0} }

// Inf returns positive infinity if `sign >= 0`, negative infinity otherwise.
func Inf(sign int) Decimal128 {
	if sign < 0 {
		return Decimal128{const_D128_NEG_INFINITY}
	}
	return Decimal128{const_D128_POS_INFINITY}
}

// --- classification ---

// IsNaN reports whether this value is a quiet or signaling NaN.
func (d Decimal128) IsNaN() bool { return d128_isNAN(d.core) }

// IsSignalingNaN reports whether this value is a signaling NaN.
func (d Decimal128) IsSignalingNaN() bool { return d128_isSNAN(d.core) }

// IsFinite reports whether this value is finite (neither infinite nor NaN).
func (d Decimal128) IsFinite() bool { return d128_isFinite(d.core) }

// IsInf reports whether this value is positive or negative infinity.
func (d Decimal128) IsInf() bool { return d128_isINF(d.core) }

// IsZero reports whether this value is positive or negative zero.
func (d Decimal128) IsZero() bool { return d128_isZER(d.core) }

// Signbit reports whether the sign of this value is negative. NaN and zero carry
// a sign like any other value.
func (d Decimal128) Signbit() bool { return d128_isNegative(d.core) }

// --- formatting ---

// String returns the GDAS "to-scientific-string" rendering (plain when the
// adjusted exponent allows, scientific otherwise) -- the form
// `BigDecimal.toString` and Python's `decimal.Decimal` produce.
func (d Decimal128) String() string {
	// Implements fmt.Stringer.
	return d128_format(d.core, const_FORMAT_AUTO)
}

// --- arithmetic (ties-to-even, no flags) ---

// Add returns the sum of the operands, rounded ties-to-even, with no flag
// tracking.
func (d Decimal128) Add(other Decimal128) Decimal128 {
	return Decimal128{d128_add_tte(d.core, other.core)}
}

// Sub returns the difference of the operands, rounded ties-to-even, with no
// flag tracking.
func (d Decimal128) Sub(other Decimal128) Decimal128 {
	return Decimal128{d128_subtract_tte(d.core, other.core)}
}

// Mul returns the product of the operands, rounded ties-to-even, with no flag
// tracking.
func (d Decimal128) Mul(other Decimal128) Decimal128 {
	return Decimal128{d128_multiply_tte(d.core, other.core)}
}

// Quo returns the quotient of the operands, rounded ties-to-even, with no flag
// tracking.
func (d Decimal128) Quo(other Decimal128) Decimal128 {
	return Decimal128{d128_divide_tte(d.core, other.core)}
}

// Neg returns bit-level negation: the operand with its sign flipped. Never
// rounds and raises no flags.
func (d Decimal128) Neg() Decimal128 { return Decimal128{d128_negate(d.core)} }

// Abs returns the magnitude (absolute value).
func (d Decimal128) Abs() Decimal128 { return Decimal128{d128_abs(d.core)} }

// --- comparison (IEEE semantics) ---

// Equal reports whether `d == other` under IEEE rules (`+0 == -0`, NaN compares
// unequal).
func (d Decimal128) Equal(other Decimal128) bool {
	return d128_isEqual(d.core, other.core)
}

// Less reports whether `d < other` under IEEE rules (false if either is NaN).
func (d Decimal128) Less(other Decimal128) bool {
	return d128_isLess(d.core, other.core)
}
