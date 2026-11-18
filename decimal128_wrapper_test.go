package decimal128

// Exercises the public wrapper surface end-to-end, through the exported API
// only — the API a real consumer sees, sitting on top of the validated core.
// The Go idioms (fmt.Stringer, error-returning parse and conversions) are
// checked inside the matching test.

import (
	"sort"
	"strings"
	"testing"
)

func TestDecimal128WrapperConstructFromInt(t *testing.T) {
	if FromInt64(42).String() != "42" || FromInt64(-7).String() != "-7" || !FromInt64(0).IsZero() {
		t.Error("FromInt64")
	}
	if !FromUInt64(7).Equal(FromInt64(7)) {
		t.Error("FromUInt64")
	}
}

func TestDecimal128WrapperParseValidAndInvalid(t *testing.T) {
	if v, err := Parse("3.14"); err != nil || v.String() != "3.14" {
		t.Error("3.14")
	}
	if _, err := Parse("-1E+10"); err != nil {
		t.Error("-1E+10")
	}
	if _, err := Parse("not a number"); err == nil {
		t.Error("not a number accepted")
	}
}

func TestDecimal128WrapperConstructionFromValueAndString(t *testing.T) {
	if !FromInt64(100).Equal(MustParse("100")) || MustParse("2.5").String() != "2.5" {
		t.Error("construction")
	}
}

func TestDecimal128WrapperArithmetic(t *testing.T) {
	// The headline decimal property: 0.1 + 0.2 is exactly 0.3.
	if MustParse("0.1").Add(MustParse("0.2")).String() != "0.3" {
		t.Error("0.1 + 0.2")
	}
	if !FromInt64(6).Mul(FromInt64(7)).Equal(FromInt64(42)) || !FromInt64(10).Sub(FromInt64(3)).Equal(FromInt64(7)) ||
		!FromInt64(1).Quo(FromInt64(4)).Equal(MustParse("0.25")) {
		t.Error("arithmetic")
	}
}

func TestDecimal128WrapperAccumulation(t *testing.T) {
	acc := Zero()
	acc = acc.Add(FromInt64(5))
	acc = acc.Sub(FromInt64(2))
	acc = acc.Mul(FromInt64(4))
	if !acc.Equal(FromInt64(12)) {
		t.Error("accumulation")
	}
}

func TestDecimal128WrapperComparingAndSorting(t *testing.T) {
	v := []Decimal128{FromInt64(3), FromInt64(1), FromInt64(2), FromInt64(-5), FromInt64(10)}
	sort.Slice(v, func(i, j int) bool { return v[i].Less(v[j]) })
	for i, n := range []int64{-5, 1, 2, 3, 10} {
		if !v[i].Equal(FromInt64(n)) {
			t.Errorf("sorted[%d]", i)
		}
	}
	if !FromInt64(1).Less(FromInt64(2)) || !Zero().Equal(FromInt64(0)) {
		t.Error("ordering")
	}
}

func TestDecimal128WrapperIeeeEqualitySemantics(t *testing.T) {
	// +0 == -0, and NaN != NaN, matching float64.
	if !FromInt64(0).Equal(FromInt64(0).Neg()) || NaN().Equal(NaN()) || NaN().Less(FromInt64(1)) {
		t.Error("IEEE equality")
	}
}

func TestDecimal128WrapperSignAndMagnitude(t *testing.T) {
	if FromInt64(3).Neg().String() != "-3" || !FromInt64(-9).Abs().Equal(FromInt64(9)) || !FromInt64(4).Neg().Equal(FromInt64(-4)) {
		t.Error("sign / magnitude")
	}
}

func TestDecimal128WrapperClassification(t *testing.T) {
	if !Inf(1).IsInf() || !Inf(-1).IsInf() || !NaN().IsNaN() || !SignalingNaN().IsSignalingNaN() ||
		!FromInt64(0).IsFinite() || !FromInt64(-1).Signbit() || FromInt64(1).Signbit() {
		t.Error("classification")
	}
}

func TestDecimal128WrapperRoundingModeBridge(t *testing.T) {
	// Toward-zero keeps the last digit a 3, never rounding the tail up.
	if !strings.HasSuffix(TowardZero.Quo(FromInt64(1), FromInt64(3)).String(), "3") ||
		!strings.HasSuffix(TowardPositive.Quo(FromInt64(1), FromInt64(3)).String(), "4") {
		t.Error("rounding bridge")
	}
}

