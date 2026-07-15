package decimal128

import "testing"

func TestOrderingClassification(t *testing.T) {
	if !FromInt64(1).IsNormal() || FromInt64(1).IsSubnormal() {
		t.Fatal("1 should be normal")
	}
	// Smallest subnormal magnitude.
	sub := MustParse("1E-6176")
	if !sub.IsSubnormal() || sub.IsNormal() {
		t.Fatalf("1E-6176 should be subnormal")
	}
	if !FromInt64(1).IsCanonical() {
		t.Fatal("internal values are always canonical")
	}
	if Zero().IsNormal() {
		t.Fatal("zero is not normal")
	}
}

func TestOrderingMinMax(t *testing.T) {
	a := FromInt64(3)
	b := FromInt64(7)
	if Minimum(a, b).String() != "3" || Maximum(a, b).String() != "7" {
		t.Fatal("min/max")
	}
	// NaN propagates for the plain forms.
	if !Minimum(a, NaN()).IsNaN() || !Maximum(NaN(), b).IsNaN() {
		t.Fatal("plain min/max should propagate NaN")
	}
	// Number forms ignore a single NaN.
	if MinimumNumber(a, NaN()).String() != "3" {
		t.Fatal("MinimumNumber should ignore NaN")
	}
	if MaximumNumber(NaN(), b).String() != "7" {
		t.Fatal("MaximumNumber should ignore NaN")
	}
	// Magnitude forms.
	if MinimumMagnitude(MustParse("-2"), FromInt64(5)).String() != "-2" {
		t.Fatal("minMagnitude")
	}
	if MaximumMagnitude(MustParse("-9"), FromInt64(5)).String() != "-9" {
		t.Fatal("maxMagnitude")
	}
}

func TestOrderingMinMaxNumberContext(t *testing.T) {
	ctx := NewContext()
	// Number form ignoring a quiet NaN raises no flag.
	if ctx.MinimumNumber(FromInt64(3), NaN()).String() != "3" {
		t.Fatal("ctx MinimumNumber")
	}
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatalf("qNaN in Number form should raise nothing, got %v", ctx.RaisedFlags())
	}
	// A signaling NaN raises InvalidOperation.
	ctx.MaximumNumber(FromInt64(3), SignalingNaN())
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Fatalf("sNaN should raise InvalidOperation, got %v", ctx.RaisedFlags())
	}
}

func TestOrderingCopySign(t *testing.T) {
	if got := FromInt64(5).Copysign(MustParse("-1")).String(); got != "-5" {
		t.Fatalf("copySign(5, -1) = %q want -5", got)
	}
	if got := MustParse("-5").Copysign(FromInt64(1)).String(); got != "5" {
		t.Fatalf("copySign(-5, +1) = %q want 5", got)
	}
}
