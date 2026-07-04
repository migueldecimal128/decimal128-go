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
		hi, lo := d.BID()
		if got := DecimalFromBID(hi, lo); got.Format(StyleCoefficientExponent) != d.Format(StyleCoefficientExponent) {
			t.Errorf("BID round-trip %q: got %q", s, got)
		}
		// DPD round-trip.
		hi, lo = d.DPD()
		if got := DecimalFromDPD(hi, lo); got.Format(StyleCoefficientExponent) != d.Format(StyleCoefficientExponent) {
			t.Errorf("DPD round-trip %q: got %q", s, got)
		}
	}
	// Canonical DPD encoding of 1 is 0x2208…0001.
	hi, lo := FromInt64(1).DPD()
	if hi != 0x2208000000000000 || lo != 0x0000000000000001 {
		t.Errorf("DPD(1) = %#016x %#016x", hi, lo)
	}
}

func TestToInt64(t *testing.T) {
	if got := MustParse("3.9").ToInt64(); got != 3 { // truncate toward zero
		t.Fatalf("3.9.ToInt64() = %d want 3", got)
	}
	if got := MustParse("-3.9").ToInt64(); got != -3 {
		t.Fatalf("-3.9.ToInt64() = %d want -3", got)
	}
	if got := TowardNegative.ToInt64(MustParse("3.9")); got != 3 {
		t.Fatalf("floor 3.9 = %d want 3", got)
	}
	if got := TowardPositive.ToInt64(MustParse("3.1")); got != 4 {
		t.Fatalf("ceil 3.1 = %d want 4", got)
	}
	if got := ToNearestEven.ToInt64(MustParse("2.5")); got != 2 {
		t.Fatalf("nearest-even 2.5 = %d want 2", got)
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
}
