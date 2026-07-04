package decimal128

import "testing"

func TestRoundingToIntegral(t *testing.T) {
	if got := MustParse("2.5").RoundToIntegral().String(); got != "2" { // ties to even
		t.Fatalf("2.5 round = %q want 2", got)
	}
	if got := MustParse("3.5").RoundToIntegral().String(); got != "4" {
		t.Fatalf("3.5 round = %q want 4", got)
	}
	if got := TowardPositive.RoundToIntegral(MustParse("2.1")).String(); got != "3" {
		t.Fatalf("ceil(2.1) = %q want 3", got)
	}
	if got := TowardNegative.RoundToIntegral(MustParse("2.9")).String(); got != "2" {
		t.Fatalf("floor(2.9) = %q want 2", got)
	}
	if got := TowardZero.RoundToIntegral(MustParse("-2.9")).String(); got != "-2" {
		t.Fatalf("trunc(-2.9) = %q want -2", got)
	}

	ctx := NewContext()
	if got := ctx.RoundToIntegral(MustParse("2.5")).String(); got != "2" {
		t.Fatalf("ctx round 2.5 = %q", got)
	}
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatalf("rounding off a fraction should raise Inexact, got %v", ctx.RaisedFlags())
	}
}

func TestRoundingAdjacent(t *testing.T) {
	one := FromInt64(1)
	up := one.NextUp()
	if !one.Less(up) {
		t.Fatal("NextUp should be greater")
	}
	if !up.NextDown().Equal(one) {
		t.Fatal("NextDown∘NextUp should return to start")
	}
}

func TestRoundingScaleAndQuantum(t *testing.T) {
	if got := MustParse("1.5").ScaleB(3).String(); got != "1.5E+3" {
		t.Fatalf("scaleB = %q want 1.5E+3", got)
	}
	if got := MustParse("2.345").Quantize(MustParse("1.00")).String(); got != "2.34" { // ties to even
		t.Fatalf("quantize = %q want 2.34", got)
	}
	if !MustParse("1.00").SameQuantum(MustParse("9.99")) {
		t.Fatal("1.00 and 9.99 should share quantum")
	}
	if MustParse("1.0").SameQuantum(MustParse("1.00")) {
		t.Fatal("1.0 and 1.00 should NOT share quantum")
	}
	if got := MustParse("1.234").QuantumExponent(); got != -3 {
		t.Fatalf("quantumExponent = %d want -3", got)
	}
	if got := MustParse("1.2300").StripTrailingZeros().String(); got != "1.23" {
		t.Fatalf("strip = %q want 1.23", got)
	}

	ctx := NewContext()
	if got := ctx.Quantize(MustParse("2.345"), MustParse("1.0")).String(); got != "2.3" {
		t.Fatalf("ctx quantize = %q want 2.3", got)
	}
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Fatalf("lossy quantize should raise Inexact, got %v", ctx.RaisedFlags())
	}
}
