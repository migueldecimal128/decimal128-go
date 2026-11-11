// Conservative lower/upper bounds on the binary exponent of a finite decimal
// value x = coeff·10^qExp, via fixed-point qExp·log2(10). Branch-free; the
// returned bounds always bracket the true binary exponent.

package decimal128

// log2(10) ≈ const / 2^32 (FLOOR <= true <= CEIL).
const (
	const_LOG2_10_FLOOR = 14_267_572_564
	const_LOG2_10_CEIL  = 14_267_572_565
)

func calcBExpMin(bitLen, qExp int) int {
	bitLenNonZeroMask := (0 - bitLen) >> 63
	bitLenLess1 := bitLen - 1
	scaledFloor := (qExp * const_LOG2_10_FLOOR) >> 32
	scaledCeil := (qExp * const_LOG2_10_CEIL) >> 32
	bExpMinWhenPos := (bitLenLess1 + scaledFloor) & bitLenNonZeroMask
	bExpMinWhenNeg := (bitLenLess1 + scaledCeil) & bitLenNonZeroMask
	signMask := qExp >> 63
	return (bExpMinWhenNeg & signMask) | (bExpMinWhenPos &^ signMask)
}

func calcBExpMax(bitLen, qExp int) int {
	bitLenNonZeroMask := (0 - bitLen) >> 63
	scaledNegCeil := (qExp * (0 - const_LOG2_10_CEIL)) >> 32
	scaledNegFloor := (qExp * (0 - const_LOG2_10_FLOOR)) >> 32
	bExpMaxWhenPos := (bitLen - scaledNegCeil - 1) & bitLenNonZeroMask
	bExpMaxWhenNeg := (bitLen - scaledNegFloor - 1) & bitLenNonZeroMask
	signMask := qExp >> 63
	return (bExpMaxWhenNeg & signMask) | (bExpMaxWhenPos &^ signMask)
}
