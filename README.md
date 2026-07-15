# decimal128-go

A native, pure-Go implementation of **IEEE 754-2019 decimal128** — 128-bit
base-10 floating point with 34 significant digits and an exponent range of
10⁻⁶¹⁴³ … 10⁺⁶¹⁴⁴.

Decimal arithmetic represents values like `0.1` exactly, so it is the right tool
for money, tax, billing, and anywhere base-2 rounding error is unacceptable. The
package is a faithful transliteration of the `decimal128-swift` reference — **not**
a binding over a C library (no cgo) — and is validated continuously against the
IBM/Cowlishaw and Intel conformance corpora.

- **Correct:** every required IEEE 754 operation (§5) plus the recommended math
  (§9 — exp, ln, log₁₀, pow, …), correctly rounded, with the five exception flags.
- **Exact where it matters:** the hot arithmetic paths are **allocation-free**.
- **Pure Go:** no cgo, no external dependencies; `go vet`-clean and race-tested.

## Install

```sh
go get github.com/migueldecimal128/decimal128-go
```

```go
import decimal128 "github.com/migueldecimal128/decimal128-go"
```

The package name is `decimal128`.

## Quick start

```go
package main

import (
	"fmt"

	decimal128 "github.com/migueldecimal128/decimal128-go"
)

func main() {
	price := decimal128.MustParse("19.99")
	qty := decimal128.FromInt(3)

	total := price.Mul(qty)
	fmt.Println(total) // 59.97  — exact, no binary rounding error

	// Parsing untrusted input returns an error.
	if d, err := decimal128.Parse("3.14159"); err == nil {
		fmt.Println(d.Sqrt()) // 1.772453102...
	}
}
```

## Arithmetic

The bare methods round ties-to-even and raise no flags, the way `float64`
arithmetic does:

```go
a := decimal128.MustParse("1.5")
b := decimal128.MustParse("0.3")

a.Add(b) // 1.8
a.Sub(b) // 1.2
a.Mul(b) // 0.45
a.Quo(b) // 5
a.Neg()  // -1.5
a.Abs()
a.Sqrt()
a.FMA(b, b) // a + b*b, single rounding (fused multiply-add)
```

### Choosing a rounding direction

Use a `Rounding` as the receiver when you need a specific direction (still no
flags):

```go
one, three := decimal128.FromInt(1), decimal128.FromInt(3)

decimal128.TowardZero.Quo(one, three)     // 0.3333333333333333333333333333333333
decimal128.TowardPositive.Quo(one, three) // 0.3333333333333333333333333333333334
```

Directions: `ToNearestEven` (default), `ToNearestAway`, `TowardZero`,
`TowardPositive`, `TowardNegative`.

### Tracking IEEE exception flags

A `Context` carries a rounding direction and an accumulating flag sink:

```go
ctx := decimal128.NewContext()

ctx.Quo(decimal128.FromInt(1), decimal128.FromInt(3))
ctx.RaisedFlags().Has(decimal128.FlagInexact) // true

ctx.Quo(decimal128.FromInt(1), decimal128.Zero())
ctx.RaisedFlags().Has(decimal128.FlagDivisionByZero) // true (flags accumulate)

ctx.ClearFlags()
```

Flags: `FlagInvalidOperation`, `FlagDivisionByZero`, `FlagOverflow`,
`FlagUnderflow`, `FlagInexact`. A `Context` is a reference type — use one per
goroutine.

## Mathematical functions

Available bare (ties-to-even) on `Decimal128` and flag-tracked on `Context`:

```go
x.Exp()       // eˣ
x.Exp10()     // 10ˣ
x.Ln()        // natural log
x.Log10()     // base-10 log
x.Pow(y)      // xʸ  (decimal exponent)
x.PowInt(n)   // xⁿ  (integer exponent, IEEE pown)
x.Root(n)     // ⁿ√x (IEEE rootn)
x.Compound(n) // (1+x)ⁿ
```

## Comparison and ordering

```go
a.Equal(b)        // IEEE: +0 == -0, NaN unequal to everything
a.Less(b)
a.LessOrEqual(b)
a.ComparePartial(b) // CmpLess | CmpEqual | CmpGreater | CmpUnordered
a.TotalOrder(b)   // §5.10 total order — never unordered

decimal128.Minimum(a, b)        // NaN propagates
decimal128.MaximumNumber(a, b)  // NaN treated as missing
```

## Classification, rounding, and quantum

```go
x.IsNaN(); x.IsInf(); x.IsFinite(); x.IsZero(); x.Signbit()
x.IsNormal(); x.IsSubnormal()
x.Class()              // e.g. PositiveNormal, NegativeZero, QuietNaN

x.RoundToIntegral()    // ties-to-even
x.Quantize(other)      // re-scale to other's exponent
x.ScaleB(n)            // × 10ⁿ
x.StripTrailingZeros()
x.ToInt64()            // truncate toward zero
```

## Formatting and interchange

```go
x.String()                                  // GDAS scientific (fmt.Stringer)
x.Text(decimal128.StyleEngineering)         // Automatic | Exponential | Engineering | CoefficientExponent

hi, lo := x.BID()                           // IEEE BID 128-bit interchange pattern
x = decimal128.DecimalFromBID(hi, lo)
hi, lo = x.DPD()                            // IEEE DPD interchange pattern
x = decimal128.DecimalFromDPD(hi, lo)
```

## Finance

A small finance facade built on the exact decimal arithmetic:

```go
decimal128.Finance.MortgagePayment(
	decimal128.MustParse("200000"), decimal128.MustParse("0.005"), 360) // 1199.10105...

decimal128.Finance.NPV(decimal128.MustParse("0.10"), cashFlows)
irr, ok := decimal128.Finance.IRR(cashFlows)
```

Also: `SimpleInterest`, `CompoundInterest`, `EffectiveAnnualRate`,
`PresentValue`/`FutureValue`, annuities, `CAGR`, `MIRR`, and
`AmortizationSchedule`.

## Validation

Every operation is checked against the conformance corpora the Swift reference
uses, both through the internal engine and through the public API:

- **native** corpus: **1547 / 1547**, 0 failures.
- **dectest** (IBM/Cowlishaw `dq*`): every wired file green.
- **WrapperRosetta** (corpora driven through the public `Decimal128` API only):
  **7769 / 0**.

The 38-digit transcendental engine, which has no conformance corpus, is validated
directly against a `math/big` oracle (correctly rounded to 38 significant digits)
plus float64 cross-checks. The suite runs clean under `go test -race` and
`go vet`.

```sh
go test ./...
go test -race ./...
go test -bench=. -benchmem -run='^$'   # see BENCHMARKS.md
```

## Design

`decimal128-go` is one of several parallel implementations of the same
specification (Swift, C, Rust, Java, Kotlin, Go). It is a native port — all three
layers (primitive integer kernels, the decimal core, the public wrapper) are
implemented in pure Go. See [`BENCHMARKS.md`](BENCHMARKS.md) for performance, and
the `GoCorePort.md` / `CrossPlatformArchitecture.md` companion docs in the
`decimal128-www` repository for the as-built record and the cross-language
architecture.

## License

MIT — see [`LICENSE`](LICENSE).
