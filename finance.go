// Public financial functions over Decimal128, forwarding to the core
// BasicFinance implementation (one byte-identical algorithm per platform). All
// compute at ties-to-even, quiet.
//
// They are grouped under the package-level Finance namespace value, so callers
// write decimal128.Finance.SimpleInterest(...), mirroring the sibling ports.

package decimal128

// AmortizationRow is one row in an amortization schedule.
type AmortizationRow struct {
	Period        int
	Payment       Decimal128
	Interest      Decimal128
	PrincipalPaid Decimal128
	Balance       Decimal128
}

// Finance groups the financial calculations. Use it as a namespace, e.g.
// decimal128.Finance.SimpleInterest(p, r, n).
var Finance financeAPI

type financeAPI struct{}

// SimpleInterest returns the simple interest earned over periods periods:
// principal × rate × periods.
func (financeAPI) SimpleInterest(principal, rate, periods Decimal128) Decimal128 {
	return Decimal128{d128_simpleInterest_tte(principal.core, rate.core, periods.core)}
}

// CompoundInterest returns the future value with compound interest:
// principal × (1 + rate)^periods.
func (financeAPI) CompoundInterest(principal, rate Decimal128, numPeriods int) Decimal128 {
	return Decimal128{d128_compoundInterest_tte(principal.core, rate.core, numPeriods)}
}

// EffectiveAnnualRate returns (1 + nominalRate / timesPerYear)^timesPerYear − 1.
func (financeAPI) EffectiveAnnualRate(nominalRate Decimal128, timesPerYear int) Decimal128 {
	return Decimal128{d128_effectiveAnnualRate_tte(nominalRate.core, timesPerYear)}
}

// MortgagePayment returns the fixed-rate mortgage / annuity payment.
// periodicRate = 0 → principal / numPayments.
func (financeAPI) MortgagePayment(principal, periodicRate Decimal128, numPayments int) Decimal128 {
	return Decimal128{d128_mortgagePayment_tte(principal.core, periodicRate.core, numPayments)}
}

// AmortizationSchedule returns the full amortization schedule — one
// AmortizationRow per payment period.
func (financeAPI) AmortizationSchedule(principal, periodicRate Decimal128, numPayments int) []AmortizationRow {
	core := d128_amortizationSchedule_tte(principal.core, periodicRate.core, numPayments)
	rows := make([]AmortizationRow, len(core))
	for i, r := range core {
		rows[i] = AmortizationRow{
			Period:        r.period,
			Payment:       Decimal128{r.payment},
			Interest:      Decimal128{r.interest},
			PrincipalPaid: Decimal128{r.principalPaid},
			Balance:       Decimal128{r.balance},
		}
	}
	return rows
}

// PresentValueAnnuity returns the present value of an ordinary annuity.
// periodicRate = 0 → payment × numPeriods.
func (financeAPI) PresentValueAnnuity(payment, periodicRate Decimal128, numPeriods int) Decimal128 {
	return Decimal128{d128_presentValueAnnuity_tte(payment.core, periodicRate.core, numPeriods)}
}

// FutureValueAnnuity returns the future value of an ordinary annuity.
// periodicRate = 0 → payment × numPeriods.
func (financeAPI) FutureValueAnnuity(payment, periodicRate Decimal128, numPeriods int) Decimal128 {
	return Decimal128{d128_futureValueAnnuity_tte(payment.core, periodicRate.core, numPeriods)}
}

// PresentValue returns the present value of a single future cash flow:
// futureValue / (1+rate)^numPeriods.
func (financeAPI) PresentValue(futureValue, rate Decimal128, numPeriods int) Decimal128 {
	return Decimal128{d128_presentValue_tte(futureValue.core, rate.core, numPeriods)}
}

// FutureValue returns the future value of a single present cash flow:
// presentValue × (1+rate)^numPeriods.
func (financeAPI) FutureValue(presentValue, rate Decimal128, numPeriods int) Decimal128 {
	return Decimal128{d128_futureValue_tte(presentValue.core, rate.core, numPeriods)}
}

// CAGR returns the compound annual growth rate: (futureValue/presentValue)^(1/numPeriods) − 1.
// presentValue and futureValue must be positive.
func (financeAPI) CAGR(presentValue, futureValue Decimal128, numPeriods int) Decimal128 {
	return Decimal128{d128_cagr_tte(presentValue.core, futureValue.core, numPeriods)}
}

// NPV returns the net present value of cashFlows discounted at rate.
// cashFlows[0] is the immediate (t = 0, undiscounted) flow.
func (financeAPI) NPV(rate Decimal128, cashFlows []Decimal128) Decimal128 {
	return Decimal128{d128_npv_tte(rate.core, unwrapSlice(cashFlows))}
}

// IRR returns the internal rate of return (the rate making NPV = 0) using the
// default guess 0.1, tolerance 1e-10, and 200 iterations. ok is false if it does
// not converge. Needs ≥ 2 cash flows.
func (f financeAPI) IRR(cashFlows []Decimal128) (result Decimal128, ok bool) {
	return f.IRRWithParams(cashFlows, MustParse("0.1"), MustParse("1e-10"), 200)
}

// IRRWithParams is IRR with an explicit initial guess, convergence tolerance,
// and iteration cap. ok is false if it does not converge.
func (financeAPI) IRRWithParams(cashFlows []Decimal128, guess, tolerance Decimal128, maxIter int) (result Decimal128, ok bool) {
	r := Decimal128{d128_irr_tte(unwrapSlice(cashFlows), guess.core, tolerance.core, maxIter)}
	if r.IsNaN() {
		return Decimal128{}, false
	}
	return r, true
}

// MIRR returns the modified internal rate of return — positive flows reinvested
// at reinvestRate, negative flows discounted at financeRate. Needs at least one
// negative cash flow.
func (financeAPI) MIRR(cashFlows []Decimal128, financeRate, reinvestRate Decimal128) Decimal128 {
	return Decimal128{d128_mirr_tte(unwrapSlice(cashFlows), financeRate.core, reinvestRate.core)}
}

// unwrapSlice converts a slice of public values to their core engine values.
func unwrapSlice(xs []Decimal128) []d128 {
	cs := make([]d128, len(xs))
	for i, x := range xs {
		cs[i] = x.core
	}
	return cs
}
