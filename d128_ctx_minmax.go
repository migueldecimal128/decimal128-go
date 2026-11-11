// Context-aware IEEE 754-2019 §5.3.1 min/max operations.
//
// The quiet free functions (d128_minimum, d128_maximum, ...) are value-only.
// These _ctx forms add flag observability: the invalid-operation check is
// performed on the inputs before dispatching to the quiet function — if either
// operand is a signaling NaN, invalidOperation is recorded on ctx. The numeric
// result is then whatever the quiet function returns (a quieted NaN per IEEE).

package decimal128

func minMaxSignalSNaN(x, y d128, ctx *context) {
	if d128_isSNAN(x) || d128_isSNAN(y) {
		ctx.flags.set(const_INVALID_OPERATION)
	}
}

func d128_minimum_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_minimum(x, y)
}

func d128_maximum_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_maximum(x, y)
}

func d128_minimumNumber_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_minimumNumber(x, y)
}

func d128_maximumNumber_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_maximumNumber(x, y)
}

func d128_minimumMagnitude_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_minimumMagnitude(x, y)
}

func d128_maximumMagnitude_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_maximumMagnitude(x, y)
}

func d128_minimumMagnitudeNumber_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_minimumMagnitudeNumber(x, y)
}

func d128_maximumMagnitudeNumber_ctx(x, y d128, ctx *context) d128 {
	minMaxSignalSNaN(x, y, ctx)
	return d128_maximumMagnitudeNumber(x, y)
}
