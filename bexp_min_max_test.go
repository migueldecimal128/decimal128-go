package decimal128

// Exhaustively verifies that calcBExpMin / calcBExpMax match the true
// floor/ceil values computed via float64 arithmetic, for every combination of
// bitLen in 0...255 and qExp in -7000...7000, and that the resulting interval
// width is always 0 or 1.

import (
	"math"
	"testing"
)

func TestBExpMinMaxExhaustive(t *testing.T) {
	log2of10 := math.Log2(10)
	trueMin := func(bitLen, qExp int) int {
		if bitLen == 0 {
			return 0
		}
		return int(math.Floor(float64(bitLen-1) + float64(qExp)*log2of10))
	}
	trueMax := func(bitLen, qExp int) int {
		if bitLen == 0 {
			return 0
		}
		return int(math.Ceil(float64(bitLen)+float64(qExp)*log2of10)) - 1
	}
	for bitLen := 0; bitLen <= 255; bitLen++ {
		for qExp := -7000; qExp <= 7000; qExp++ {
			mn, mx := calcBExpMin(bitLen, qExp), calcBExpMax(bitLen, qExp)
			if mn != trueMin(bitLen, qExp) || mx != trueMax(bitLen, qExp) {
				t.Fatalf("bExp mismatch at bitLen=%d qExp=%d", bitLen, qExp)
			}
			if d := mx - mn; d != 0 && d != 1 {
				t.Fatalf("delta out of range at bitLen=%d qExp=%d: %d", bitLen, qExp, d)
			}
		}
	}
}
