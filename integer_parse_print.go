// Integer-to-ASCII digit rendering (the print substrate). The renderers fill
// decimal digits into a raw byte pointer via reciprocal-multiply digit extraction
// without bounds checking.

package decimal128

import "unsafe"

// Reciprocal-multiply magic constants for division by powers of 10.
const (
	mU32Div1e1   int64 = 0xCCCCCCCD
	sU32Div1e1         = 35
	mU32Div1e2   int64 = 0x51EB851F
	sU32Div1e2         = 37
	mU64Div1e4   int64 = 0x346DC5D63886594B
	sU64Div1e4Hi       = 11
	// bit pattern 0xABCC77118461CEFD (Kotlin's Long.MIN | 0x2BCC...).
	mU64Div1e8   int64 = -0x543388EE7B9E3103
	sU64Div1e8Hi       = 26
	divisor1e8   int64 = 100_000_000
)

func writeByte(p unsafe.Pointer, off int, b byte) {
	*(*byte)(unsafe.Pointer(uintptr(p) + uintptr(off))) = b
}

func readByte(p unsafe.Pointer, off int) byte {
	return *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(off)))
}

func uIntToASCII(digitPrintCount, u int, ascii unsafe.Pointer, off int) {
	verify("", u >= 0)
	u64ToASCII(digitPrintCount, int64(u), ascii, off)
}

// u64ToASCII renders the low digitPrintCount decimal digits of value into ascii
// starting at off.
func u64ToASCII(digitPrintCount int, value int64, ascii unsafe.Pointer, off int) {
	digitsRemaining := digitPrintCount
	ich := off + digitsRemaining
	t := value
	for digitsRemaining >= 8 {
		q := ushr_64_64(umulHi_64_64x64(t, mU64Div1e8), sU64Div1e8Hi)
		r := t - (q * divisor1e8)
		t = q
		render8DigitsBeforeIndex(r, ascii, ich)
		ich -= 8
		digitsRemaining -= 8
	}
	if digitsRemaining > 0 {
		renderTailDigitsBeforeIndex(digitsRemaining, t, ascii, ich)
	}
}

// u128ToASCII renders the low digitPrintCount decimal digits of a 128-bit value.
func u128ToASCII(digitPrintCount int, value u128, ascii unsafe.Pointer, off int) {
	hi := value.dw1
	lo := value.dw0
	ich := off + digitPrintCount
	remainingDigitCount := digitPrintCount

	const mask32 int64 = 0xFFFF_FFFF

	for hi != 0 {
		q2 := ushr_64_64(umulHi_64_64x64(hi, mU64Div1e8), sU64Div1e8Hi)
		r2 := hi - (q2 * divisor1e8)
		s2 := (r2 << 32) | hi32_64(lo)
		q1 := ushr_64_64(umulHi_64_64x64(s2, mU64Div1e8), sU64Div1e8Hi)
		r1 := s2 - (q1 * divisor1e8)
		s1 := (r1 << 32) | (lo & mask32)
		q0 := ushr_64_64(umulHi_64_64x64(s1, mU64Div1e8), sU64Div1e8Hi)
		r0 := s1 - (q0 * divisor1e8)

		lo = (q1 << 32) + q0
		hi = q2
		render8DigitsBeforeIndex(r0, ascii, ich)
		ich -= 8
		remainingDigitCount -= 8
	}
	for remainingDigitCount >= 8 {
		q := ushr_64_64(umulHi_64_64x64(lo, mU64Div1e8), sU64Div1e8Hi)
		r := lo - (q * divisor1e8)
		lo = q
		render8DigitsBeforeIndex(r, ascii, ich)
		ich -= 8
		remainingDigitCount -= 8
	}
	if remainingDigitCount > 0 {
		renderTailDigitsBeforeIndex(remainingDigitCount, lo, ascii, ich)
	}
}

func render8DigitsBeforeIndex(dw int64, ascii unsafe.Pointer, offMaxx int) {
	abcd := ushr_64_64(umulHi_64_64x64(dw, mU64Div1e4), sU64Div1e4Hi)
	efgh := dw - (abcd * 10000)

	ab := ushr_64_64(abcd*mU32Div1e2, sU32Div1e2)
	cd := abcd - (ab * 100)
	ef := ushr_64_64(efgh*mU32Div1e2, sU32Div1e2)
	gh := efgh - (ef * 100)

	a := ushr_64_64(ab*mU32Div1e1, sU32Div1e1)
	b := ab - (a * 10)
	c := ushr_64_64(cd*mU32Div1e1, sU32Div1e1)
	d := cd - (c * 10)
	e := ushr_64_64(ef*mU32Div1e1, sU32Div1e1)
	f := ef - (e * 10)
	g := ushr_64_64(gh*mU32Div1e1, sU32Div1e1)
	h := gh - (g * 10)

	writeByte(ascii, offMaxx-8, byte(a)+ascii_0)
	writeByte(ascii, offMaxx-7, byte(b)+ascii_0)
	writeByte(ascii, offMaxx-6, byte(c)+ascii_0)
	writeByte(ascii, offMaxx-5, byte(d)+ascii_0)
	writeByte(ascii, offMaxx-4, byte(e)+ascii_0)
	writeByte(ascii, offMaxx-3, byte(f)+ascii_0)
	writeByte(ascii, offMaxx-2, byte(g)+ascii_0)
	writeByte(ascii, offMaxx-1, byte(h)+ascii_0)
}

func renderTailDigitsBeforeIndex(renderDigitCount int, value int64, ascii unsafe.Pointer, offMaxx int) {
	t := value
	remainingDigitCount := renderDigitCount
	ib := offMaxx
	const zero = ascii_0

	for remainingDigitCount >= 4 {
		t0 := ushr_64_64(umulHi_64_64x64(t, mU64Div1e4), sU64Div1e4Hi)
		abcd := t - (t0 * 10000)
		t = t0
		ab := ushr_64_64(abcd*mU32Div1e2, sU32Div1e2)
		cd := abcd - (ab * 100)
		a := ushr_64_64(ab*mU32Div1e1, sU32Div1e1)
		b := ab - (a * 10)
		c := ushr_64_64(cd*mU32Div1e1, sU32Div1e1)
		d := cd - (c * 10)
		writeByte(ascii, ib-4, byte(a)+zero)
		writeByte(ascii, ib-3, byte(b)+zero)
		writeByte(ascii, ib-2, byte(c)+zero)
		writeByte(ascii, ib-1, byte(d)+zero)
		ib -= 4
		remainingDigitCount -= 4
	}
	for remainingDigitCount > 0 {
		divTen := ushr_64_64(t*0xCCCCCCCD, 35)
		digit := byte(t - (divTen * 10))
		ib -= 1
		writeByte(ascii, ib, zero+digit)
		t = divTen
		remainingDigitCount -= 1
	}
}
