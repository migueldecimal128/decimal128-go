package decimal128

// Rosetta operator partition (Rosetta.md §9). For each corpus source, every
// operator token the corpus contains is in exactly one of that source's include
// (token -> dispatch token) or exclude (token -> reason it is not run) maps;
// the coverage tests below fail otherwise. Keys and reasons mirror Swift's
// RosettaIntelMaps / RosettaDectestMaps / RosettaFptestMaps verbatim; include
// values are this harness's own dispatch tokens. `native` is exempt: its tokens
// are dispatch names already, and an unknown one fails at dispatch.
//
// Test-only code, exempt from the §3 constraint regime.

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// dectestFiles is the full dectest file list (Swift's), read by the core runner,
// the wrapper runner and the coverage test alike (R-6).
var dectestFiles = []string{
	"_754_2019.dectest", "decQuad.decTest", "dqAbs.decTest", "dqAdd.decTest",
	"dqBase.decTest", "dqCanonical.decTest", "dqClass.decTest", "dqCompare.decTest",
	"dqCompareSig.decTest", "dqCompareTotal.decTest", "dqCompareTotalMag.decTest", "dqCopy.decTest",
	"dqCopyAbs.decTest", "dqCopyNegate.decTest", "dqCopySign.decTest", "dqDivide.decTest",
	"dqDivideInt.decTest", "dqEncode.decTest", "dqFMA.decTest", "dqLogB.decTest",
	"dqMax.decTest", "dqMaxMag.decTest", "dqMin.decTest", "dqMinMag.decTest",
	"dqMinus.decTest", "dqMultiply.decTest", "dqNextMinus.decTest", "dqNextPlus.decTest",
	"dqPlus.decTest", "dqQuantize.decTest", "dqReduce.decTest", "dqRemainder.decTest",
	"dqRemainderNear.decTest", "dqSameQuantum.decTest", "dqScaleB.decTest", "dqShift.decTest",
	"dqSubtract.decTest", "dqToIntegral.decTest", "exp.decTest", "ln.decTest",
	"log10.decTest",
}

// fptestFiles is the full fptest file list (Swift's).
var fptestFiles = []string{
	"Decimal-Basic-Types-Inputs.fptest",
	"Decimal-Basic-Types-Intermediate.fptest",
	"Decimal-Clamping.fptest",
	"Decimal-Mul-Trailing-Zeros.fptest",
	"Decimal-Overflow-correctedByMTH.fptest",
	"Decimal-Rounding.fptest",
	"Decimal-Trailing-And-Leading-Zeros-Input.fptest",
	"Decimal-Trailing-And-Leading-Zeros-Result.fptest",
	"Decimal-Underflow.fptest",
}

// intelInScope is the coarse width pre-filter: only decimal128 (bid128_) tokens
// reach the partition; bid32/bid64/binary-interchange tokens are out of scope.
func intelInScope(op string) bool {
	return strings.HasPrefix(op, "bid128_")
}

