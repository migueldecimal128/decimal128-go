// Financial functions built on the decimal128 Core ops. They live in Core (not the
// wrapper) so every platform shares one byte-identical implementation. These are
// compositions of the IEEE primitives, not IEEE operations themselves, so they expose
// no rounding/flag ladder -- a single ties-to-even (_tte) form each. compound(x, n) is
// (1+x)^n (d128_compound); rootn is the integer root.

package decimal128

var const_FIN_ONE = d128_fromInt64(1)
var const_FIN_ZERO = d128_zero(false, 0)
var const_FIN_HALF_CENT = decimalFNZ_64x64(false, -3, 5, 0) // 0.005

// --- simple & compound interest ---

// Simple interest earned over `periods` periods: principal * rate * periods.
func d128_simpleInterest_tte(principal, rate, periods d128) d128 {
	return d128_multiply_tte(d128_multiply_tte(principal, rate), periods)
}

// Future value with compound interest: principal * (1 + rate)^numPeriods.
func d128_compoundInterest_tte(principal, rate d128, numPeriods int) d128 {
	return d128_multiply_tte(principal, d128_compound_tte(rate, numPeriods))
}

// Effective annual rate: (1 + nominalRate / timesPerYear)^timesPerYear - 1.
func d128_effectiveAnnualRate_tte(nominalRate d128, timesPerYear int) d128 {
	perPeriod := d128_divide_tte(nominalRate, d128_fromInt64(timesPerYear))
	return d128_subtract_tte(d128_compound_tte(perPeriod, timesPerYear), const_FIN_ONE)
}

// --- mortgage / annuity payment ---

// Fixed-rate mortgage / annuity payment: PV * r * (1+r)^n / ((1+r)^n - 1).
// r = 0 -> PV / n (interest-free loan).
func d128_mortgagePayment_tte(principal, periodicRate d128, numPayments int) d128 {
	if d128_isZER(periodicRate) {
		return d128_divide_tte(principal, d128_fromInt64(numPayments))
	}
	factor := d128_compound_tte(periodicRate, numPayments) // (1+r)^n
	num := d128_multiply_tte(d128_multiply_tte(principal, periodicRate), factor)
	denom := d128_subtract_tte(factor, const_FIN_ONE)
	return d128_divide_tte(num, denom)
}

// amortizationRow is one row in an amortization schedule (core form).
type amortizationRow struct {
	period        int
	payment       d128
	interest      d128
	principalPaid d128
	balance       d128
}

// Full amortization schedule -- one `AmortizationRow` per payment period. The
// final balance is clamped to exactly zero when its magnitude is below half a
// cent (fixed-payment rounding artifact).
func d128_amortizationSchedule_tte(principal, periodicRate d128, numPayments int) []amortizationRow {
	demand("amortizationSchedule: numPayments exceeds 100000", numPayments <= 100000)
	payment := d128_mortgagePayment_tte(principal, periodicRate, numPayments)
	rows := make([]amortizationRow, 0, max(0, numPayments))
	balance := principal
	p := 1
	for p <= numPayments {
		interest := d128_multiply_tte(balance, periodicRate)
		principalPaid := d128_subtract_tte(payment, interest)
		balance = d128_subtract_tte(balance, principalPaid)
		if p == numPayments && d128_isLess(d128_abs(balance), const_FIN_HALF_CENT) {
			balance = const_FIN_ZERO
		}
		rows = append(rows, amortizationRow{p, payment, interest, principalPaid, balance})
		p += 1
	}
	return rows
}

// --- annuity present / future value ---

// Present value of an ordinary annuity: PMT * (1 - (1+r)^-n) / r. r = 0 -> PMT * n.
func d128_presentValueAnnuity_tte(payment, periodicRate d128, numPeriods int) d128 {
	if d128_isZER(periodicRate) {
		return d128_multiply_tte(payment, d128_fromInt64(numPeriods))
	}
	discount := d128_divide_tte(const_FIN_ONE, d128_compound_tte(periodicRate, numPeriods))
	oneMinus := d128_subtract_tte(const_FIN_ONE, discount)
	return d128_divide_tte(d128_multiply_tte(payment, oneMinus), periodicRate)
}

// Future value of an ordinary annuity: PMT * ((1+r)^n - 1) / r. r = 0 -> PMT * n.
func d128_futureValueAnnuity_tte(payment, periodicRate d128, numPeriods int) d128 {
	if d128_isZER(periodicRate) {
		return d128_multiply_tte(payment, d128_fromInt64(numPeriods))
	}
	growth := d128_subtract_tte(d128_compound_tte(periodicRate, numPeriods), const_FIN_ONE)
	return d128_divide_tte(d128_multiply_tte(payment, growth), periodicRate)
}

