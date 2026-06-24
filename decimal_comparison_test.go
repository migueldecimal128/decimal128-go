package decimal128

import "testing"

func TestComparisonNumeric(t *testing.T) {
	one := FromInt(1)
	two := FromInt(2)
	if one.Compare(two) != CmpLess {
		t.Fatal("1 cmp 2")
	}
	if two.Compare(one) != CmpGreater {
		t.Fatal("2 cmp 1")
	}
	if one.Compare(MustParse("1.0")) != CmpEqual {
		t.Fatal("1 cmp 1.0 (cohorts equal)")
	}
	if one.Compare(NaN()) != CmpUnordered {
		t.Fatal("1 cmp NaN")
	}
	if !one.LessOrEqual(one) || !one.LessOrEqual(two) || two.LessOrEqual(one) {
		t.Fatal("LessOrEqual")
	}
	if !one.IsUnordered(NaN()) || one.IsUnordered(two) {
		t.Fatal("IsUnordered")
	}
}

func TestComparisonTotalOrder(t *testing.T) {
	// Distinct cohorts: for positives the smaller exponent sorts first, so
	// 1.00 (exp -2) totalOrders before 1.0 (exp -1) → 1.0 > 1.00.
	a := MustParse("1.0")
	b := MustParse("1.00")
	if a.TotalOrder(b) != CmpGreater {
		t.Fatalf("1.0 vs 1.00 total = %v want Greater", a.TotalOrder(b))
	}
	if b.TotalOrder(a) != CmpLess {
		t.Fatal("1.00 vs 1.0 total")
	}
	if a.TotalOrder(a) != CmpEqual {
		t.Fatal("reflexive total order")
	}
	// -0 totalOrders below +0 (never unordered).
	if MustParse("-0").TotalOrder(Zero()) != CmpLess {
		t.Fatal("-0 < +0 in total order")
	}
	// NaN totalOrders above everything finite — never unordered.
	if FromInt(1).TotalOrder(NaN()) != CmpLess {
		t.Fatal("finite < +NaN in total order")
	}
	if FromInt(1).TotalOrder(NaN()) == CmpUnordered {
		t.Fatal("total order must never be unordered")
	}
	// Magnitude order ignores sign: |-5| vs |3|.
	if MustParse("-5").TotalOrderByMagnitude(FromInt(3)) != CmpGreater {
		t.Fatal("|−5| > |3|")
	}
	if !MustParse("-3").IsTotallyOrderedByMagnitude(FromInt(5)) {
		t.Fatal("|−3| ≤ |5|")
	}
}

func TestComparisonContextFlags(t *testing.T) {
	ctx := NewContext()
	// Quiet compare with a quiet NaN: unordered, no flag.
	if ctx.Compare(FromInt(1), NaN()) != CmpUnordered {
		t.Fatal("quiet compare NaN")
	}
	if !ctx.RaisedFlags().IsEmpty() {
		t.Fatalf("quiet compare with qNaN should raise nothing, got %v", ctx.RaisedFlags())
	}
	// Signaling compare with a quiet NaN: unordered + InvalidOperation.
	if ctx.CompareSignaling(FromInt(1), NaN()) != CmpUnordered {
		t.Fatal("signaling compare NaN")
	}
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Fatalf("signaling compare with NaN should raise InvalidOperation, got %v", ctx.RaisedFlags())
	}
}