// intelInclude: bid128_ operator -> harness dispatch token. The C99-named
// entries (copy, negate, from_*, ldexp, scalbln, logb, llquantexp, nearbyint,
// llrint, lrint, llround, lround) map to the IEEE 754 operation the core
// implements (RosettaCoveragePartitionWorkOrder.md Phase 1).
var intelInclude = map[string]string{
	"bid128_abs":                         "d128_abs",
	"bid128_add":                         "add",
	"bid128_class":                       "class",
	"bid128_copy":                        "d128_copy",
	"bid128_copySign":                    "copysign",
	"bid128_div":                         "divide",
	"bid128_exp":                         "d128_exp_ctx",
	"bid128_exp10":                       "d128_exp10_ctx",
	"bid128_expm1":                       "d128_expm1_ctx",
	"bid128_fma":                         "fma",
	"bid128_from_int32":                  "from_int64",
	"bid128_from_int64":                  "from_int64",
	"bid128_from_string":                 "parse_ctx",
	"bid128_from_uint32":                 "from_uint64",
	"bid128_from_uint64":                 "from_uint64",
	"bid128_isFinite":                    "d128_isFinite",
	"bid128_isInf":                       "d128_isInf",
	"bid128_isNaN":                       "d128_isNaN",
	"bid128_isNormal":                    "d128_isNormal",
	"bid128_isSignaling":                 "d128_isSignaling",
	"bid128_isSigned":                    "d128_isSigned",
	"bid128_isSubnormal":                 "d128_isSubnormal",
	"bid128_isZero":                      "d128_isZero",
	"bid128_ldexp":                       "scaleb",
	"bid128_llquantexp":                  "d128_quantexp",
	"bid128_llrint":                      "to_int64_xmode",
	"bid128_llround":                     "d128_to_int64_rninta",
	"bid128_log":                         "d128_ln_ctx",
	"bid128_log10":                       "d128_log10_ctx",
	"bid128_log1p":                       "d128_lnp1_ctx",
	"bid128_logb":                        "logb",
	"bid128_lrint":                       "to_int64_xmode",
	"bid128_lround":                      "d128_to_int64_rninta",
	"bid128_maxnum":                      "max",
	"bid128_maxnum_mag":                  "maxmag",
	"bid128_minnum":                      "min",
	"bid128_minnum_mag":                  "minmag",
	"bid128_mul":                         "multiply",
	"bid128_nearbyint":                   "round_integral_mode",
	"bid128_negate":                      "d128_negate",
	"bid128_nextdown":                    "nextminus",
	"bid128_nextup":                      "nextplus",
	"bid128_quantexp":                    "d128_quantexp",
	"bid128_quantize":                    "quantize",
	"bid128_quiet_equal":                 "d128_isEqual_ctx",
	"bid128_quiet_greater":               "d128_isGreater_ctx",
	"bid128_quiet_greater_equal":         "d128_isGreaterEqual_ctx",
	"bid128_quiet_greater_unordered":     "d128_isGreaterUnordered_ctx",
	"bid128_quiet_less":                  "d128_isLess_ctx",
	"bid128_quiet_less_equal":            "d128_isLessEqual_ctx",
	"bid128_quiet_less_unordered":        "d128_isLessUnordered_ctx",
	"bid128_quiet_not_equal":             "d128_isNotEqual_ctx",
	"bid128_quiet_not_greater":           "d128_isNotGreater_ctx",
	"bid128_quiet_not_less":              "d128_isNotLess_ctx",
	"bid128_quiet_ordered":               "d128_isOrdered_ctx",
	"bid128_quiet_unordered":             "d128_isUnordered_ctx",
	"bid128_rem":                         "remaindernear",
	"bid128_round_integral_exact":        "tointegralx",
	"bid128_round_integral_nearest_away": "d128_round_integral_nearest_away",
	"bid128_round_integral_nearest_even": "d128_round_integral_nearest_even",
	"bid128_round_integral_negative":     "d128_round_integral_negative",
	"bid128_round_integral_positive":     "d128_round_integral_positive",
	"bid128_round_integral_zero":         "d128_round_integral_zero",
	"bid128_sameQuantum":                 "samequantum",
	"bid128_scalbln":                     "scaleb",
	"bid128_scalbn":                      "scaleb",
	"bid128_signaling_greater":           "d128_isGreaterSignaling_ctx",
	"bid128_signaling_greater_equal":     "d128_isGreaterEqualSignaling_ctx",
	"bid128_signaling_greater_unordered": "d128_isGreaterUnorderedSignaling_ctx",
	"bid128_signaling_less":              "d128_isLessSignaling_ctx",
	"bid128_signaling_less_equal":        "d128_isLessEqualSignaling_ctx",
	"bid128_signaling_less_unordered":    "d128_isLessUnorderedSignaling_ctx",
	"bid128_signaling_not_greater":       "d128_isNotGreaterSignaling_ctx",
	"bid128_signaling_not_less":          "d128_isNotLessSignaling_ctx",
	"bid128_sqrt":                        "squareroot",
	"bid128_sub":                         "subtract",
	"bid128_to_int64_ceil":               "d128_to_int64_ceil",
	"bid128_to_int64_floor":              "d128_to_int64_floor",
	"bid128_to_int64_int":                "d128_to_int64_int",
	"bid128_to_int64_rnint":              "d128_to_int64_rnint",
	"bid128_to_int64_rninta":             "d128_to_int64_rninta",
	"bid128_to_int64_xceil":              "d128_to_int64_xceil",
	"bid128_to_int64_xfloor":             "d128_to_int64_xfloor",
	"bid128_to_int64_xint":               "d128_to_int64_xint",
	"bid128_to_int64_xrnint":             "d128_to_int64_xrnint",
	"bid128_to_int64_xrninta":            "d128_to_int64_xrninta",
	"bid128_to_uint64_ceil":              "d128_to_uint64_ceil",
	"bid128_to_uint64_floor":             "d128_to_uint64_floor",
	"bid128_to_uint64_int":               "d128_to_uint64_int",
	"bid128_to_uint64_rnint":             "d128_to_uint64_rnint",
	"bid128_to_uint64_rninta":            "d128_to_uint64_rninta",
	"bid128_to_uint64_xceil":             "d128_to_uint64_xceil",
	"bid128_to_uint64_xfloor":            "d128_to_uint64_xfloor",
	"bid128_to_uint64_xint":              "d128_to_uint64_xint",
	"bid128_to_uint64_xrnint":            "d128_to_uint64_xrnint",
	"bid128_to_uint64_xrninta":           "d128_to_uint64_xrninta",
	"bid128_totalOrder":                  "d128_totalOrder",
	"bid128_totalOrderMag":               "d128_totalOrderMag",
}

