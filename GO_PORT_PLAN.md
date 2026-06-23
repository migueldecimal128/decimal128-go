# decimal128-go — Native Go Port Plan

*Draft 2026-06-23. A native, three-layer Go implementation of IEEE 754-2019
decimal128 — NOT a wrapper over `decimal128-c`. Governed by
`CrossPlatformArchitecture.md` (the cross-language constraint regime).*

## 0. Premises

- **Native, all three layers** (Primitive / Core / Wrapper) implemented in pure Go.
  No cgo, no binding over the C core.
- **Reference precedence:** **Swift (`decimal128-swift`) is PRIMARY** — the canonical
  arm every other port was transliterated from, and its value-struct model is the
  closest fit for Go. Secondary: **Java/Kotlin** (the primitive-layer model — they
  synthesize 128/256-bit from non-native limbs, exactly Go's situation), **Rust** (a
  same-shape native cross-check), **C** (the canonical Rosetta harness shape).
- **Constraint regime applies to the Go core** (§3 of the architecture doc): the
  C-mappable subset — `while`-only loops with a single `int` index, `if`-only
  branching, braces always, ASCII identifiers, C naming + the §8 mangling grammar,
  `SCREAMING_SNAKE_CASE` core enums with load-bearing integer values, no recursion
  (except `remainder*`, one level), no generics/closures/method-receiver dispatch in
  shippable core paths. Debug scaffolding is exempt.
- **Validation = the four-corpus Rosetta suite** (IBM/Cowlishaw dectest, IBM
  FPgen/fptest, Intel libbid, native) at bitwise-value + exception-flag parity,
  co-evolved with the Core so each operator is validated against real vectors as it
  lands. Target engine parity: **52,820 / 0**; wrapper parity: **46,507 / 0** (the
  numbers the Swift/Rust ports hit).

## 1. Decisions to settle first (the `D-G*` series)

Mirrors Rust's `D-R*`. Recommended answers in **bold**; these are the only places Go
genuinely diverges and should be locked before coding.

- **D-G1 — Module / package layout.** **One Go module `github.com/<org>/decimal128`,
  internal packages `internal/primitive`, `internal/core`, public root package
  `decimal128` (the wrapper).** Go's `internal/` gives the exact public/internal
  boundary the architecture wants (§2.5.2), enforced by the compiler. Core sub-tiers
  (core/math/print/interchange/finance) are **flat files within `internal/core`**
  (Go has no sub-package privacy gradient; like Swift/C they are semantically-inert
  groupings — §2.5.3). Filename + symbol prefix is the cross-port locator, not path.
