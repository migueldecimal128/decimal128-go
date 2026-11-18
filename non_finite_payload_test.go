package decimal128

// The NaN-payload getter. d128_nanPayload returns a 128-bit payload, not an
// (op, operands, result) tuple, so it stays bespoke — the predicate assertions
// are native cases (nonfinite.txt), and payload round-tripping through
// formatting is covered by nanpayload.txt / printnan.txt.

import "testing"

func TestNonFinitePayloadPayloadGetter(t *testing.T) {
	if d128_nanPayload(d128_infinity(false)) != (u128{}) ||
		d128_nanPayload(d128_nan_64x64(false, true, 1234, 0)) != (u128{dw0: 1234}) ||
		d128_nanPayload(d128_nan_64x64(true, false, 999, 0)) != (u128{dw0: 999}) {
		t.Error("nanPayload")
	}
}