// intelExclude: bid128_ operator -> reason it is not run (Swift's reasons,
// verbatim).
var intelExclude = map[string]string{
	"bid128_acos":              "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_acosh":             "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_asin":              "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_asinh":             "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_atan":              "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_atan2":             "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_atanh":             "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_cbrt":              "not an IEEE 754-2019 operation (rootn covers cube roots)",
	"bid128_cos":               "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_cosh":              "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_erf":               "not an IEEE 754-2019 operation",
	"bid128_erfc":              "not an IEEE 754-2019 operation",
	"bid128_exp2":              "IEEE 754-2019 §9.2 recommended operation, not implemented",
	"bid128_fdim":              "C99 <math.h> helper with no IEEE 754 counterpart in the core",
	"bid128_fmod":              "C fmod is exact for any quotient; d128_remainderTrunc (GDAS remainder) raises division-impossible when the quotient exceeds 34 digits (10 of 90 vectors differ)",
	"bid128_frexp":             "C99 <math.h> helper with no IEEE 754 counterpart in the core (two results)",
	"bid128_hypot":             "IEEE 754-2019 §9.2 recommended operation, not implemented",
	"bid128_ilogb":             "integer-result logB — not implemented; bid128_logb covers the value",
	"bid128_inf":               "special-value constructor — out of scope (library builds these directly)",
	"bid128_isCanonical":       "isCanonical over a UBD value is always true (754-conformant: UBD admits only canonical representations). Intel tests BID-encoding canonicality, which is normalized away at decode per 754-2019 — not a property of a decoded value, so not expressible through d128_isCanonical (which is correctly a `{ true }` stub).",
	"bid128_lgamma":            "not an IEEE 754-2019 operation",
	"bid128_log2":              "IEEE 754-2019 §9.2 recommended operation, not implemented (log2p1 is)",
	"bid128_modf":              "C99 <math.h> helper with no IEEE 754 counterpart in the core (two results)",
	"bid128_nan":               "special-value constructor — out of scope (library builds these directly)",
	"bid128_nextafter":         "nextafter/nexttoward — declined (not in IEEE 754-2019; see decision log)",
	"bid128_nexttoward":        "nextafter/nexttoward — declined (not in IEEE 754-2019; see decision log)",
	"bid128_pow":               "not yet passing (134/168 in Phase 0): inexact-result cohort (punchlist: pow drops a trailing zero), x < 0 with a huge integer y (punchlist), and Intel conventions; include when fixed",
	"bid128_quantum":           "754 quantum as a decimal (1E q) — not implemented; quantexp covers the exponent",
	"bid128_radix":             "radix() constant accessor — trivially 10; not a test-worthy operation",
	"bid128_sin":               "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_sinh":              "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_tan":               "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_tanh":              "trig/inverse-trig — not required by IEEE 754-2019 decimal",
	"bid128_tgamma":            "not an IEEE 754-2019 operation",
	"bid128_to_bid32":          "narrowing to other decimal width — out of scope (this library is decimal128 only)",
	"bid128_to_bid64":          "narrowing to other decimal width — out of scope (this library is decimal128 only)",
	"bid128_to_binary128":      "decimal→binary interchange conversion — out of scope",
	"bid128_to_binary32":       "decimal→binary interchange conversion — out of scope",
	"bid128_to_binary64":       "decimal→binary interchange conversion — out of scope",
	"bid128_to_binary80":       "decimal→binary interchange conversion — out of scope",
	"bid128_to_int16_ceil":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_floor":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_int":      "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_rnint":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_rninta":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_xceil":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_xfloor":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_xint":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_xrnint":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int16_xrninta":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_ceil":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_floor":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_int":      "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_rnint":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_rninta":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_xceil":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_xfloor":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_xint":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_xrnint":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int32_xrninta":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_ceil":      "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_floor":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_int":       "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_rnint":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_rninta":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_xceil":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_xfloor":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_xint":      "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_xrnint":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_int8_xrninta":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_string":         "Intel's own text format (+cccE±q, +Inf, +NaN); our raw format differs only in sign and special-value spelling",
	"bid128_to_uint16_ceil":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_floor":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_int":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_rnint":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_rninta":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_xceil":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_xfloor":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_xint":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_xrnint":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint16_xrninta": "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_ceil":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_floor":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_int":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_rnint":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_rninta":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_xceil":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_xfloor":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_xint":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_xrnint":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint32_xrninta": "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_ceil":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_floor":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_int":      "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_rnint":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_rninta":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_xceil":    "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_xfloor":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_xint":     "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_xrnint":   "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
	"bid128_to_uint8_xrninta":  "8/16/32-bit integer conversion — out of scope (int64 and uint64 are in scope)",
}

