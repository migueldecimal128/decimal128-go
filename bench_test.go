package decimal128

import "testing"

// Performance baseline for the Go port, established through the public API (the
// surface users actually call). Inputs are prepared outside the timed loop
// (kernel-only framing, the analog of the C harness's bridge.hpp), and each loop
// body returns into a package sink so nothing is optimised away.
//
// Input classes mirror the C/libbid harness:
//   SE — same exponent      (no coefficient alignment)
//   NE — near exponent      (small alignment shift)
//   FE — far exponent       (full shift / one operand swamped)
//   N  — representative normal values
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

// representative operands per input class.
var (
	benchA   = MustParse("12345.67890123456789012345678901234")
	benchB   = MustParse("98765.43210987654321098765432109876")
	benchSEa = MustParse("123456.789")     // same exponent as SEb
	benchSEb = MustParse("987654.321")     //
	benchNEa = MustParse("12345.678")      // exponent off by 3 from NEb
	benchNEb = MustParse("9876.54321")     //
	benchFEa = MustParse("1.23456789E+30") // operands far apart in magnitude
	benchFEb = MustParse("9.87654321E-30") //
	benchPos = MustParse("2.5")
	benchBig = MustParse("123456789.987654321")
)

func BenchmarkAdd(b *testing.B) {
	b.Run("SE", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchSEa.Add(benchSEb)
		}
	})
	b.Run("NE", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchNEa.Add(benchNEb)
		}
	})
	b.Run("FE", func(b *testing.B) {
		for b.Loop() {
			benchSink = benchFEa.Add(benchFEb)
		}
	})
}

func BenchmarkSub(b *testing.B) {
	for b.Loop() {
		benchSink = benchA.Sub(benchB)
	}
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
		benchSink = ctx.Add(benchSEa, benchSEb)
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
