package decimal128

import "testing"

// Ported from the Swift TestFinanceWrapper: exercises the public Finance API
// with the same expected values and tolerances.

func financeCheck(t *testing.T, label string, expected, actual Decimal128, tol Decimal128) {
	t.Helper()
	diff := actual.Sub(expected).Abs()
	if !diff.LessOrEqual(tol) {
		t.Errorf("%s: expected %s, got %s (diff %s)", label, expected, actual, diff)
	}
}

func TestFinance(t *testing.T) {
	d := MustParse
	tol := MustParse("1e-15")
	flows := []Decimal128{d("-1000"), d("400"), d("500"), d("400")}

	financeCheck(t, "simpleInterest", d("150"),
		Finance.SimpleInterest(d("1000"), d("0.05"), d("3")), tol)
	financeCheck(t, "compoundInterest", d("1628.894626777441406250"),
		Finance.CompoundInterest(d("1000"), d("0.05"), 10), tol)
	financeCheck(t, "effectiveAnnualRate", d("0.12682503013196972066"),
		Finance.EffectiveAnnualRate(d("0.12"), 12), tol)

	financeCheck(t, "mortgagePayment", d("1199.10105030550478918292"),
		Finance.MortgagePayment(d("200000"), d("0.005"), 360), tol)
	financeCheck(t, "mortgagePayment zero-rate", d("1000"),
		Finance.MortgagePayment(d("12000"), d("0"), 12), tol)

	sched := Finance.AmortizationSchedule(d("200000"), d("0.005"), 360)
	financeCheck(t, "amort row1 interest", d("1000"), sched[0].Interest, d("1e-10"))
	financeCheck(t, "amort final balance", d("0"), sched[len(sched)-1].Balance, d("0.01"))
	if sched[0].Period != 1 || len(sched) != 360 {
		t.Errorf("amort schedule shape: period0=%d len=%d", sched[0].Period, len(sched))
	}

	financeCheck(t, "presentValueAnnuity", d("10621.6936288139243718"),
		Finance.PresentValueAnnuity(d("500"), d("0.01"), 24), tol)
	financeCheck(t, "futureValueAnnuity", d("7867.2209929367732338"),
		Finance.FutureValueAnnuity(d("200"), d("0.005"), 36), tol)

	financeCheck(t, "npv r=0.10", d("77.38542449286250939143"),
		Finance.NPV(d("0.10"), flows), tol)
	financeCheck(t, "npv r=0.20", d("-87.96296296296296296296"),
		Finance.NPV(d("0.20"), flows), tol)

	irr, ok := Finance.IRR(flows)
	if !ok {
		t.Fatal("IRR did not converge")
	}
	financeCheck(t, "irr", d("0.14332259275356270085"), irr, tol)
	financeCheck(t, "npv at irr", d("0"), Finance.NPV(irr, flows), d("1e-8"))

	financeCheck(t, "mirr", d("0.13490286191516542854"),
		Finance.MIRR(flows, d("0.10"), d("0.12")), tol)

	financeCheck(t, "presentValue", d("680.58319703375316322003"),
		Finance.PresentValue(d("1000"), d("0.08"), 5), tol)
	financeCheck(t, "futureValue", d("751.8151294956800000"),
		Finance.FutureValue(d("500"), d("0.06"), 7), tol)
	financeCheck(t, "cagr", d("0.07177346253629316421"),
		Finance.CAGR(d("1000"), d("2000"), 10), tol)
}