// dectestInclude: dectest operation (lower-cased) -> harness dispatch token.
// canonical (dqCanonical decodes non-canonical DPD encodings) runs as copy;
// class compares d128_ieeeClass by its dectest name.
var dectestInclude = map[string]string{
	"abs":           "abs",
	"add":           "add",
	"apply":         "apply",
	"canonical":     "canonical",
	"class":         "class",
	"compare":       "compare",
	"comparesig":    "comparesig",
	"comparetotal":  "comparetotal",
	"comparetotmag": "comparetotmag",
	"copy":          "copy",
	"copyabs":       "copyabs",
	"copynegate":    "copynegate",
	"copysign":      "copysign",
	"divide":        "divide",
	"divideint":     "divideint",
	"exp":           "d128_exp_ctx",
	"fma":           "fma",
	"ln":            "d128_ln_ctx",
	"log10":         "d128_log10_ctx",
	"logb":          "logb",
	"max":           "max",
	"maxmag":        "maxmag",
	"min":           "min",
	"minmag":        "minmag",
	"minus":         "minus",
	"multiply":      "multiply",
	"nextminus":     "nextminus",
	"nextplus":      "nextplus",
	"quantize":      "quantize",
	"reduce":        "reduce",
	"remainder":     "remainder",
	"remaindernear": "remaindernear",
	"samequantum":   "samequantum",
	"scaleb":        "scaleb",
	"subtract":      "subtract",
	"toeng":         "toeng",
	"tointegralx":   "tointegralx",
	"tosci":         "tosci",
}

// dectestExclude: dectest operation -> reason.
var dectestExclude = map[string]string{
	"plus":  "plus (unary 0+x) — not an IEEE 754-2019 operation (Cowlishaw GDAS only); see dq-coverage decision log",
	"shift": "shift (coefficient digit shift) — not an IEEE 754-2019 operation (Cowlishaw GDAS only); see dq-coverage decision log",
}

// fptestInclude: fptest operator symbol -> harness dispatch token.
var fptestInclude = map[string]string{
	"+": "add",
	"-": "subtract",
	"*": "multiply",
	"/": "divide",
}

