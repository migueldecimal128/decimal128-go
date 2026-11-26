// Context-aware IEEE 754-2019 Section 5.3.1 min/max. The quiet forms (d128_minimum,
// d128_maximum, ...) raise no flags. The _ctx forms add flag observability: they check
// the inputs before dispatching to the quiet form and record invalidOperation on ctx
// when either operand is a signaling NaN, then return the quiet form's result (a
// quieted NaN per IEEE).

package decimal128

func d128_minimum_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_minimum(x, y)
}

func d128_maximum_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_maximum(x, y)
}

func d128_minimumNumber_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_minimumNumber(x, y)
}

func d128_maximumNumber_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_maximumNumber(x, y)
}

func d128_minimumMagnitude_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_minimumMagnitude(x, y)
}

func d128_maximumMagnitude_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_maximumMagnitude(x, y)
}

func d128_minimumMagnitudeNumber_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_minimumMagnitudeNumber(x, y)
}

func d128_maximumMagnitudeNumber_ctx(x, y d128, ctx *context) d128 {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
	return d128_maximumMagnitudeNumber(x, y)
}
