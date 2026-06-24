// Financial functions built on the decimal128 core ops. They live in the core
// (not just the wrapper) so every platform shares one byte-identical
// implementation. These are compositions of the IEEE primitives, not IEEE
// operations themselves, so they expose no rounding/flag ladder — a single
// ties-to-even (_tte) form each. compound(x, n) is (1+x)^n.

package decimal128

var finOne = d128_fromInt(1)
var finZero = d128_zero(false, 0)
var finHalfCent = decimalFNZ_64x64(false, -3, 5, 0) // 0.005

// --- simple & compound interest ---

// d128_simpleInterest_tte: principal × rate × periods.
func d128_simpleInterest_tte(principal, rate, periods d128) d128 {
	return d128_multiply_tte(d128_multiply_tte(principal, rate), periods)
}

// d128_compoundInterest_tte: principal × (1 + rate)^numPeriods.
func d128_compoundInterest_tte(principal, rate d128, numPeriods int) d128 {
	return d128_multiply_tte(principal, d128_compound_tte(rate, numPeriods))
}

// d128_effectiveAnnualRate_tte: (1 + nominalRate / timesPerYear)^timesPerYear − 1.
func d128_effectiveAnnualRate_tte(nominalRate d128, timesPerYear int) d128 {
	perPeriod := d128_divide_tte(nominalRate, d128_fromInt(timesPerYear))
	return d128_subtract_tte(d128_compound_tte(perPeriod, timesPerYear), finOne)
}

// --- mortgage / annuity payment ---

// d128_mortgagePayment_tte: PV × r × (1+r)^n / ((1+r)^n − 1). r = 0 → PV / n.
func d128_mortgagePayment_tte(principal, periodicRate d128, numPayments int) d128 {
	if d128_isZER(periodicRate) {
		return d128_divide_tte(principal, d128_fromInt(numPayments))
	}
	factor := d128_compound_tte(periodicRate, numPayments) // (1+r)^n
	num := d128_multiply_tte(d128_multiply_tte(principal, periodicRate), factor)
	denom := d128_subtract_tte(factor, finOne)
	return d128_divide_tte(num, denom)
}

// amortizationRowCore is one row in an amortization schedule (core form).
type amortizationRowCore struct {
	period        int
	payment       d128
	interest      d128
	principalPaid d128
	balance       d128
}

// d128_amortizationSchedule_tte returns one row per payment period. The final
// balance is clamped to exactly zero when its magnitude is below half a cent (a
// fixed-payment rounding artifact).
func d128_amortizationSchedule_tte(principal, periodicRate d128, numPayments int) []amortizationRowCore {
	payment := d128_mortgagePayment_tte(principal, periodicRate, numPayments)
	rows := make([]amortizationRowCore, 0, numPayments)
	balance := principal
	for p := 1; p <= numPayments; p++ {
		interest := d128_multiply_tte(balance, periodicRate)
		principalPaid := d128_subtract_tte(payment, interest)
		balance = d128_subtract_tte(balance, principalPaid)
		if p == numPayments && d128_isLess(d128_abs(balance), finHalfCent) {
			balance = finZero
		}
		rows = append(rows, amortizationRowCore{p, payment, interest, principalPaid, balance})
	}
	return rows
}

// --- annuity present / future value ---

// d128_presentValueAnnuity_tte: PMT × (1 − (1+r)^−n) / r. r = 0 → PMT × n.
func d128_presentValueAnnuity_tte(payment, periodicRate d128, numPeriods int) d128 {
	if d128_isZER(periodicRate) {
		return d128_multiply_tte(payment, d128_fromInt(numPeriods))
	}
	discount := d128_divide_tte(finOne, d128_compound_tte(periodicRate, numPeriods))
	oneMinus := d128_subtract_tte(finOne, discount)
	return d128_divide_tte(d128_multiply_tte(payment, oneMinus), periodicRate)
}

// d128_futureValueAnnuity_tte: PMT × ((1+r)^n − 1) / r. r = 0 → PMT × n.
func d128_futureValueAnnuity_tte(payment, periodicRate d128, numPeriods int) d128 {
	if d128_isZER(periodicRate) {
		return d128_multiply_tte(payment, d128_fromInt(numPeriods))
	}
	growth := d128_subtract_tte(d128_compound_tte(periodicRate, numPeriods), finOne)
	return d128_divide_tte(d128_multiply_tte(payment, growth), periodicRate)
}

