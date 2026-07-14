// Padé rational evaluation in the D38 domain, shared by the exp / exp10 / ln /
// log10 kernels. Evaluates P(z)/Q(z) by fused Horner accumulation of the
// numerator and denominator weight tables, then a single D38 divide.

package decimal128

func d38_padeEval(zArg d38, p, q []d38) d38 {
	pAcc := p[len(p)-1]
	for i := len(p) - 2; i >= 0; i-- {
		pAcc = d38_fma(pAcc, zArg, p[i])
	}

	qAcc := q[len(q)-1]
	for i := len(q) - 2; i >= 0; i-- {
		qAcc = d38_fma(qAcc, zArg, q[i])
	}

	return d38_divide(pAcc, qAcc)
}
