package decimal128

// Ported from the Kotlin-legacy TestFinance. Tolerance-based (the legacy
// compared |actual − expected| ≤ tol); the Core finance ops are byte-identical
// compositions of d128 arithmetic, so the results match the legacy values.

import "testing"

func TestFinanceFinance(t *testing.T) {
	dec := func(s string) d128 { return mustDec(t, s) }
	check := func(label, expected string, actual d128, tol string) {
		t.Helper()
		diff := d128_abs(d128_subtract_tte(actual, dec(expected)))
		if !d128_isLessEqual(diff, dec(tol)) {
			t.Errorf("%s: expected %s, got %s", label, expected, d128_format(actual, const_FORMAT_AUTO))
		}
	}
	const tol = "1e-15"
	flows := []d128{dec("-1000"), dec("400"), dec("500"), dec("400")}
	// simple & compound interest
	check("simpleInterest", "150", d128_simpleInterest_tte(dec("1000"), dec("0.05"), dec("3")), tol)
	check("compoundInterest", "1628.894626777441406250", d128_compoundInterest_tte(dec("1000"), dec("0.05"), 10), tol)
	check("effectiveAnnualRate", "0.12682503013196972066", d128_effectiveAnnualRate_tte(dec("0.12"), 12), tol)
	// mortgage payment
	check("mortgagePayment", "1199.10105030550478918292", d128_mortgagePayment_tte(dec("200000"), dec("0.005"), 360), tol)
	check("mortgagePayment zero-rate", "1000", d128_mortgagePayment_tte(dec("12000"), dec("0"), 12), tol)
	// amortization schedule
	sched := d128_amortizationSchedule_tte(dec("200000"), dec("0.005"), 360)
	check("amort row1 interest", "1000", sched[0].interest, "1e-10")
	check("amort final balance", "0", sched[len(sched)-1].balance, "0.01")
	// annuity PV / FV
	check("presentValueAnnuity", "10621.6936288139243718", d128_presentValueAnnuity_tte(dec("500"), dec("0.01"), 24), tol)
	check("futureValueAnnuity", "7867.2209929367732338", d128_futureValueAnnuity_tte(dec("200"), dec("0.005"), 36), tol)
	// NPV
	check("npv r=0.10", "77.38542449286250939143", d128_npv_tte(dec("0.10"), flows), tol)
	check("npv r=0.20", "-87.96296296296296296296", d128_npv_tte(dec("0.20"), flows), tol)
	// IRR + NPV-at-IRR ≈ 0
	irr := d128_irr_tte(flows, dec("0.1"), dec("1e-10"), 200)
	if d128_isNAN(irr) {
		t.Fatal("IRR did not converge")
	}
	check("irr", "0.14332259275356270085", irr, tol)
	check("npv at irr", "0", d128_npv_tte(irr, flows), "1e-8")
	// MIRR
	check("mirr", "0.13490286191516542854", d128_mirr_tte(flows, dec("0.10"), dec("0.12")), tol)
	// single PV / FV / CAGR
	check("presentValue", "680.58319703375316322003", d128_presentValue_tte(dec("1000"), dec("0.08"), 5), tol)
	check("futureValue", "751.8151294956800000", d128_futureValue_tte(dec("500"), dec("0.06"), 7), tol)
	check("cagr", "0.07177346253629316421", d128_cagr_tte(dec("1000"), dec("2000"), 10), tol)
}