- **D-G2 — Dword signedness. SETTLED 2026-06-23: signed `int64` dwords (strict field
  parity).** Go follows the §4.3 rule that every core uses signed `int64` dwords. Public
  field types (`dw0..dw3`) and all primitive signatures are `int64`, identical to
  Swift/Rust/Java/Kotlin/C. Inside primitive bodies, cast to `uint64` where unsigned
  semantics are required — `math/bits` (Add64/Sub64/Mul64/Div64/LeadingZeros64) is
  `uint64`-native, so each call site casts at the boundary (`uint64(dw)` in, `int64(r)`
  out). This is the Swift/C stance (they "could use unsigned but chose signed for
  uniformity"). The casts are confined to the hand-written primitive layer (§4.3), which
  already receives per-platform care; the rest of the codebase sees only `int64`.
- **D-G3 — Native 128-bit.** Go has **no `uint128`.** The primitive layer synthesizes
  128/256-bit from `uint64` limbs **via `math/bits`** (`bits.Add64`, `bits.Mul64`,
  `bits.Div64`, `bits.LeadingZeros64`) — the Java/Kotlin model, but with hardware
  carry/wide-mul intrinsics the JVM lacks. So Go's Primitive layer tracks the
  synthesize-from-limbs group while its Core memory model tracks the value-return
  group — both already-sanctioned divergences (§10), not new ones.
- **D-G4 — Memory model = value-return (NOT fill-in).** Go has value structs and no
  exceptions, like Swift/C. **All result aggregates (`Quot128Residue`, `Quot256Rem64`,
  `Diff256Swap`, `U128`, `U256`, `D128`, `D38`) are returned by value** —
  `q := divPow10_q128res_256(x, k)`. No thread-local fill-in pool (that is the
  Java/Kotlin workaround for heap pressure; Go value structs stay on the stack/in
  registers). This means **destination-first does not apply** — Go uses the Swift/C
  value-return signature shape directly.
- **D-G5 — `U128` representation.** Rust confines a native `u128` behind a dword-only
  newtype. Go has no `u128`, so **`U128` is a struct `{ dw0, dw1 uint64 }`** (matching
  the U256 shape), with `Dw0()/Dw1()` accessors. No newtype-over-native needed.
- **D-G6 — `verify` / `demand` / `impossible` (§9).** **`verify` gated on a
  package-level `const verifyEnabled = false`** so the compiler dead-code-eliminates it
  in production (Go has no macro; a `const bool` guard is the idiomatic equivalent of
  the Swift `#if VERIFY_ENABLED`). `demand(cond, msg)` and `impossible(msg)` use
  `panic` (the non-returning contract). Source location via `runtime.Caller` if needed
  (Go's analogue of `#file/#line`); the `string` message stays mandatory.
- **D-G7 — Enums.** Core enums are **`const` integer groups** (`type Round int`; the
  `int`-backed value-class analogue), `SCREAMING_SNAKE_CASE` names, load-bearing values
  (Residue 0–3, Round 0–4, Exception754 bit positions 0–4). User-facing wrapper enums
  are idiomatic Go (`camelCase`/exported), crossing the boundary as `int`.
- **D-G8 — `Int` width.** The core passes `qExp` and similar small values as Go `int`
  (64-bit on target platforms, like Swift's `Int`). Safe per §4.4 (qExp fits in far
  fewer than 32 bits).
- **D-G9 — Context / flags.** `DecContext` is a heap pointer carrying a mutable flag
  word (Go has no `Cell`; a pointer-to-struct with a plain `int64` flag field is the
  natural form). Threaded as `*Context`; the `_ctxnull` forms take a nilable `*Context`.
- **D-G10 — No operator overloading in core; wrapper gets Go-idiomatic methods.** The
  core is flat `d128_*` free functions. The wrapper exposes methods on `Decimal128`
  (`a.Add(b)`, `ctx.Add(a,b)`, `r.Add(a,b)` — modifier-as-receiver, §6.4 note). Go has
  no operator overloading, so the wrapper is method-based (no `+`), which is already the
  regime's baseline.

## 2. Target package layout

```
decimal128-go/
  go.mod                          // module github.com/<org>/decimal128
  verify.go                       // verify / demand / impossible (D-G6)
  decimal128.go        ...        // WRAPPER (root pkg): Decimal128 value + methods
  context.go  rounding.go  flags.go  ieee_class.go  style.go  comparison.go
  ordering.go  math.go  interchange.go  format.go  finance.go
  internal/
    primitive/                    // PRIMITIVE layer  (~30 files, ~4.4k LOC)
      tier0.go  u128.go  u256.go  residue.go
      u128_arith.go  u128_divrem.go  u128_strip.go
      u256_addsub.go  u256_mul.go  u256_divrem64.go  u256_double.go
      quot64_rem64.go ... quot256_residue.go  diff128_swap.go  diff256_swap.go
      stripped128_strip_count.go  calc_bit_len_digit_len.go
      mul_pow10.go  div_pow10_barrett.go  div_pow10_recipmul128.go
      reciprocal_div_mg.go  rrmp10_kernel.go  rrmp10_tables.go
      barrett_step.go  div_magic.go  primitive_tables.go
    core/                         // CORE layer  (~55 files, ~8k LOC)
      d128.go  d38.go  context.go  exception754.go  flags.go  round.go
      residue.go  parse_status.go  invalid_cause.go
      d128_arith_addsub.go  d128_arith_mul.go  d128_arith_div.go
      d128_arith_fma.go  d128_arith_compare.go
      d128_sqrt.go  d128_exp.go  d128_log.go  d128_pow.go  d128_pown.go
      d128_rootn.go  d128_compound.go  (+ *_constants.go)        // math tier
      d128_print.go  d128_parse.go  integer_parse_print.go        // print tier
      d128_serde_bid.go  d128_serde_dpd.go                        // interchange tier
      d128_to_integral.go  d128_logb_scaleb.go  d128_min_max.go
      d128_next_up_down.go  d128_total_order.go  d128_non_computational.go
      d128_strip_trailing_zeros.go  d128_constants.go
      finalize.go  d38_*.go  div_pow10.go  u256_div*.go  u256_sqrt.go
      basic_finance.go                                            // finance engine
    rosetta/                      // TEST harness (internal; co-evolved with core)
      case.go  dectest.go  fptest.go  intel.go  native.go
      dispatch.go  wrapper.go
  testdata/resources/{dectest,fptest,intel,native}/   // the four corpora (copied)
```

Symbol/file names are **identical to Swift** (cross-port grep parity, §2.5.3 / §8):
`umulHi_64_64x64`, `divPow10_q128res_256_barrett`, `d128_add_tte`, etc.

## 3. Surface to replicate (from the Swift reference; inventory confirmed via Rust)

- **Primitive (~96 public funcs):** tier0 scalar (8), U128 arith (11), U256 shift/add/sub/mul
  (20), 128/256 divrem (4), Barrett step + pow10 (7), recipmul/magic (2), Möller–Granlund
  (4), mulPow10 (5), RRMP10 (2), bit/digit-len (8), double/strip (3), residue classify (5).
  Result aggregates: `Quot{64,128,256}*`, `Diff{128,256}Swap`, `Stripped128StripCount`,
  `U128`, `U256` — field-for-field (`dw0..dw3`; `residue` last under value-return, no JVM
  header-gap reorder needed — §4.8 applies only to JVM ports).
- **Core operators**, each in `_tte` / `_rnd` / `_ctx` forms where applicable:
  add, subtract, multiply, square, divide, divideInteger, remainderNear, remainderTrunc,
  fma, sqrt, compare, compareSignaling; exp, exp10, ln, log10, pow, pown, rootn, compound;
  roundToIntegral (×5 rounding), toInt64 (×5), logB, scaleB, quantize, nextUp/nextDown,
  min/max family (8), stripTrailingZeros, negate/abs/copySign, parse, format, BID & DPD
  encode/decode. D38 extended-precision engine underneath (addsub/div/fma/sqrt/round).
- **Value types:** `D128 { ubdLo64, ubdHi64 }`, `D38 { coeffLo64, coeffHi64, qExpAndSignBit }`.
- **Enums:** Exception754 (0–4), Round (0–4), Residue (0–3), FormatStyle; wrapper-facing
  Rounding, IeeeClass (0–9), DecimalStyle (0–3), DecimalFlags, DecimalComparison.
- **Wrapper:** public `Decimal128` value type + methods, `Context`, `Rounding`,
  `Finance` facade (13 functions + AmortizationRow), Display/format, interchange bits.

## 4. Phased build (bottom-up, mirrors Rust's as-built sequencing)

Each phase ends on a **green gate**; Rosetta arms light up as operators land (a missing
arm on the native corpus is a compile error, never a silent skip).

- **Phase 0 — Scaffold. ✅ DONE 2026-06-23.** `go.mod`
  (`github.com/migueldecimal128/decimal128-go`, go 1.26.4); `internal/verify`
  (`Verify`/`Demand`/`Impossible`, D-G6, gated on `verify.Enabled`); placeholder
  `internal/primitive`, `internal/core`, root `decimal128`; four corpora copied to
  `testdata/resources` (dectest/fptest/intel/native/golden, ~17 MB); Rosetta skeleton
  (`Source`/`Case`/`Outcome`, native-corpus parser `ParseNativeLine`, zero-arm `Dispatch`,
  `Run` tally). **Gate met:** `gofmt`/`go build`/`go vet`/`go test ./...` all green; the
  harness parses 22 native `add.txt` cases → 22 Unsupported, 0 Failed (0 vectors executed,
  as expected with no arms wired).
- **Phase 1 — Primitive layer. ✅ DONE 2026-06-23.** All 33 non-prototype Swift
  Primitive files ported across 8 increments (commits c380966…7c49924): tier0; u128/u256
  types + arith + shifts; result aggregates; residue; u256 mul; Möller–Granlund (Alg.3/4/5);
  128÷64 / 128÷128 / 256÷64 divrem (full kernel set); pow10/pow5 tables; mulPow10; bit/digit
  length; double bridges; Barrett divPow10 + strip; magic divide; recipmul128; and the
  RRMP10 range-reciprocal engine + tables. Signed int64 dwords (D-G2); uint64 limb bodies
  via `math/bits`. **Gate met:** 41 tests validate every op against a `math/big` oracle
  (incl. RRMP10's 67 records / ~6300 cases, quotient+residue bit-exact); green under
  `-race`, `gofmt`/`vet` clean. No decimal semantics yet (that's Phase 2).
- **Phase 2 — Core foundation.** `D128`, `D38`, context/flags/round/residue/exception754,
  `finalize_128_ctx` / `finalize_256_ctx` + the FE fast-path entries (`finalizeAddSubFnz`,
  `finalizeSwampedFnz` — §2.4.6/§2.4.7), BID/DPD serde, parse, print. **Gate:** dectest
  encode/decode + parse/print vectors green (serde and parse are exercised early).
- **Phase 3 — Required arithmetic (§5).** add/sub (with the unaligned FE helpers),
  multiply/square, divide/divideInteger/remainder, fma, sqrt, compare/compareSignaling,
  roundToIntegral, scaleB/logB, quantize, min/max, nextUp/nextDown, non-computational.
  Wire each Rosetta arm as it lands. **Gate:** the bulk of dectest + fptest + Intel pass
  (~the 12k+ arithmetic vectors), bitwise value + flags.
- **Phase 4 — Recommended math (§9).** exp/exp10, ln/log10, pow, pown, rootn, compound
  (+ their constants; D38-domain Padé kernels). The **native** corpus is the sole coverage
  here. **Gate:** native corpus green for the math arms; full engine Rosetta → **52,820 / 0**.
- **Phase 5 — Wrapper.** Public `Decimal128` + methods, `Context` (modifier-as-receiver),
  `Rounding`, classification/ordering, interchange bits, Display/format, `DecimalStyle`.
  Add the WrapperRosetta gate (corpora driven through the public API only). **Gate:**
  WrapperRosetta → **46,507 / 0**.
- **Phase 6 — Finance.** `basic_finance` engine + public `Finance` facade (interest,
  annuities, NPV/IRR/MIRR, mortgage/amortization, CAGR). **Gate:** finance wrapper tests
  (port the Rust/Swift `TestFinanceWrapper`).
- **Phase 7 — Hardening.** `go vet`, race tests on `Context`, benchmark harness vs the
  other ports, document as-built decisions in a new **`GoCorePort.md`** companion (in
  `decimal128-www`, style of `RustCorePort.md`).

## 5. Sizing & risk

- **Size:** ~13.5k LOC shippable (primitive ~4.4k, core ~8k, wrapper ~1.4k) + the Rosetta
  harness + ported tests. Comparable to the Rust port.
- **Highest-risk items:** the RRMP10 kernel + tables (~1.2k LOC, the divPow10 engine) and
  the D38 transcendental kernels (exp/ln Padé) — port verbatim from Swift, validate against
  native-corpus vectors early. The Möller–Granlund reciprocal-divide primitives are the
  current division algorithm (all ports share the same recent migration commit) — port the
  Swift versions, not any retired Knuth-D path.
- **The one open decision blocking a clean start: D-G2 (dword signedness).** Everything else
  has a recommended default that follows Swift.

## 6. Validation invariant

"Passing" = **bitwise-equal value** (`d128_bitwiseEQ`: sign, exponent, coefficient cohort,
NaN payload, ±0 all exact) **AND exact exception flags** (`xuozi` string), per corpus, with
the per-corpus reconciliation the harness already encodes (fptest trap-unwrap + tininess-`u`
strip; Intel NaN-payload/number-cohort normalize + int-sentinel allowance; dectest
value-only ops). The Go harness re-implements only the per-corpus *parsers*; the dispatch
table and comparison logic mirror the shared design.
```
