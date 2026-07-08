package decimal128

import "testing"

func TestWrapperParseAndString(t *testing.T) {
	cases := []struct{ in, want string }{
		{"3.14", "3.14"},
		{"-1E+10", "-1E+10"},
		{"0", "0"},
		{"Infinity", "Infinity"},
		{"-Infinity", "-Infinity"},
		{"NaN", "NaN"},
		{"1.00", "1.00"},
	}
	for _, c := range cases {
		d, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.in, err)
		}
		if got := d.String(); got != c.want {
			t.Fatalf("Parse(%q).String() = %q want %q", c.in, got, c.want)
		}
	}
}

func TestWrapperParseError(t *testing.T) {
	if _, err := Parse("not a number"); err == nil {
		t.Fatal("expected error for invalid input")
	}
	if d := ParseOrNaN("garbage"); !d.IsNaN() {
		t.Fatal("ParseOrNaN should return NaN on bad input")
	}
}

func TestWrapperArithmetic(t *testing.T) {
	a := MustParse("1.5")
	b := MustParse("0.3")
	if got := a.Add(b).String(); got != "1.8" {
		t.Fatalf("Add = %q want 1.8", got)
	}
	if got := a.Sub(b).String(); got != "1.2" {
		t.Fatalf("Sub = %q want 1.2", got)
	}
	if got := a.Mul(b).String(); got != "0.45" {
		t.Fatalf("Mul = %q want 0.45", got)
	}
	if got := MustParse("1").Quo(MustParse("4")).String(); got != "0.25" {
		t.Fatalf("Quo = %q want 0.25", got)
	}
	if got := a.Neg().String(); got != "-1.5" {
		t.Fatalf("Neg = %q want -1.5", got)
	}
	if got := MustParse("-7").Abs().String(); got != "7" {
		t.Fatalf("Abs = %q want 7", got)
	}
}

func TestWrapperClassification(t *testing.T) {
	if !NaN().IsNaN() || NaN().IsFinite() {
		t.Fatal("NaN classification")
	}
	if !SignalingNaN().IsSignalingNaN() {
		t.Fatal("signaling NaN classification")
	}
	if !Inf(1).IsInf() || Inf(1).Signbit() {
		t.Fatal("+Inf classification")
	}
	if !Inf(-1).Signbit() {
		t.Fatal("-Inf sign")
	}
	if !Zero().IsZero() || !Zero().IsFinite() {
		t.Fatal("zero classification")
	}
	if !FromInt64(42).IsFinite() || FromInt64(42).IsZero() {
		t.Fatal("FromInt64 classification")
	}
}

func TestWrapperCompare(t *testing.T) {
	one := FromInt64(1)
	two := FromInt64(2)
	if !one.Less(two) || two.Less(one) {
		t.Fatal("Less")
	}
	if !one.Equal(MustParse("1.0")) {
		t.Fatal("Equal across cohorts")
	}
	if NaN().Equal(NaN()) || NaN().Less(one) {
		t.Fatal("NaN comparisons must be false")
	}
	// +0 == -0
	if !Zero().Equal(MustParse("-0")) {
		t.Fatal("+0 should equal -0")
	}
}

func TestWrapperRounding(t *testing.T) {
	a := MustParse("1")
	b := MustParse("3")
	// 1/3 under different directions.
	if got := TowardZero.Quo(a, b).String(); got != "0.3333333333333333333333333333333333" {
		t.Fatalf("TowardZero 1/3 = %q", got)
	}
	if got := TowardPositive.Quo(a, b).String(); got != "0.3333333333333333333333333333333334" {
		t.Fatalf("TowardPositive 1/3 = %q", got)
	}
	if TiesToEven.core() != TIES_TO_EVEN || TowardNegative.core() != TOWARD_NEGATIVE {
		t.Fatal("Rounding.core() mapping")
	}
}