// fptestExclude is empty: every operator fptest carries is in scope.
var fptestExclude = map[string]string{}

// rosettaArity is the operand count every case of a dispatch token must carry
// (R-7). A token missing here has no dispatch arm. An explicit table so it
// ports verbatim.
var rosettaArity = map[string]int{
	// 1 operand
	"d128_toString":                     1,
	"d128_toExpString":                  1,
	"d128_toEngString":                  1,
	"d128_toRawString":                  1,
	"d128_isSigned":                     1,
	"d128_isInf":                        1,
	"d128_isFinite":                     1,
	"d128_isNaN":                        1,
	"d128_isSignaling":                  1,
	"d128_isQNaN":                       1,
	"d128_isNormal":                     1,
	"d128_isSubnormal":                  1,
	"d128_isZero":                       1,
	"d128_quantexp":                     1,
	"d128_to_int64_rnint":               1,
	"d128_to_int64_rninta":              1,
	"d128_to_int64_ceil":                1,
	"d128_to_int64_floor":               1,
	"d128_to_int64_int":                 1,
	"d128_to_int64_xrnint":              1,
	"d128_to_int64_xrninta":             1,
	"d128_to_int64_xceil":               1,
	"d128_to_int64_xfloor":              1,
	"d128_to_int64_xint":                1,
	"d128_to_uint64_rnint":              1,
	"d128_to_uint64_rninta":             1,
	"d128_to_uint64_ceil":               1,
	"d128_to_uint64_floor":              1,
	"d128_to_uint64_int":                1,
	"d128_to_uint64_xrnint":             1,
	"d128_to_uint64_xrninta":            1,
	"d128_to_uint64_xceil":              1,
	"d128_to_uint64_xfloor":             1,
	"d128_to_uint64_xint":               1,
	"d128_roundToIntegralSignalInexact": 1,
	"tointegralx":                       1,
	"d128_round_integral_nearest_even":  1,
	"d128_round_integral_nearest_away":  1,
	"d128_round_integral_zero":          1,
	"d128_round_integral_positive":      1,
	"d128_round_integral_negative":      1,
	"d128_sqrt_ctx":                     1,
	"squareroot":                        1,
	"d128_stripTrailingZeros":           1,
	"reduce":                            1,
	"d128_logB_ctxnull":                 1,
	"logb":                              1,
	"d128_nextUp_ctxnull":               1,
	"nextplus":                          1,
	"d128_nextDown_ctxnull":             1,
	"nextminus":                         1,
	"d128_copy":                         1,
	"copy":                              1,
	"d128_abs":                          1,
	"copyabs":                           1,
	"d128_negate":                       1,
	"copynegate":                        1,
	"d128_ieeeClass":                    1,
	"class":                             1,
	"d128_canonical":                    1,
	"canonical":                         1,
	"d128_exp_ctx":                      1,
	"d128_exp10_ctx":                    1,
	"d128_ln_ctx":                       1,
	"d128_log10_ctx":                    1,
	"d128_lnp1_ctx":                     1,
	"d128_log2p1_ctx":                   1,
	"d128_log10p1_ctx":                  1,
	"d128_expm1_ctx":                    1,
	"d128_exp10m1_ctx":                  1,
	"d128_fitsInt64":                    1,
	"d128_fitsUInt64":                   1,
	"d128_square_ctx":                   1,
	"d128_from_bid64":                   1,
	"d128_from_dpd64":                   1,
	"d128_parse":                        1,
	"apply":                             1,
	"abs":                               1,
	"minus":                             1,
	"tosci":                             1,
	"toeng":                             1,
	"from_int64":                        1,
	"from_uint64":                       1,
	"parse_ctx":                         1,
	"round_integral_mode":               1,
	"to_int64_xmode":                    1,
	// 2 operands
	"d128_add_ctx":                         2,
	"add":                                  2,
	"d128_subtract_ctx":                    2,
	"subtract":                             2,
	"d128_multiply_ctx":                    2,
	"multiply":                             2,
	"d128_divide_ctx":                      2,
	"divide":                               2,
	"d128_compare_ctx":                     2,
	"compare":                              2,
	"d128_compareSignaling_ctx":            2,
	"comparesig":                           2,
	"d128_isEqual_ctx":                     2,
	"d128_isLess_ctx":                      2,
	"d128_isGreater_ctx":                   2,
	"d128_isGreaterEqual_ctx":              2,
	"d128_isGreaterUnordered_ctx":          2,
	"d128_isLessEqual_ctx":                 2,
	"d128_isLessUnordered_ctx":             2,
	"d128_isNotEqual_ctx":                  2,
	"d128_isNotGreater_ctx":                2,
	"d128_isNotLess_ctx":                   2,
	"d128_isOrdered_ctx":                   2,
	"d128_isUnordered_ctx":                 2,
	"d128_isEqualSignaling_ctx":            2,
	"d128_isGreaterSignaling_ctx":          2,
	"d128_isGreaterEqualSignaling_ctx":     2,
	"d128_isGreaterUnorderedSignaling_ctx": 2,
	"d128_isLessSignaling_ctx":             2,
	"d128_isLessEqualSignaling_ctx":        2,
	"d128_isLessUnorderedSignaling_ctx":    2,
	"d128_isNotEqualSignaling_ctx":         2,
	"d128_isNotGreaterSignaling_ctx":       2,
	"d128_isNotLessSignaling_ctx":          2,
	"d128_isOrderedSignaling_ctx":          2,
	"d128_isUnorderedSignaling_ctx":        2,
	"d128_totalOrderMag":                   2,
	"d128_divideInteger_ctx":               2,
	"divideint":                            2,
	"d128_remainderTrunc_ctx":              2,
	"remainder":                            2,
	"d128_remainderNear_ctx":               2,
	"remaindernear":                        2,
	"d128_quantize_ctxnull":                2,
	"quantize":                             2,
	"d128_scaleB_ctxnull":                  2,
	"scaleb":                               2,
	"d128_isSameQuantum":                   2,
	"samequantum":                          2,
	"d128_minimum_ctx":                     2,
	"d128_maximum_ctx":                     2,
	"d128_minimumMagnitude_ctx":            2,
	"d128_maximumMagnitude_ctx":            2,
	"d128_minimumNumber_ctx":               2,
	"min":                                  2,
	"d128_maximumNumber_ctx":               2,
	"max":                                  2,
	"d128_minimumMagnitudeNumber_ctx":      2,
	"minmag":                               2,
	"d128_maximumMagnitudeNumber_ctx":      2,
	"maxmag":                               2,
	"d128_totalOrder":                      2,
	"comparetotal":                         2,
	"comparetotmag":                        2,
	"d128_copySign":                        2,
	"copysign":                             2,
	"d128_pow_ctx":                         2,
	"d128_pown_ctx":                        2,
	"d128_rootn_ctx":                       2,
	"d128_compound_ctx":                    2,
	// 3 operands
	"d128_fma_ctx": 3,
	"fma":          3,
}