func TestDecimal128WrapperFormatStyles(t *testing.T) {
	v := MustParse("1500")
	if v.Text(StyleAutomatic) != "1500" || v.Text(StyleExponential) != "1.500E+3" || v.Text(StyleCoefficientExponent) != "1500E+0" {
		t.Error("styles")
	}
	// Engineering keeps a value within plain range as-is (it only forces an
	// exponent — a multiple of 3 — once out of plain range).
	if v.Text(StyleEngineering) != "1500" {
		t.Error("engineering")
	}
	w := MustParse("0.00001234")
	if w.Text(StyleExponential) != "1.234E-5" || w.String() != "0.00001234" {
		t.Error("small value")
	}
}

func TestDecimal128WrapperDecimalFlagsBitLayout(t *testing.T) {
	// Raw bit positions are load-bearing: they must equal the core's
	// Exception754 ordering so a snapshot crosses as a bare integer.
	if FlagInvalidOperation != 1<<0 || FlagDivisionByZero != 1<<1 || FlagOverflow != 1<<2 || FlagUnderflow != 1<<3 || FlagInexact != 1<<4 {
		t.Error("flag bit layout")
	}
}

func TestDecimal128WrapperDecimalFlagsSetAlgebra(t *testing.T) {
	f := FlagInexact | FlagOverflow
	if !f.Has(FlagOverflow) || f.Has(FlagUnderflow) || f != (1<<4)|(1<<2) {
		t.Error("set membership")
	}
	f |= FlagUnderflow
	if !f.Has(FlagUnderflow) || !Flags(0).IsEmpty() {
		t.Error("insert / empty")
	}
}

func TestDecimal128WrapperContextAccumulatesAndClearsFlags(t *testing.T) {
	ctx := NewContext()
	if !ctx.RaisedFlags().IsEmpty() {
		t.Error("fresh context has flags")
	}
	ctx.Quo(FromInt64(1), FromInt64(3)) // 1/3 is inexact
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Error("1/3 not inexact")
	}
	ctx.ClearFlags()
	if !ctx.RaisedFlags().IsEmpty() {
		t.Error("ClearFlags")
	}
	ctx.Add(FromInt64(2), FromInt64(2)) // an exact operation raises nothing
	if !ctx.RaisedFlags().IsEmpty() {
		t.Error("exact add raised flags")
	}
}

func TestDecimal128WrapperContextSignalsDivisionByZero(t *testing.T) {
	ctx := NewContext()
	if !ctx.Quo(FromInt64(1), FromInt64(0)).IsInf() || !ctx.RaisedFlags().Has(FlagDivisionByZero) {
		t.Error("division by zero")
	}
}

func TestDecimal128WrapperContextRoundingDirects(t *testing.T) {
	// Same operation, two directions, through a context.
	down := NewContextWithRounding(TowardZero)
	up := NewContextWithRounding(TowardPositive)
	if !strings.HasSuffix(down.Quo(FromInt64(2), FromInt64(3)).String(), "6") || // 0.666…6 truncated
		!strings.HasSuffix(up.Quo(FromInt64(2), FromInt64(3)).String(), "7") || // rounded up
		!down.RaisedFlags().Has(FlagInexact) {
		t.Error("context rounding")
	}
	ctx := NewContext()
	ctx.SetRounding(TowardZero)
	if ctx.Rounding() != TowardZero {
		t.Error("SetRounding")
	}
}

// MARK: - Broadened op coverage

func TestDecimal128WrapperSquareRootAndFma(t *testing.T) {
	if !FromInt64(144).Sqrt().Equal(FromInt64(12)) || !FromInt64(9).Sqrt().Equal(FromInt64(3)) {
		t.Error("sqrt")
	}
	// 1 + 2*3 with a single rounding (the receiver is the addend).
	if !FromInt64(1).FMA(FromInt64(2), FromInt64(3)).Equal(FromInt64(7)) {
		t.Error("fma")
	}
	ctx := NewContext()
	ctx.Sqrt(FromInt64(2)) // √2 is inexact
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Error("√2 not inexact")
	}
}

