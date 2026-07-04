package decimal128

import "testing"

func TestMathBareMethods(t *testing.T) {
	cases := []struct {
		name string
		got  Decimal128
		want string
	}{
		{"Sqrt", MustParse("9").Sqrt(), "3"},
		{"Exp(0)", Zero().Exp(), "1"},
		{"Ln(1)", FromInt64(1).Ln(), "0"},
		{"Log10(1000)", MustParse("1000").Log10(), "3"},
		{"Exp10(2)", FromInt64(2).Exp10(), "1E+2"},
		{"PowInt", FromInt64(2).PowInt(10), "1024"},
		{"Pow", MustParse("9").Pow(MustParse("0.5")), "3"},
		{"Root", MustParse("27").Root(3), "3"},
		{"Compound", MustParse("0.1").Compound(2), "1.21"},
		{"AddProduct", FromInt64(1).AddProduct(FromInt64(2), FromInt64(3)), "7"},
		{"TruncatingRemainder", MustParse("7").TruncatingRemainder(MustParse("3")), "1"},
		{"Remainder", MustParse("7").Remainder(MustParse("3")), "1"},
		{"IntegerDivide", MustParse("7").IntegerDivide(MustParse("2")), "3"},
		{"LogB", MustParse("250").LogB(), "2"},
	}
	for _, c := range cases {
		if got := c.got.String(); got != c.want {
			t.Errorf("%s = %q want %q", c.name, got, c.want)
		}
	}
}

func TestMathContextFlags(t *testing.T) {
	ctx := NewContext()
	// sqrt(2) is irrational → inexact.
	if got := ctx.Sqrt(FromInt64(2)); !got.IsFinite() {
		t.Fatal("sqrt(2) should be finite")
	}
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatalf("sqrt(2) should raise Inexact, got %v", ctx.RaisedFlags())
	}

	ctx.ClearFlags()
	// ln(0) = -inf, divisionByZero.
	r := ctx.Ln(Zero())
	if !r.IsInf() || !r.Signbit() {
		t.Fatalf("ln(0) = %v want -Inf", r)
	}
	if !ctx.RaisedFlags().Has(FlagDivisionByZero) {
		t.Fatalf("ln(0) should raise DivisionByZero, got %v", ctx.RaisedFlags())
	}

	ctx.ClearFlags()
	// FMA exact: 2*3+1 = 7, no flags.
	if got := ctx.FMA(FromInt64(2), FromInt64(3), FromInt64(1)).String(); got != "7" {
		t.Fatalf("FMA = %q", got)
	}
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatalf("exact FMA should raise no flags, got %v", ctx.RaisedFlags())
	}

	// Context power/root/compound agree with the bare forms.
	if ctx.PowInt(FromInt64(2), 8).String() != FromInt64(2).PowInt(8).String() {
		t.Fatal("ctx.PowInt disagrees with bare PowInt")
	}
	if ctx.Pow(MustParse("9"), MustParse("0.5")).String() != MustParse("9").Pow(MustParse("0.5")).String() {
		t.Fatal("ctx.Pow disagrees with bare Pow")
	}
}
