package decimal128

import "testing"

func TestClass(t *testing.T) {
	cases := []struct {
		in   Decimal128
		want Class
	}{
		{SignalingNaN(), ClassSignalingNaN},
		{NaN(), ClassQuietNaN},
		{Inf(-1), ClassNegativeInfinity},
		{Inf(1), ClassPositiveInfinity},
		{MustParse("-1"), ClassNegativeNormal},
		{FromInt64(1), ClassPositiveNormal},
		{MustParse("1E-6176"), ClassPositiveSubnormal},
		{MustParse("-1E-6176"), ClassNegativeSubnormal},
		{Zero(), ClassPositiveZero},
		{MustParse("-0"), ClassNegativeZero},
	}
	for _, c := range cases {
		if got := c.in.Class(); got != c.want {
			t.Errorf("%s.Class() = %v want %v", c.in, got, c.want)
		}
	}
	if ClassNegativeZero.String() != "NegativeZero" {
		t.Fatal("Class.String")
	}
}
