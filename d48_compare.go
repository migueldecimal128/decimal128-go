// Magnitude comparison over D48: -1 / 0 / +1 (Swift D48Compare.swift).
//
// One deliberate difference from d38_cmpMag (D48ZigPrototypeWorkOrder.md Phase
// 1): D38 aligns by multiplying the larger-exponent coefficient up, which needs
// the exponent gap to fit the wide type. D48 first compares scientific
// exponents (exact for nonzero values), so any gap is safe; alignment then
// needs at most 47 extra digits.

package decimal128

func d48_cmpMag(x, y d48) int {
	xZero := d48_isZero(x)
	yZero := d48_isZero(y)
	if xZero || yZero {
		r := 0
		if !xZero {
			r++
		}
		if !yZero {
			r--
		}
		return r
	}
	xs := d48_sciExp(x)
	ys := d48_sciExp(y)
	if xs != ys {
		if xs > ys {
			return 1
		}
		return -1
	}
	// Same leading-digit position: |qDelta| = |digitLen difference| <= 47.
	qDelta := d48_qExp(x) - d48_qExp(y)
	if qDelta >= 0 {
		return cmp_int_320x320(mulPow10_320(d48_u320(x), qDelta), d48_u320(y))
	}
	return cmp_int_320x320(d48_u320(x), mulPow10_320(d48_u320(y), -qDelta))
}