func TestDecimal128WrapperTranscendentals(t *testing.T) {
	if !FromInt64(0).Exp().Equal(FromInt64(1)) || !FromInt64(2).Exp10().Equal(FromInt64(100)) ||
		!FromInt64(1).Ln().Equal(FromInt64(0)) || !MustParse("1000").Log10().Equal(FromInt64(3)) {
		t.Error("transcendentals")
	}
}

func TestDecimal128WrapperRemainders(t *testing.T) {
	// Truncating: quotient toward zero (5/3 -> 1, rem 2); IEEE nearest: 5/3 -> 2, rem -1.
	if !FromInt64(5).RemainderTrunc(FromInt64(3)).Equal(FromInt64(2)) || !FromInt64(5).RemainderNear(FromInt64(3)).Equal(FromInt64(-1)) {
		t.Error("remainders")
	}
}

func TestDecimal128WrapperRoundToIntegral(t *testing.T) {
	if !MustParse("2.5").RoundToIntegral().Equal(FromInt64(2)) || !MustParse("3.5").RoundToIntegral().Equal(FromInt64(4)) || // ties to even
		!TowardZero.RoundToIntegral(MustParse("2.5")).Equal(FromInt64(2)) || !TowardPositive.RoundToIntegral(MustParse("2.5")).Equal(FromInt64(3)) {
		t.Error("round to integral")
	}
	ctx := NewContext()
	ctx.RoundToIntegral(MustParse("2.5"))
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Error("not inexact")
	}
}

func TestDecimal128WrapperAdjacentValues(t *testing.T) {
	if !FromInt64(1).NextUp().Greater(FromInt64(1)) || !FromInt64(1).NextDown().Less(FromInt64(1)) {
		t.Error("adjacent values")
	}
}

func TestDecimal128WrapperScaleAndQuantum(t *testing.T) {
	v := MustParse("2.5")
	if !FromInt64(5).ScaleB(2).Equal(FromInt64(500)) || v.Quantize(MustParse("0.1")).String() != "2.5" ||
		v.Quantize(FromInt64(1)).String() != "2" || // to integer, ties to even
		!v.SameQuantum(MustParse("0.1")) || v.SameQuantum(FromInt64(1)) {
		t.Error("scale / quantum")
	}
}

func TestDecimal128WrapperMinMax(t *testing.T) {
	if !Minimum(FromInt64(2), FromInt64(5)).Equal(FromInt64(2)) || !Maximum(FromInt64(2), FromInt64(5)).Equal(FromInt64(5)) ||
		!MinimumMagnitude(FromInt64(-3), FromInt64(2)).Equal(FromInt64(2)) || !MaximumMagnitude(FromInt64(-3), FromInt64(2)).Equal(FromInt64(-3)) ||
		!Minimum(NaN(), FromInt64(5)).IsNaN() { // NaN propagates
		t.Error("min / max")
	}
}

func TestDecimal128WrapperFinerClassification(t *testing.T) {
	if !FromInt64(1).IsNormal() || FromInt64(0).IsNormal() || !FromInt64(1).IsCanonical() || FromInt64(1).IsSubnormal() ||
		!MustParse("1E-6176").IsSubnormal() {
		t.Error("finer classification")
	}
}

func TestDecimal128WrapperTotalOrderAndCopySign(t *testing.T) {
	if !FromInt64(1).IsTotallyOrdered(FromInt64(2)) || FromInt64(2).IsTotallyOrdered(FromInt64(1)) ||
		!FromInt64(1).IsTotallyOrdered(NaN()) { // NaN is greatest in total order
		t.Error("total order")
	}
	if !FromInt64(5).Copysign(FromInt64(-1)).Equal(FromInt64(-5)) || !FromInt64(-5).Copysign(FromInt64(1)).Equal(FromInt64(5)) {
		t.Error("copysign")
	}
}

func TestDecimal128WrapperThreeWayCompare(t *testing.T) {
	if FromInt64(1).ComparePartial(FromInt64(2)) != CmpLess || FromInt64(2).ComparePartial(FromInt64(2)) != CmpEqual ||
		FromInt64(3).ComparePartial(FromInt64(2)) != CmpGreater || NaN().ComparePartial(FromInt64(1)) != CmpUnordered ||
		MustParse("1.0").ComparePartial(MustParse("1.00")) != CmpEqual { // cohorts compare equal
		t.Error("three-way compare")
	}
}

