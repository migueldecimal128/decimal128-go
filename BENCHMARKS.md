# decimal128-go benchmarks

A `testing.B` baseline for the public `Decimal128` API, established with
kernel-only framing (operands prepared outside the timed loop; results returned
into package sinks so nothing is optimised away). Add/sub alignment bands follow
the `decimal128-www/BenchmarkMatrix.md` taxonomy (consolidated cross-port results
live in `decimal128-www/BenchmarkResults.md`):

| Band | Meaning |
|------|---------|
| SQ | same qExp — no coefficient alignment |
| NQ | near qExp, narrow operands — small shift, no rounding |
| OQ | overlapping qExp — wide operands + small gap ⇒ align + round |
| FQ | far qExp — full shift, one operand swamped |

The Go port is a native, pure-Go implementation, so there is no in-process
libbid to compare against (unlike `decimal128-c`); this measures the port's own
performance.

## Running

```sh
go test -bench=. -benchmem -run='^$'
```

Append `-benchtime=2s` for steadier numbers, or `-cpuprofile` / `-memprofile`
for profiling. `-run='^$'` skips the unit tests so only benchmarks run.

## Baseline

*darwin/amd64 · Intel Core i9-9880H @ 2.30 GHz · Go 1.26 · single-threaded
latency, `-benchtime=200ms`. Absolute numbers vary by machine; the relative
shape is the point.*

| Operation              | ns/op | allocs/op | Notes |
|------------------------|------:|----------:|-------|
| Abs                    |   1.2 | 0 | sign-bit only |
| Add (SQ)               |   3.7 | 0 | same qExp, no alignment |
| Add (NQ)               |   9.8 | 0 | near qExp, narrow, no round |
| ContextAdd             |   4.9 | 0 | flag-tracking path |
| BID encode             |   3.7 | 0 | binary-integer significand |
| BID decode             |   4.1 | 0 | |
| Sub                    |  13.2 | 0 | unlike-sign alignment |
| Equal / Compare        | 17–18 | 0 | |
| Add (OQ)               |   —   | 0 | overlap + round (new band; x86 measurement pending) |
| Add (FQ)               |  22.0 | 0 | far qExp, full shift + residue |
| ToInt64                |  31.5 | 0 | |
| TotalOrder             |  37.3 | 0 | two-direction ≤ |
| RoundToIntegral        |  60.3 | 0 | |
| PowInt (n=20)          |  76.9 | 0 | binary exponentiation |
| Quantize               |  93.2 | 0 | |
| String                 |  94.0 | 1 | the returned string |
| Quo                    |  94.9 | 0 | Möller–Granlund |
| FMA                    | 104.1 | 0 | wide product + single round |
| Parse                  | 155.6 | 1 | |
| DPD encode / decode    | 165–169 | 0 | Cowlishaw declet codec |
| Mul                    | 194.6 | 0 | full 34×34 → finalize |
| Sqrt                   | 390.2 | 0 | exact integer floor root + pin |
| Exp / Ln / Log10       | 5.5–5.8 µs | 0 | [9,9] Padé in D38 |
| Root (n=3) / Pow       | 11.9–12.1 µs | 0 | log + exp10 composition |
| Finance: MortgagePayment | 1.5 µs | 0 | compound + divides |
| Finance: NPV (4 flows) | 0.76 µs | 1 | unwrap slice |
| Finance: Amortization (360) | 59 µs | 2 | 360-row schedule |

## Observations

- **The core arithmetic is allocation-free.** Only the string-returning
  (`Parse`, `String`) and slice-returning (`NPV`, `AmortizationSchedule`) APIs
  allocate; everything else is 0 alloc/op, as the value-return memory model (D-G3)
  intends.
- **Add cost scales with exponent distance** (3.7 → 9.8 → 22 ns across SQ/NQ/FQ),
  exactly the alignment/residue work the binary-exponent pre-check is meant to
  expose. (The new OQ band — wide operands + small gap — and the arm64 numbers are
  in `decimal128-www/BenchmarkResults.md`.)
- **`Mul` (195 ns) is slower than `Quo` (95 ns)** here — the full 34×34 → 68-digit
  product plus its scale-down-and-round finalize outweighs the Möller–Granlund
  divide on these operands.
- **BID is ~40× cheaper than DPD** for interchange (4 ns vs 165 ns): BID's
  significand is a plain binary integer, while DPD runs the per-declet Cowlishaw
  codec.
- **Transcendentals are µs-scale** (exp/ln ≈ 5.6 µs; pow/root ≈ 12 µs) — the
  expected cost of the 38-digit Padé kernels with their square-root reduction and
  three reconstruction squarings; `pow`/`root` roughly double it by composing a
  log with an exp10.
- **`ContextAdd` (4.9 ns) is within ~1 ns of bare `Add`** (3.7 ns SQ), confirming
  the flag-tracking wrapper adds negligible overhead.