// --- single cash-flow helpers ---

// Present value of a single future cash flow: FV / (1+r)^n.
func d128_presentValue_tte(futureValue, rate d128, numPeriods int) d128 {
	return d128_divide_tte(futureValue, d128_compound_tte(rate, numPeriods))
}

// Future value of a single present cash flow: PV * (1+r)^n.
func d128_futureValue_tte(presentValue, rate d128, numPeriods int) d128 {
	return d128_multiply_tte(presentValue, d128_compound_tte(rate, numPeriods))
}

// Compound annual growth rate: (FV/PV)^(1/n) - 1. PV and FV must be positive.
func d128_cagr_tte(presentValue, futureValue d128, numPeriods int) d128 {
	verify("cagr: presentValue must be positive", func() bool { return d128_isGreater(presentValue, const_FIN_ZERO) })
	verify("cagr: futureValue must be positive", func() bool { return d128_isGreater(futureValue, const_FIN_ZERO) })
	ratio := d128_divide_tte(futureValue, presentValue)
	return d128_subtract_tte(d128_rootn_tte(ratio, numPeriods), const_FIN_ONE)
}

// --- NPV / IRR / MIRR ---

// Net present value of `cashFlows` discounted at `rate`. `cashFlows[0]` is the
// immediate (t = 0, undiscounted) flow. NPV = sum CF_t / (1+rate)^t.
func d128_npv_tte(rate d128, cashFlows []d128) d128 {
	result := const_FIN_ZERO
	t := 0
	for t < len(cashFlows) {
		factor := d128_compound_tte(rate, t) // (1+rate)^t
		result = d128_add_tte(result, d128_divide_tte(cashFlows[t], factor))
		t += 1
	}
	return result
}

// Internal rate of return -- the rate making NPV(cashFlows) = 0, via
// Newton-Raphson with the analytic derivative. Returns NaN if the derivative
// vanishes or it fails to converge within `maxIter`. Needs >= 2 cash flows.
func d128_irr_tte(cashFlows []d128, guess, tolerance d128, maxIter int) d128 {
	verify("irr: need at least 2 cash flows", func() bool { return len(cashFlows) >= 2 })
	rate := guess
	iter := 0
	for iter < maxIter {
		npvVal := const_FIN_ZERO
		npvDeriv := const_FIN_ZERO
		divisor := const_FIN_ONE
		onePlusR := d128_add_tte(const_FIN_ONE, rate)
		t := 0
		for t < len(cashFlows) {
			cf := cashFlows[t]
			npvVal = d128_add_tte(npvVal, d128_divide_tte(cf, divisor))
			if t > 0 {
				// derivative term: −t · CF_t / (divisor · (1+r))
				cfT := d128_multiply_tte(cf, d128_fromInt64(t))
				denom := d128_multiply_tte(divisor, onePlusR)
				npvDeriv = d128_subtract_tte(npvDeriv, d128_divide_tte(cfT, denom))
			}
			divisor = d128_multiply_tte(divisor, onePlusR)
			t += 1
		}
		if d128_isZER(npvDeriv) {
			return const_D128_QNAN0
		}
		step := d128_divide_tte(npvVal, npvDeriv)
		rate = d128_subtract_tte(rate, step)
		if d128_isLess(d128_abs(step), tolerance) {
			return rate
		}
		iter += 1
	}
	return const_D128_QNAN0 // did not converge
}

// Modified IRR -- positive flows reinvested at `reinvestRate`, negative flows
// discounted at `financeRate`: (FV_pos / |PV_neg|)^(1/(n-1)) - 1. Needs at
// least one negative cash flow.
func d128_mirr_tte(cashFlows []d128, financeRate, reinvestRate d128) d128 {
	n := len(cashFlows)
	pvNeg := const_FIN_ZERO
	fvPos := const_FIN_ZERO
	t := 0
	for t < n {
		cf := cashFlows[t]
		if d128_isLess(cf, const_FIN_ZERO) {
			pvNeg = d128_add_tte(pvNeg, d128_divide_tte(cf, d128_compound_tte(financeRate, t)))
		} else if d128_isGreater(cf, const_FIN_ZERO) {
			fvPos = d128_add_tte(fvPos, d128_multiply_tte(cf, d128_compound_tte(reinvestRate, n-1-t)))
		}
		t += 1
	}
	verify("mirr: no negative cash flows found", func() bool { return !d128_isZER(pvNeg) })
	ratio := d128_divide_tte(fvPos, d128_negate(pvNeg))
	return d128_subtract_tte(d128_rootn_tte(ratio, n-1), const_FIN_ONE)
}
