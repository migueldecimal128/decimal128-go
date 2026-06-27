package decimal128

import "testing"

// Performance baseline for the Go port, established through the public API (the
// surface users actually call). Inputs are prepared outside the timed loop
// (kernel-only framing, the analog of the C harness's bridge.hpp), and each loop
// body returns into a package sink so nothing is optimised away.
//
// Add/sub alignment bands follow the BenchmarkMatrix.md taxonomy (SQ/NQ/OQ/FQ):
//   SQ — same qExp           (Δ=0, no coefficient alignment)
//   NQ — near qExp, no round (small shift, narrow operands, result fits 34 digits)
//   OQ — overlapping qExp    (wide operands + small gap ⇒ align + round)
//   FQ — far qExp            (full shift / one operand swamped)
//
// Run: go test -bench=. -benchmem -run='^$'

var (
	benchSink   Decimal128
	benchBool   bool
	benchInt    int64
	benchStr    string
	benchCmp    Comparison
	benchHi     uint64
	benchLo     uint64
	benchFinRow []AmortizationRow
)

// representative operands per alignment band.
var (
	benchA   = MustParse("12345.67890123456789012345678901234")
	benchB   = MustParse("98765.43210987654321098765432109876")
	benchSQa = MustParse("123456.789") // SQ: same qExp (both 10^-3)
	benchSQb = MustParse("987654.321") //
	benchNQa = MustParse("12345.678")  // NQ: near qExp, narrow ⇒ no rounding
	benchNQb = MustParse("9876.54321") //   (gap 2, ~8-9 digits, result fits 34)
	// OQ: two 34-digit coefficients an exponent-gap of 4 apart ⇒ the aligned sum
	// is ~38 digits and must round (trim 4), with the smaller operand overlapping.
	benchOQa = MustParse("1234567890123456789012345678901234")    // 34 digits, exp 0
	benchOQb = MustParse("9876543210987654321098765432109876E-4") // 34 digits, exp -4
	benchFQa = MustParse("1.23456789E+30")                        // FQ: far apart in magnitude
	benchFQb = MustParse("9.87654321E-30")                        //   (gap 60 ⇒ swamped)
	benchPos = MustParse("2.5")
	benchBig = MustParse("123456789.987654321")
)

func BenchmarkAdd(b *testing.B) {
	b.Run("SQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchSQa.Add(benchSQb)
		}
	})
	b.Run("NQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchNQa.Add(benchNQb)
		}
	})
	b.Run("OQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchOQa.Add(benchOQb)
		}
	})
	b.Run("FQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchFQa.Add(benchFQb)
		}
	})
}

func BenchmarkSub(b *testing.B) {
	b.Run("SQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchSQa.Sub(benchSQb)
		}
	})
	b.Run("NQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchNQa.Sub(benchNQb)
		}
	})
	b.Run("OQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchOQa.Sub(benchOQb)
		}
	})
	b.Run("FQ", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchFQa.Sub(benchFQb)
		}
	})
}

func BenchmarkMul(b *testing.B) {
	for b.Loop() {
		benchSink = benchA.Mul(benchB)
	}
}

func BenchmarkQuo(b *testing.B) {
	for b.Loop() {
		benchSink = benchA.Quo(benchB)
	}
}

// PROTOTYPE PT measurement: non-PT small divisor vs power-of-ten divisor (two encodings).
var (
	benchCDb  = MustParse("98.76") // non-PT general small divisor
	benchPTb  = MustParse("1000")  // power of ten, Cy=1000 (m=3 + strip)
	benchPT1b = MustParse("1E3")   // power of ten, Cy=1 (m=0, trivial)
)

func BenchmarkQuoCD(b *testing.B)  { for b.Loop() { benchSink = benchA.Quo(benchCDb) } }
func BenchmarkQuoPT(b *testing.B)  { for b.Loop() { benchSink = benchA.Quo(benchPTb) } }
func BenchmarkQuoPT1(b *testing.B) { for b.Loop() { benchSink = benchA.Quo(benchPT1b) } }

