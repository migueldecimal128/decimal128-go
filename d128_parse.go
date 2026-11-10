// String → decimal128 parsing. Builds and correctly-rounds a finite value
// (needs core finalize), or recognizes Infinity/NaN. d128_parse returns ok=false
// on syntactic failure (Swift's nil); d128_parseOrNaN is best-effort.

package decimal128

type d128ParseResult struct {
	status parseStatus
	value  d128
}

func parseSuccess(value d128) d128ParseResult { return d128ParseResult{psSuccess, value} }
func parseFailure(status parseStatus) d128ParseResult {
	return d128ParseResult{status, d128_QNAN0()}
}

type finiteParseResult struct {
	status  parseStatus
	sign    bool
	qExp    int
	coeff   u128
	residue residue
}

func finiteFailure(status parseStatus) finiteParseResult {
	return finiteParseResult{status, false, 0, u128{0, 0}, RESIDUE_EXACT}
}

// --- parse (boundary API) ---

func d128_parse_ctxnull(s string, ctxnull *context) (d128, bool) {
	t := newTxt(s)
	r := parseDec34_ctxnull(&t, ctxnull)
	if r.status == psSuccess {
		return r.value, true
	}
	verify("", func() bool { return d128_isNAN(r.value) })
	ctxSignalInvalidOperation_ctxnull(ctxnull, causeParseMalformed)
	return d128_QNAN0(), false
}

func d128_parse(s string) (d128, bool)                   { return d128_parse_ctxnull(s, nil) }
func d128_parse_ctx(s string, ctx *context) (d128, bool) { return d128_parse_ctxnull(s, ctx) }

func d128_parseOrNaN(s string) d128 {
	d, ok := d128_parse(s)
	if ok {
		return d
	}
	return d128_QNAN0()
}

// d48_parse parses a D48 exactly: an optional sign, digits with at most one
// '.', an optional exponent; at most 48 significant digits. Panics (impossible)
// on malformed or inexact input (Swift d48_parse). A zero parses to d48_ZERO
// (no −0).
func d48_parse(s string) d48 {
	b := []byte(s)
	i := 0
	sign := false
	if (i < len(b)) && ((b[i] == ascii_plus) || (b[i] == ascii_minus)) {
		sign = b[i] == ascii_minus
		i++
	}
	var coeff u320
	sigDigits := 0
	anyDigit := false
	fracDigits := 0
	hasDot := false
	for i < len(b) {
		ch := b[i]
		if (ch >= ascii_0) && (ch <= ascii_9) {
			anyDigit = true
			if hasDot {
				fracDigits++
			}
			if !coeff.isZero() || (ch != ascii_0) {
				sigDigits++
			}
			if sigDigits > 48 {
				impossible("D48 parse: more than 48 significant digits")
			}
			coeff = add_320_320x64(mulPow10_320(coeff, 1), int64(ch-ascii_0))
		} else if (ch == ascii_dot) && !hasDot {
			hasDot = true
		} else {
			break
		}
		i++
	}
	if !anyDigit {
		impossible("D48 parse error")
	}
	exp := 0
	if (i < len(b)) && ((b[i] == 'e') || (b[i] == 'E')) {
		i++
		expSign := false
		if (i < len(b)) && ((b[i] == ascii_plus) || (b[i] == ascii_minus)) {
			expSign = b[i] == ascii_minus
			i++
		}
		start := i
		for (i < len(b)) && (b[i] >= ascii_0) && (b[i] <= ascii_9) {
			exp = exp*10 + int(b[i]-ascii_0)
			if exp > 1_000_000_000 {
				impossible("D48 parse: exponent out of range")
			}
			i++
		}
		if i == start {
			impossible("D48 parse error")
		}
		if expSign {
			exp = -exp
		}
	}
	if i != len(b) {
		impossible("D48 parse error")
	}
	if coeff.isZero() {
		return d48_ZERO // the internal domain has no −0
	}
	return d48_fromSignQExp_160(sign, exp-fracDigits, coeff.u192())
}

type txt struct {
	b   []byte
	pos int
}

func newTxt(s string) txt { return txt{b: []byte(s)} }

func (t *txt) nextChar() byte {
	if t.pos < len(t.b) {
		ch := t.b[t.pos]
		t.pos++
		return ch
	}
	return 0
}

func (t *txt) nextCharLowercase() byte {
	if t.pos < len(t.b) {
		ch := t.b[t.pos]
		t.pos++
		return ch | 0x20
	}
	return 0
}