func TestDecimal128WrapperCompareSignalsOnNaN(t *testing.T) {
	// Quiet compare: a *quiet* NaN does not signal; a signaling NaN does.
	q := NewContext()
	q.ComparePartial(NaN(), FromInt64(1))
	if !q.RaisedFlags().IsEmpty() {
		t.Error("quiet NaN signaled")
	}
	s := NewContext()
	s.ComparePartial(SignalingNaN(), FromInt64(1))
	if !s.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("sNaN did not signal")
	}
	// Signaling compare: even a quiet NaN signals.
	sig := NewContext()
	if sig.ComparePartialSignaling(NaN(), FromInt64(1)) != CmpUnordered || !sig.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("signaling compare")
	}
}

func TestDecimal128WrapperTotalOrderingThreeWay(t *testing.T) {
	if FromInt64(1).TotalOrder(FromInt64(2)) != CmpLess || FromInt64(2).TotalOrder(FromInt64(2)) != CmpEqual ||
		MustParse("1.0").TotalOrder(MustParse("1.00")) == CmpEqual || // distinct cohorts differ under total order
		FromInt64(-5).TotalOrderByMagnitude(FromInt64(5)) != CmpEqual { // magnitude order ignores sign
		t.Error("total ordering")
	}
}

func TestDecimal128WrapperIeeePredicates(t *testing.T) {
	if !FromInt64(1).Equal(FromInt64(1)) || !FromInt64(1).Less(FromInt64(2)) || !FromInt64(2).LessOrEqual(FromInt64(2)) ||
		!NaN().IsUnordered(FromInt64(1)) || FromInt64(1).IsUnordered(FromInt64(2)) ||
		NaN().Equal(NaN()) || NaN().Less(FromInt64(1)) {
		t.Error("IEEE predicates")
	}
}

func TestDecimal128WrapperIntegerDivisionAndRemainder(t *testing.T) {
	if !FromInt64(7).IntegerDivide(FromInt64(2)).Equal(FromInt64(3)) {
		t.Error("integer divide")
	}
	ctx := NewContext()
	if !ctx.RemainderNear(FromInt64(7), FromInt64(0)).IsNaN() || !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("remainder by zero")
	}
}

func TestDecimal128WrapperMinMaxNumberIgnoresNaN(t *testing.T) {
	if !MaximumNumber(NaN(), FromInt64(5)).Equal(FromInt64(5)) || !MinimumNumber(FromInt64(2), NaN()).Equal(FromInt64(2)) ||
		!Maximum(NaN(), FromInt64(5)).IsNaN() {
		t.Error("number forms")
	}
}

func TestDecimal128WrapperContextParseIEEE(t *testing.T) {
	ctx := NewContext()
	if ctx.ParseOrNaN("3.14").String() != "3.14" || !ctx.RaisedFlags().IsEmpty() {
		t.Error("valid parse")
	}
	// Over-precision literal (35 digits) is rounded → inexact.
	ctxRound := NewContext()
	if !ctxRound.ParseOrNaN("1.0000000000000000000000000000000005").IsFinite() || !ctxRound.RaisedFlags().Has(FlagInexact) {
		t.Error("over-precision")
	}
	// Malformed input → quiet NaN + invalidOperation (never an error/trap).
	ctxBad := NewContext()
	if !ctxBad.ParseOrNaN("not a number").IsNaN() || !ctxBad.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("malformed")
	}
	// Rounding direction governs the over-precision rounding.
	up := NewContextWithRounding(TowardPositive)
	if !strings.HasSuffix(up.ParseOrNaN("1.0000000000000000000000000000000001").String(), "1") {
		t.Error("directed parse")
	}
}

func TestDecimal128WrapperClassAndQuantumExponent(t *testing.T) {
	for v, want := range map[string]DecimalClass{
		"1": ClassPositiveNormal, "-1": ClassNegativeNormal, "0": ClassPositiveZero, "-0": ClassNegativeZero,
		"Infinity": ClassPositiveInfinity, "NaN": ClassQuietNaN, "sNaN": ClassSignalingNaN, "1E-6176": ClassPositiveSubnormal,
	} {
		if MustParse(v).DecimalClass() != want {
			t.Errorf("class of %s", v)
		}
	}
	// Quantum exponent = the value's decimal (stored) exponent.
	if MustParse("1.00").QuantumExponent() != -2 || MustParse("1E5").QuantumExponent() != 5 {
		t.Error("quantum exponent")
	}
	ctx := NewContext()
	ctx.QuantumExponent(NaN()) // NaN is an invalid quantum query
	if !ctx.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("NaN quantum")
	}
}