// --- single cash-flow helpers ---

// d128_presentValue_tte: FV / (1+r)^n.
func d128_presentValue_tte(futureValue, rate d128, numPeriods int) d128 {
	return d128_divide_tte(futureValue, d128_compound_tte(rate, numPeriods))
}

// d128_futureValue_tte: PV × (1+r)^n.
func d128_futureValue_tte(presentValue, rate d128, numPeriods int) d128 {
	return d128_multiply_tte(presentValue, d128_compound_tte(rate, numPeriods))
}

// d128_cagr_tte: (FV/PV)^(1/n) − 1. PV and FV must be positive.
func d128_cagr_tte(presentValue, futureValue d128, numPeriods int) d128 {
	verify("cagr: presentValue must be positive", d128_isGreater(presentValue, finZero))
	verify("cagr: futureValue must be positive", d128_isGreater(futureValue, finZero))
	ratio := d128_divide_tte(futureValue, presentValue)
	return d128_subtract_tte(d128_rootn_tte(ratio, numPeriods), finOne)
}

// --- NPV / IRR / MIRR ---

// d128_npv_tte: Σ CF_t / (1+rate)^t. cashFlows[0] is the immediate (t=0) flow.
func d128_npv_tte(rate d128, cashFlows []d128) d128 {
	result := finZero
	for t := 0; t < len(cashFlows); t++ {
		factor := d128_compound_tte(rate, t) // (1+rate)^t
		result = d128_add_tte(result, d128_divide_tte(cashFlows[t], factor))
	}
	return result
}

// d128_irr_tte: the rate making NPV(cashFlows) = 0, via Newton-Raphson with the
// analytic derivative. Returns NaN if the derivative vanishes or it fails to
// converge within maxIter. Needs ≥ 2 cash flows.
func d128_irr_tte(cashFlows []d128, guess, tolerance d128, maxIter int) d128 {
	verify("irr: need at least 2 cash flows", len(cashFlows) >= 2)
	rate := guess
	for iter := 0; iter < maxIter; iter++ {
		npvVal := finZero
		npvDeriv := finZero
		divisor := finOne
		onePlusR := d128_add_tte(finOne, rate)
		for t := 0; t < len(cashFlows); t++ {
			cf := cashFlows[t]
			npvVal = d128_add_tte(npvVal, d128_divide_tte(cf, divisor))
			if t > 0 {
				// derivative term: −t · CF_t / (divisor · (1+r))
				cfT := d128_multiply_tte(cf, d128_fromInt(t))
				denom := d128_multiply_tte(divisor, onePlusR)
				npvDeriv = d128_subtract_tte(npvDeriv, d128_divide_tte(cfT, denom))
			}
			divisor = d128_multiply_tte(divisor, onePlusR)
		}
		if d128_isZER(npvDeriv) {
			return D128_QNAN0
		}
		step := d128_divide_tte(npvVal, npvDeriv)
		rate = d128_subtract_tte(rate, step)
		if d128_isLess(d128_abs(step), tolerance) {
			return rate
		}
	}
	return D128_QNAN0 // did not converge
}

// d128_mirr_tte: (FV_pos / |PV_neg|)^(1/(n−1)) − 1 — positive flows reinvested at
// reinvestRate, negative flows discounted at financeRate. Needs ≥ 1 negative flow.
func d128_mirr_tte(cashFlows []d128, financeRate, reinvestRate d128) d128 {
	n := len(cashFlows)
	pvNeg := finZero
	fvPos := finZero
	for t := 0; t < n; t++ {
		cf := cashFlows[t]
		if d128_isLess(cf, finZero) {
			pvNeg = d128_add_tte(pvNeg, d128_divide_tte(cf, d128_compound_tte(financeRate, t)))
		} else if d128_isGreater(cf, finZero) {
			fvPos = d128_add_tte(fvPos, d128_multiply_tte(cf, d128_compound_tte(reinvestRate, n-1-t)))
		}
	}
	verify("mirr: no negative cash flows found", !d128_isZER(pvNeg))
	ratio := d128_divide_tte(fvPos, d128_negate(pvNeg))
	return d128_subtract_tte(d128_rootn_tte(ratio, n-1), finOne)
}