func parseDec34_ctxnull(t *txt, ctxnull *context) d128ParseResult {
	chFirst := t.nextChar()
	ch := chFirst
	sign := ch == ascii_minus
	if ch == ascii_plus || ch == ascii_minus {
		ch = t.nextChar()
	}
	if (ch >= ascii_0 && ch <= ascii_9) || ch == ascii_dot {
		r := parseFiniteValueText(sign, ch, t, 34)
		if r.status == psSuccess {
			rounding := ROUND_TIES_TO_EVEN
			if ctxnull != nil {
				rounding = ctxnull.round
			}
			value := finalize_128_ctxnull(r.sign, r.qExp, r.coeff, r.residue, rounding, ctxnull)
			return parseSuccess(value)
		}
		return parseFailure(r.status)
	}
	chLower := ch | 0x20
	if chLower == ascii_i {
		return parseInfinityText(sign, ch, t)
	}
	if chLower == ascii_n || chLower == ascii_s || chLower == ascii_q {
		return parseNanText(sign, ch, t)
	}
	if chFirst == 0 {
		return parseFailure(psEmptyString)
	}
	return parseFailure(psInvalidFormat)
}

var infinityBytes = int64(ascii_i) | (int64(ascii_n) << 8) | (int64(ascii_f) << 16) | (int64(ascii_i) << 24) |
	(int64(ascii_n) << 32) | (int64(ascii_i) << 40) | (int64(ascii_t) << 48) | (int64(ascii_y) << 56)

func parseInfinityText(sign bool, chFirst byte, t *txt) d128ParseResult {
	ch := chFirst | 0x20
	var chPrev byte = 0
	i := 0
	for i < 8 {
		target := byte(ushr_64_64(infinityBytes, i<<3) & 0x7F)
		if ch != target {
			if ch == 0 {
				break
			}
			return parseFailure(psFailureParsingInfinity)
		}
		chPrev = ch
		ch = t.nextCharLowercase()
		i++
	}
	if ch != 0 || (chPrev != ascii_f && chPrev != ascii_y) {
		return parseFailure(psFailureParsingInfinity)
	}
	return parseSuccess(d128_infinity(sign))
}

// Parse "NaN", "qNaN" or "sNaN" (case-insensitive) with an optional decimal
// payload. The payload may be written bare (NaN123) or parenthesised
// (NaN(123)); the two spellings produce bit-identical values.
//
// The parenthesised form is positional and balanced: the '(' must follow the
// word immediately, at least one digit must appear, and the ')' must be the
// last character of the input. So "NaN()", "NaN(1", "NaN(1]", "NaN(1)2(3)"
// and "NaN (1)" are all rejected, and "NaN[1]" / "NaN{1}" are not this form
// at all. The payload alphabet stays decimal digits -- this is a
// paren-tolerant spelling of our own payload syntax, not C strtod's
// NAN(n-char-sequence), which reads letters and underscores too.
//
// Rendering never emits parentheses: NaN(123) prints back as NaN123.
func parseNanText(sign bool, chFirst byte, t *txt) d128ParseResult {
	ch := chFirst | 0x20
	hasS := ch == ascii_s
	hasQ := ch == ascii_q
	if hasS || hasQ {
		ch = t.nextCharLowercase()
	}
	if ch != ascii_n || t.nextCharLowercase() != ascii_a || t.nextCharLowercase() != ascii_n {
		return parseFailure(psFailureParsingNan)
	}
	var accumulator int64 = 0
	var first19accumulator int64 = 0
	remainingDigitCount := 0
	ch = t.nextChar()
	hasParen := ch == ascii_lParen
	if hasParen {
		ch = t.nextChar()
	}
	payloadDigitCount := 0
	// The loop stops ON the first non-digit rather than failing there, so the
	// terminating character is available to the balance check below.
	for ch >= ascii_0 && ch <= ascii_9 {
		if payloadDigitCount < 33 {
			d := ch - ascii_0
			if payloadDigitCount < 19 {
				accumulator = accumulator*10 + int64(d)
			} else if payloadDigitCount == 19 {
				first19accumulator = accumulator
				accumulator = int64(d)
				remainingDigitCount = 1
			} else {
				accumulator = accumulator*10 + int64(d)
				remainingDigitCount++
			}
		} else {
			accumulator = 0
			first19accumulator = 0
			remainingDigitCount = 0
		}
		payloadDigitCount++
		ch = t.nextChar()
	}
	if hasParen {
		// at least one digit, and the ')' must be the last character of the input
		if payloadDigitCount == 0 || ch != ascii_rParen || t.nextChar() != 0 {
			return parseFailure(psFailureParsingNan)
		}
	} else if ch != 0 {
		// unchanged: a bare payload runs to end of input
		return parseFailure(psFailureParsingNan)
	}
	payload := u128{0, 0}
	if first19accumulator == 0 {
		payload.dw0 = accumulator
		payload.dw1 = 0
	} else {
		payload = mulPow10_128_128(first19accumulator, 0, remainingDigitCount)
		payload = add_128_128x128(payload.dw0, payload.dw1, accumulator, 0)
	}
	return parseSuccess(d128_nan_64x64(sign, hasS, payload.dw0, payload.dw1))
}