// fitOperands applies the R-7 operand-count rule. Too few operands is always an
// error. Intel's own runner (readtest.c) splits a line by token count and passes
// each function only the operands it takes, so an Intel line may carry extra
// operands the op ignores (18 nextup/nextdown lines do): those are trimmed here.
// Every other source must match exactly.
func fitOperands(c *rosettaCase) bool {
	n, ok := rosettaArity[c.operation]
	if !ok {
		return false
	}
	if c.src == srcIntel && len(c.operands) > n {
		c.operands = c.operands[:n]
	}
	return len(c.operands) == n
}

// MARK: - corpus token scans (before any vector filter or skip list)

func scanLines(t *testing.T, path string, each func(line string)) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot read corpus file %s: %v", path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		each(sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
}

func intelCorpusTokens(t *testing.T) map[string]bool {
	set := map[string]bool{}
	scanLines(t, filepath.Join(resourcesDir(), "intel", "readtest.in"), func(line string) {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		if f := strings.Fields(line); len(f) > 0 && intelInScope(f[0]) {
			set[f[0]] = true
		}
	})
	return set
}

func dectestCorpusTokens(t *testing.T) map[string]bool {
	set := map[string]bool{}
	for _, name := range dectestFiles {
		scanLines(t, filepath.Join(resourcesDir(), "dectest", name), func(line string) {
			if i := strings.Index(line, "--"); i >= 0 {
				line = line[:i]
			}
			if !strings.Contains(line, "->") {
				return
			}
			if f := strings.Fields(line); len(f) > 1 {
				set[strings.ToLower(f[1])] = true
			}
		})
	}
	return set
}

