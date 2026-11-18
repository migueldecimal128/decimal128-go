package decimal128

// Exercises the public quiet comparison predicates on Decimal128 (the named
// IEEE forms).

import "testing"

func TestQuietComparisonNamedQuietPredicates(t *testing.T) {
	two, three := MustParse("2"), MustParse("3")
	if !two.Equal(MustParse("2.0")) || !two.NotEqual(three) || two.NotEqual(MustParse("2.0")) ||
		!two.Less(three) || !two.LessOrEqual(MustParse("2.0")) ||
		!three.Greater(two) || two.Greater(three) || two.Greater(MustParse("2.0")) ||
		!three.GreaterOrEqual(two) || !two.GreaterOrEqual(MustParse("2.0")) || two.GreaterOrEqual(three) {
		t.Error("named quiet predicates")
	}
}

func TestQuietComparisonNanIsUnordered(t *testing.T) {
	two, nan := MustParse("2"), NaN()
	// NaN is unordered: not-equal is true; ordering predicates are false.
	if !two.NotEqual(nan) || two.Greater(nan) || two.GreaterOrEqual(nan) || nan.Greater(two) || !two.IsUnordered(nan) {
		t.Error("NaN unordered")
	}
}
