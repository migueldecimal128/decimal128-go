package decimal128

// Exercises the composite quiet predicates on Decimal128 and the signaling
// predicate family on *Context (which raise InvalidOperation on any NaN
// operand).

import "testing"

func TestCompareFamilyQuietComposites(t *testing.T) {
	two, three, nan := MustParse("2"), MustParse("3"), NaN()
	// NotGreater == NOT(self > other): true when <= or unordered; NotLess mirrors.
	if !two.NotGreater(three) || three.NotGreater(two) || !two.NotGreater(nan) ||
		!three.NotLess(two) || two.NotLess(three) || !two.NotLess(nan) {
		t.Error("NotGreater / NotLess")
	}
	// IsOrdered: true iff neither operand is NaN.
	if !two.IsOrdered(three) || two.IsOrdered(nan) {
		t.Error("IsOrdered")
	}
	// GreaterUnordered / LessUnordered: the relation OR unordered.
	if !three.GreaterUnordered(two) || two.GreaterUnordered(three) || !two.GreaterUnordered(nan) ||
		!two.LessUnordered(three) || three.LessUnordered(two) || !two.LessUnordered(nan) {
		t.Error("GreaterUnordered / LessUnordered")
	}
}

func TestCompareFamilySignalingPredicates(t *testing.T) {
	two, three := MustParse("2"), MustParse("3")
	ctx := NewContextWithRounding(TiesToEven)
	// Boolean results agree with the ordering; no NaN, so no flag raised.
	if !ctx.GreaterSignaling(three, two) || ctx.GreaterSignaling(two, three) || !ctx.LessSignaling(two, three) ||
		!ctx.EqualSignaling(two, MustParse("2.0")) || !ctx.NotEqualSignaling(two, three) ||
		!ctx.LessOrEqualSignaling(two, MustParse("2.0")) || !ctx.GreaterOrEqualSignaling(three, two) ||
		!ctx.NotGreaterSignaling(two, three) || !ctx.NotLessSignaling(three, two) ||
		!ctx.IsOrderedSignaling(two, three) || ctx.IsUnorderedSignaling(two, three) {
		t.Error("signaling predicates")
	}
	if ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("ordered operands raised invalidOperation")
	}
}

func TestCompareFamilySignalingRaisesOnNaN(t *testing.T) {
	two, nan := MustParse("2"), NaN()
	ctx := NewContextWithRounding(TiesToEven)
	// Any NaN operand raises InvalidOperation into the context.
	ctx.GreaterSignaling(two, nan)
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("GreaterSignaling on NaN")
	}
	// A fresh context: unordered-signaling also signals on NaN.
	ctx2 := NewContextWithRounding(TiesToEven)
	ctx2.IsUnorderedSignaling(two, nan)
	if !ctx2.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("IsUnorderedSignaling on NaN")
	}
	// The unordered composites signal too, and still answer true.
	ctx3 := NewContextWithRounding(TiesToEven)
	if !ctx3.GreaterUnorderedSignaling(two, nan) || !ctx3.LessUnorderedSignaling(two, nan) || !ctx3.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("unordered composites on NaN")
	}
}