func BenchmarkFMA(b *testing.B) {
	for b.Loop() {
		benchSink = benchA.AddProduct(benchB, benchPos)
	}
}

func BenchmarkSqrt(b *testing.B) {
	for b.Loop() {
		benchSink = benchBig.Sqrt()
	}
}

func BenchmarkAbs(b *testing.B) {
	for b.Loop() {
		benchSink = benchB.Abs()
	}
}

func BenchmarkCompare(b *testing.B) {
	for b.Loop() {
		benchCmp = benchA.Compare(benchB)
	}
}

func BenchmarkEqual(b *testing.B) {
	for b.Loop() {
		benchBool = benchA.Equal(benchB)
	}
}

func BenchmarkTotalOrder(b *testing.B) {
	for b.Loop() {
		benchCmp = benchA.TotalOrder(benchB)
	}
}

func BenchmarkExp(b *testing.B) {
	for b.Loop() {
		benchSink = benchPos.Exp()
	}
}

func BenchmarkLn(b *testing.B) {
	for b.Loop() {
		benchSink = benchBig.Ln()
	}
}

func BenchmarkLog10(b *testing.B) {
	for b.Loop() {
		benchSink = benchBig.Log10()
	}
}

func BenchmarkPow(b *testing.B) {
	for b.Loop() {
		benchSink = benchPos.Pow(benchPos)
	}
}

func BenchmarkPowInt(b *testing.B) {
	for b.Loop() {
		benchSink = benchPos.PowInt(20)
	}
}

func BenchmarkRoot(b *testing.B) {
	for b.Loop() {
		benchSink = benchBig.Root(3)
	}
}

func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		benchSink = ParseOrNaN("12345.67890123456789012345678901234")
	}
}

func BenchmarkString(b *testing.B) {
	for b.Loop() {
		benchStr = benchA.String()
	}
}

func BenchmarkToInt64(b *testing.B) {
	for b.Loop() {
		benchInt = benchBig.ToInt64()
	}
}

func BenchmarkBIDEncode(b *testing.B) {
	for b.Loop() {
		benchHi, benchLo = benchA.BID()
	}
}

func BenchmarkBIDDecode(b *testing.B) {
	hi, lo := benchA.BID()
	b.ResetTimer()
	for b.Loop() {
		benchSink = DecimalFromBID(hi, lo)
	}
}

func BenchmarkDPDEncode(b *testing.B) {
	for b.Loop() {
		benchHi, benchLo = benchA.DPD()
	}
}

func BenchmarkDPDDecode(b *testing.B) {
	hi, lo := benchA.DPD()
	b.ResetTimer()
	for b.Loop() {
		benchSink = DecimalFromDPD(hi, lo)
	}
}

func BenchmarkRoundToIntegral(b *testing.B) {
	for b.Loop() {
		benchSink = benchA.RoundToIntegral()
	}
}

func BenchmarkQuantize(b *testing.B) {
	q := MustParse("1.00")
	b.ResetTimer()
	for b.Loop() {
		benchSink = benchA.Quantize(q)
	}
}

func BenchmarkContextAdd(b *testing.B) {
	ctx := NewContext()
	b.ResetTimer()
	for b.Loop() {
		benchSink = ctx.Add(benchSQa, benchSQb)
	}
}

func BenchmarkFinanceMortgage(b *testing.B) {
	p := MustParse("200000")
	r := MustParse("0.005")
	b.ResetTimer()
	for b.Loop() {
		benchSink = Finance.MortgagePayment(p, r, 360)
	}
}

func BenchmarkFinanceNPV(b *testing.B) {
	rate := MustParse("0.10")
	flows := []Decimal128{MustParse("-1000"), MustParse("400"), MustParse("500"), MustParse("400")}
	b.ResetTimer()
	for b.Loop() {
		benchSink = Finance.NPV(rate, flows)
	}
}

func BenchmarkFinanceAmortization(b *testing.B) {
	p := MustParse("200000")
	r := MustParse("0.005")
	b.ResetTimer()
	for b.Loop() {
		benchFinRow = Finance.AmortizationSchedule(p, r, 360)
	}
}