func parseFiniteValueText(sign bool, chFirst byte, t *txt, maxDigitLen int) finiteParseResult {
	res := RESIDUE_EXACT
	hasCoefficientDigit := false
	significantDigitCount := 0
	hasDot := false
	expSign := false

	ch := chFirst
	verify("", func() bool { return ch != 0 })

	var chPrev byte = 0
	fractionalDigitCount := 0
	exp := 0
	var accumulator int64 = 0
	var first19accumulator int64 = 0
	remainingDigitCount := 0

	for (ch >= ascii_0 && ch <= ascii_9) || ch == ascii_dot || ch == ascii_underbar {
		if ch >= ascii_0 && ch <= ascii_9 {
			d := ch - ascii_0
			hasCoefficientDigit = true
			inc := 1
			if significantDigitCount+int(d) == 0 {
				inc = 0
			}
			significantDigitCount += inc
			if significantDigitCount < maxDigitLen+1 {
				if significantDigitCount < 20 {
					accumulator = accumulator*10 + int64(d)
				} else if significantDigitCount == 20 {
					first19accumulator = accumulator
					accumulator = int64(d)
					remainingDigitCount = 1
				} else {
					accumulator = accumulator*10 + int64(d)
					remainingDigitCount++
				}
			} else if significantDigitCount == maxDigitLen+1 {
				res = residueFromDecimalDigit(int(d))
			} else {
				res = residueMerge(res, residueFromDecimalDigit(int(d)))
			}
			if hasDot {
				fractionalDigitCount++
			}
		} else if ch == ascii_dot {
			if hasDot {
				return finiteFailure(psDoubleDot)
			}
			if chPrev == ascii_underbar {
				return finiteFailure(psInvalidUnderscoreLocation)
			}
			hasDot = true
		} else if ch == ascii_underbar {
			if chPrev < ascii_0 || chPrev > ascii_9 {
				return finiteFailure(psInvalidUnderscoreLocation)
			}
		} else {
			impossible("cannot be here")
		}
		chPrev = ch
		ch = t.nextChar()
	}
	if !hasCoefficientDigit {
		return finiteFailure(psNoCoefficientDigit)
	}
	if chPrev == ascii_underbar {
		return finiteFailure(psInvalidUnderscoreLocation)
	}
	if (ch | 0x20) == ascii_e {
		if chPrev == ascii_underbar {
			return finiteFailure(psInvalidUnderscoreLocation)
		}
		chPrev = ch
		ch = t.nextChar()
		if ch == ascii_minus || ch == ascii_plus {
			expSign = ch == ascii_minus
			ch = t.nextChar()
		}
		hasExpDigit := false
		var expSignificantDigitCount int64 = 0
		for (ch >= ascii_0 && ch <= ascii_9) || ch == ascii_underbar {
			if ch >= ascii_0 && ch <= ascii_9 {
				hasExpDigit = true
				eDigit := ch - ascii_0
				if (expSignificantDigitCount | int64(eDigit)) != 0 {
					expSignificantDigitCount++
				}
				exp = exp*10 + int(ch-ascii_0)
			} else {
				if chPrev < ascii_0 || chPrev > ascii_9 {
					return finiteFailure(psInvalidUnderscoreLocation)
				}
			}
			chPrev = ch
			ch = t.nextChar()
		}
		if chPrev == ascii_underbar {
			return finiteFailure(psInvalidUnderscoreLocation)
		}
		if !hasExpDigit {
			return finiteFailure(psNoExponentDigit)
		}
		if expSignificantDigitCount > 4 {
			exp = 9999
		}
	}
	if ch != 0 {
		return finiteFailure(psUnexpectedChar)
	}

	signedExp := exp
	if expSign {
		signedExp = -exp
	}
	qExp := signedExp - fractionalDigitCount + max(0, significantDigitCount-maxDigitLen)

	coeff := u128{0, 0}
	if first19accumulator == 0 {
		coeff.dw0 = accumulator
		coeff.dw1 = 0
	} else {
		coeff = mulPow10_128_128(first19accumulator, 0, remainingDigitCount)
		coeff = add_128_128x128(coeff.dw0, coeff.dw1, accumulator, 0)
	}

	return finiteParseResult{psSuccess, sign, qExp, coeff, res}
}