func fptestCorpusTokens(t *testing.T) map[string]bool {
	set := map[string]bool{}
	for _, name := range fptestFiles {
		scanLines(t, filepath.Join(resourcesDir(), "fptest", name), func(line string) {
			if f := strings.Fields(line); len(f) > 0 && strings.HasPrefix(f[0], "d128") {
				set[f[0][len("d128"):]] = true
			}
		})
	}
	return set
}

// MARK: - coverage meta-tests

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// partitionErrors is the §9 check: no key in both maps, no corpus token in
// neither, no map key the corpus no longer contains, no exclusion without a
// reason, and a non-empty corpus.
func partitionErrors(source string, corpus map[string]bool, include, exclude map[string]string) []string {
	var errs []string
	if len(corpus) == 0 {
		errs = append(errs, source+": no operators found in the corpus — path or parser broken?")
	}
	both := map[string]bool{}
	for k := range include {
		if _, ok := exclude[k]; ok {
			both[k] = true
		}
	}
	if len(both) != 0 {
		errs = append(errs, source+": contradictory decision for: "+strings.Join(sortedKeys(both), ", "))
	}
	undecided := map[string]bool{}
	for k := range corpus {
		_, inc := include[k]
		_, exc := exclude[k]
		if !inc && !exc {
			undecided[k] = true
		}
	}
	if len(undecided) != 0 {
		errs = append(errs, source+": undecided operator(s) — add to include or exclude:\n  "+strings.Join(sortedKeys(undecided), "\n  "))
	}
	stale := map[string]bool{}
	for _, m := range []map[string]string{include, exclude} {
		for k := range m {
			if !corpus[k] {
				stale[k] = true
			}
		}
	}
	if len(stale) != 0 {
		errs = append(errs, source+": map key(s) not present in the corpus — prune:\n  "+strings.Join(sortedKeys(stale), "\n  "))
	}
	noReason := map[string]bool{}
	for k, why := range exclude {
		if strings.TrimSpace(why) == "" {
			noReason[k] = true
		}
	}
	if len(noReason) != 0 {
		errs = append(errs, source+": exclusion(s) with no reason: "+strings.Join(sortedKeys(noReason), ", "))
	}
	return errs
}

func TestRosettaIntelOperatorCoverage(t *testing.T) {
	if errs := partitionErrors("intel", intelCorpusTokens(t), intelInclude, intelExclude); len(errs) != 0 {
		t.Fatal(strings.Join(errs, "\n"))
	}
}

func TestRosettaDectestOperatorCoverage(t *testing.T) {
	if errs := partitionErrors("dectest", dectestCorpusTokens(t), dectestInclude, dectestExclude); len(errs) != 0 {
		t.Fatal(strings.Join(errs, "\n"))
	}
}

func TestRosettaFptestOperatorCoverage(t *testing.T) {
	if errs := partitionErrors("fptest", fptestCorpusTokens(t), fptestInclude, fptestExclude); len(errs) != 0 {
		t.Fatal(strings.Join(errs, "\n"))
	}
}

// TestRosettaIncludeTargetsHaveArity: every token an include map points at has
// an arity (and so a dispatch arm), so an included operator can't reach the
// runner unrunnable. Go dispatches on strings, so this stands in for Swift's
// compile-time exhaustiveness check.
func TestRosettaIncludeTargetsHaveArity(t *testing.T) {
	for src, m := range map[string]map[string]string{"intel": intelInclude, "dectest": dectestInclude, "fptest": fptestInclude} {
		for k, v := range m {
			if _, ok := rosettaArity[v]; !ok {
				t.Errorf("%s: %s -> %s has no arity", src, k, v)
			}
		}
	}
}
