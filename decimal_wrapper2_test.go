package decimal128

import "testing"

func TestStyleFormat(t *testing.T) {
	d := MustParse("1.23E+5")
	cases := []struct {
		style Style
		want  string
	}{
		{StyleAutomatic, d.String()},
		{StyleExponential, "1.23E+5"},
		{StyleEngineering, "123E+3"},
		{StyleCoefficientExponent, "123E+3"},
	}
	for _, c := range cases {
		if got := d.Format(c.style); got != c.want {
			t.Errorf("Format(%v) = %q want %q", c.style, got, c.want)
		}
	}
}

func TestInterchangeRoundTrip(t *testing.T) {
	for _, s := range []string{"3.14", "-1E+10", "0", "-0", "Infinity", "-Infinity", "NaN", "1.00", "9.999999999999999999999999999999999E+6144"} {
		d := MustParse(s)
		// BID round-trip (bitwise via re-render).
		hi, lo := d.BID128()
		if got := FromBID128(hi, lo); got.Format(StyleCoefficientExponent) != d.Format(StyleCoefficientExponent) {
			t.Errorf("BID round-trip %q: got %q", s, got)
		}
		// DPD round-trip.
		hi, lo = d.DPD128()
		if got := FromDPD128(hi, lo); got.Format(StyleCoefficientExponent) != d.Format(StyleCoefficientExponent) {
			t.Errorf("DPD round-trip %q: got %q", s, got)
		}
	}
	// Canonical DPD encoding of 1 is 0x2208…0001.
	hi, lo := FromInt64(1).DPD128()
	if hi != 0x2208000000000000 || lo != 0x0000000000000001 {
		t.Errorf("DPD(1) = %#016x %#016x", hi, lo)
	}
}

func TestInt64Conversions(t *testing.T) {
	threeSeven := MustParse("3.7")
	twoFive := MustParse("2.5")

	// (a) Range guard: rounding-independent, range only (not integrality).
	if !threeSeven.FitsInt64() {
		t.Fatalf("3.7 should fit int64")
	}
	if !threeSeven.FitsUInt64() {
		t.Fatalf("3.7 should fit uint64")
	}
	if !threeSeven.Neg().FitsInt64() {
		t.Fatalf("-3.7 should fit int64")
	}
	if threeSeven.Neg().FitsUInt64() { // negative nonzero
		t.Fatalf("-3.7 should not fit uint64")
	}
	if MustParse("-0").FitsUInt64() != true { // -0 converts to 0
		t.Fatalf("-0 should fit uint64")
	}
	if MustParse("1E100").FitsInt64() {
		t.Fatalf("1E100 should not fit int64")
	}
	if NaN().FitsInt64() || Inf(1).FitsInt64() {
		t.Fatalf("NaN/Inf should not fit int64")
	}

	// (b) Fallible-exact: succeeds only for exact integers in range.
	if got, err := FromInt64(3).Int64(); err != nil || got != 3 {
		t.Fatalf("Int64(3) = %d, %v want 3, nil", got, err)
	}
	if got, err := FromInt64(-3).Int64(); err != nil || got != -3 {
		t.Fatalf("Int64(-3) = %d, %v want -3, nil", got, err)
	}
	if got, err := FromInt64(3).Uint64(); err != nil || got != 3 {
		t.Fatalf("Uint64(3) = %d, %v want 3, nil", got, err)
	}
	if _, err := threeSeven.Int64(); err == nil { // fractional → error
		t.Fatalf("Int64(3.7) should fail")
	}
	if _, err := FromInt64(-3).Uint64(); err == nil { // negative → error
		t.Fatalf("Uint64(-3) should fail")
	}
	if _, err := MustParse("1E100").Int64(); err == nil { // out of range → error
		t.Fatalf("Int64(1E100) should fail")
	}
	if _, err := NaN().Int64(); err == nil { // NaN → error
		t.Fatalf("Int64(NaN) should fail")
	}
	if _, err := Inf(1).Int64(); err == nil { // infinity → error
		t.Fatalf("Int64(Inf) should fail")
	}
	// The failure is a typed *IntConversionError.
	if _, err := threeSeven.Int64(); err != nil {
		if _, ok := err.(*IntConversionError); !ok {
			t.Fatalf("expected *IntConversionError, got %T", err)
		}
	}

	// (c) Directed forms round a fraction (unchanged).
	if got := TiesToEven.ToInt64(threeSeven); got != 4 {
		t.Fatalf("tiesToEven 3.7 = %d want 4", got)
	}
	if got := TiesToEven.ToInt64(twoFive); got != 2 { // ties to even
		t.Fatalf("tiesToEven 2.5 = %d want 2", got)
	}
	if got := TowardPositive.ToInt64(twoFive); got != 3 {
		t.Fatalf("towardPositive 2.5 = %d want 3", got)
	}
	if got := TowardNegative.ToInt64(MustParse("3.9")); got != 3 {
		t.Fatalf("floor 3.9 = %d want 3", got)
	}

	// Context: exact integer raises nothing; fractional ToInt64Exact raises Inexact.
	ctx := NewContext()
	if got := ctx.ToInt64(FromInt64(42)); got != 42 || !ctx.RaisedFlags().IsEmpty() {
		t.Fatalf("ctx.ToInt64(42) = %d flags %v", got, ctx.RaisedFlags())
	}
	ctx.ToInt64Exact(MustParse("3.5"))
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatalf("ToInt64Exact of a fraction should raise Inexact, got %v", ctx.RaisedFlags())
	}
	// NaN conversion raises InvalidOperation.
	ctx.ClearFlags()
	ctx.ToInt64(NaN())
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Fatalf("NaN→int should raise InvalidOperation, got %v", ctx.RaisedFlags())
	}
	// Out-of-range is an invalid conversion.
	ctx.ClearFlags()
	ctx.ToInt64(MustParse("1E100"))
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Fatalf("1E100→int should raise InvalidOperation, got %v", ctx.RaisedFlags())
	}
}
