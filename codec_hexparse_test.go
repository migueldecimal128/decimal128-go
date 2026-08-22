package decimal128

// Test-corpus operand hex parsers. These decode the Intel BID readtest and IBM
// dectest DPD hex dialects used only by the test corpora (codec_test.go,
// rosetta_harness_test.go, wrapper_rosetta_test.go); they are not part of the
// core interchange codec surface, so they live in the test harness.

// --- Intel BID test-vector hex parser ---
//
// Accepts "[hi16,lo16]" (35 chars) or "[hex32]" (34 chars). Returns ok=false on
// malformed input. The result's dw1 holds the BID high word, dw0 the low word.

func bid128ParseIntelHex(s string) (u128, bool) {
	n := len(s)
	if n != 34 && n != 35 {
		return u128{}, false
	}
	chars := []byte(s)
	if chars[0] != '[' || chars[n-1] != ']' {
		return u128{}, false
	}
	var hiStart, loStart int
	if n == 35 {
		if chars[17] != ',' {
			return u128{}, false
		}
		hiStart = 1
		loStart = 18
	} else {
		hiStart = 1
		loStart = 17
	}
	hi, ok := parseHex16(chars, hiStart)
	if !ok {
		return u128{}, false
	}
	lo, ok := parseHex16(chars, loStart)
	if !ok {
		return u128{}, false
	}
	return fromLoHi_128_64x64(lo, hi), true
}

// dpd128ParseDecTestHex parses an IBM dectest "#" + 32-hex-digit DPD pattern.
func dpd128ParseDecTestHex(s string) (u128, bool) {
	if len(s) != 33 {
		return u128{}, false
	}
	chars := []byte(s)
	if chars[0] != '#' {
		return u128{}, false
	}
	hi, ok := parseHex16(chars, 1)
	if !ok {
		return u128{}, false
	}
	lo, ok := parseHex16(chars, 17)
	if !ok {
		return u128{}, false
	}
	return fromLoHi_128_64x64(lo, hi), true
}

func parseHex16(chars []byte, off int) (int64, bool) {
	var dw int64 = 0
	i := 0
	for i < 16 {
		ch := chars[off+i]
		var nib int64
		if ch >= '0' && ch <= '9' {
			nib = int64(ch - '0')
		} else if ch >= 'A' && ch <= 'F' {
			nib = int64(ch - 'A' + 10)
		} else if ch >= 'a' && ch <= 'f' {
			nib = int64(ch - 'a' + 10)
		} else {
			return 0, false
		}
		dw = (dw << 4) | nib
		i += 1
	}
	return dw, true
}
