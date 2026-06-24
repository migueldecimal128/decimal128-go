package decimal128

import "testing"

func TestContextFlagAccumulation(t *testing.T) {
	ctx := NewContext()
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatal("fresh context should have no flags")
	}

	// 1/3 is inexact.
	ctx.Quo(FromInt(1), FromInt(3))
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatalf("expected Inexact, got %v", ctx.RaisedFlags())
	}

	// 1/0 raises divisionByZero (flags accumulate).
	ctx.Quo(FromInt(1), Zero())
	if !ctx.RaisedFlags().Has(FlagDivisionByZero) {
		t.Fatalf("expected DivisionByZero accumulated, got %v", ctx.RaisedFlags())
	}
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatal("earlier Inexact flag should still be set")
	}

	ctx.ClearFlags()
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatalf("ClearFlags left %v", ctx.RaisedFlags())
	}
}

func TestContextExactNoFlags(t *testing.T) {
	ctx := NewContext()
	if got := ctx.Add(FromInt(2), FromInt(3)); got.String() != "5" {
		t.Fatalf("2+3 = %q", got)
	}
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatalf("exact add should raise no flags, got %v", ctx.RaisedFlags())
	}
}

func TestContextRounding(t *testing.T) {
	ctx := NewContextWithRounding(TowardPositive)
	if got := ctx.Quo(FromInt(1), FromInt(3)).String(); got != "0.3333333333333333333333333333333334" {
		t.Fatalf("TowardPositive 1/3 = %q", got)
	}
	// Changing rounding preserves accumulated flags.
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatal("expected Inexact before SetRounding")
	}
	ctx.SetRounding(TowardZero)
	if ctx.Rounding() != TowardZero {
		t.Fatal("SetRounding not applied")
	}
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatal("SetRounding should preserve accumulated flags")
	}
	if got := ctx.Quo(FromInt(1), FromInt(3)).String(); got != "0.3333333333333333333333333333333333" {
		t.Fatalf("TowardZero 1/3 = %q", got)
	}
}

func TestContextParseOrNaN(t *testing.T) {
	ctx := NewContext()
	if d := ctx.ParseOrNaN("2.5"); d.String() != "2.5" {
		t.Fatalf("parse 2.5 = %q", d)
	}
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatal("clean parse should raise no flags")
	}
	d := ctx.ParseOrNaN("nonsense")
	if !d.IsNaN() {
		t.Fatal("malformed parse should yield NaN")
	}
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Fatalf("malformed parse should raise InvalidOperation, got %v", ctx.RaisedFlags())
	}
}

func TestFlagsString(t *testing.T) {
	if Flags(0).String() != "{}" {
		t.Fatal("empty flags string")
	}
	f := FlagInexact | FlagOverflow
	if got := f.String(); got != "{Overflow, Inexact}" {
		t.Fatalf("flags string = %q", got)
	}
}