func TestDecimal128WrapperToInt64Conversions(t *testing.T) {
	threeSeven, twoFive := MustParse("3.7"), MustParse("2.5")
	// (a) Range guard: rounding-independent, range only (not integrality).
	if !threeSeven.FitsInt64() || !threeSeven.FitsUInt64() || !threeSeven.Neg().FitsInt64() || threeSeven.Neg().FitsUInt64() ||
		MustParse("1E100").FitsInt64() || NaN().FitsInt64() {
		t.Error("fits")
	}
	// (b) Fallible-exact: succeeds only for exact integers in range.
	if n, err := FromInt64(3).Int64(); err != nil || n != 3 {
		t.Error("Int64(3)")
	}
	if n, err := FromInt64(-3).Int64(); err != nil || n != -3 {
		t.Error("Int64(-3)")
	}
	if n, err := FromInt64(3).Uint64(); err != nil || n != 3 {
		t.Error("Uint64(3)")
	}
	for _, bad := range []func() error{
		func() error { _, err := threeSeven.Int64(); return err },         // fractional
		func() error { _, err := FromInt64(-3).Uint64(); return err },     // negative
		func() error { _, err := MustParse("1E100").Int64(); return err }, // out of range
		func() error { _, err := NaN().Int64(); return err },              // NaN
	} {
		if bad() == nil {
			t.Error("inexact conversion succeeded")
		}
	}
	// (c) Directed forms round a fraction (unchanged).
	if TiesToEven.ToInt64(threeSeven) != 4 || TiesToEven.ToInt64(twoFive) != 2 || TowardPositive.ToInt64(twoFive) != 3 {
		t.Error("directed forms")
	}
	// The signal-inexact variant records a discarded fraction…
	ctx := NewContext()
	ctx.ToInt64Exact(threeSeven)
	if !ctx.RaisedFlags().Has(FlagInexact) {
		t.Error("discarded fraction not inexact")
	}
	// …but an exact value raises nothing.
	ctxExact := NewContext()
	ctxExact.ToInt64Exact(FromInt64(5))
	if !ctxExact.RaisedFlags().IsEmpty() {
		t.Error("exact value raised flags")
	}
	// Out-of-range is an invalid conversion.
	ctxRange := NewContext()
	ctxRange.ToInt64(MustParse("1E100"))
	if !ctxRange.RaisedFlags().Has(FlagInvalidOperation) {
		t.Error("out of range")
	}
}

func TestDecimal128WrapperBidDpdInterchange(t *testing.T) {
	// Both encodings round-trip losslessly, preserving value AND cohort.
	for _, s := range []string{"3.14", "-1E+10", "0", "-0", "1.00", "Infinity", "-Infinity", "NaN"} {
		v := MustParse(s)
		raw := v.Text(StyleCoefficientExponent)
		if FromBID128(v.BID128()).Text(StyleCoefficientExponent) != raw || FromDPD128(v.DPD128()).Text(StyleCoefficientExponent) != raw {
			t.Errorf("round trip %s", s)
		}
	}
	// Cohort survives the round trip (1.00 stays 1.00, not 1).
	if FromDPD128(MustParse("1.00").DPD128()).String() != "1.00" {
		t.Error("cohort")
	}
	// The two encodings of the same value decode equal.
	if !FromBID128(FromInt64(42).BID128()).Equal(FromInt64(42)) {
		t.Error("bid 42")
	}
}

func TestDecimal128WrapperStripTrailingZerosAndLogB(t *testing.T) {
	if MustParse("1.200").StripTrailingZeros().String() != "1.2" || !MustParse("1000").Logb().Equal(FromInt64(3)) {
		t.Error("strip / logB")
	}
	ctx := NewContext()
	r := ctx.Logb(FromInt64(0))
	if !r.IsInf() || !r.Signbit() || !ctx.RaisedFlags().Has(FlagDivisionByZero) { // logB(0) = -∞
		t.Error("logB(0)")
	}
}
