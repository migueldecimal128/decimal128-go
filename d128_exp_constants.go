// D38 constants for the exp / exp10 Padé kernels. The numerator weights are the
// gen_pade.c output (raw limbs + pre-packed qExpAndSignBit); the Q denominator
// weights are P(−z) — the odd-index weights negated (sign bit flipped). Each
// numerator entry's trailing comment is the 38-digit decimal it encodes.

package decimal128

// d38c builds a D38 from a raw (qExpAndSignBit, lo, hi) triple. The uint64 limbs
// sidestep Go's signed-constant-overflow rule for the high-bit-set words.
func d38c(qExpAndSignBit int32, lo, hi uint64) d38 {
	return d38{int64(lo), int64(hi), qExpAndSignBit}
}

// u256c builds a u256 from four uint64 limbs (dw0 = least significant).
func u256c(dw0, dw1, dw2, dw3 uint64) u256 {
	return u256{int64(dw0), int64(dw1), int64(dw2), int64(dw3)}
}

var d38_EIGHT = d38c(0, 0x8, 0) // 8

// d38_LN10 — ln(10) to 38 significant digits, used by the exp range reduction
// n = round(x / ln10). (Defined here for exp; shared with the log kernels.)
var d38_LN10 = d38c(-74, 0x00f92d091faa68dc, 0x11529e19baaed959) // 2.3025850929940456840179914546843642076

// ln(10) to 50 significant digits as a U256 coefficient × 10^-49. The range
// reduction r = x − n·ln10 cancels ~5 leading digits of x, so a 38-digit ln10
// would lose precision; this carries enough guard digits.
var u256_LN10_50_COEFF = u256c(15733924570658476943, 10197472284333201977, 67666894227, 0)

const LN10_50_QEXP = -49

// [9,9] Padé numerator weights for exp(z).
var padeExpPWeights = []d38{
	d38c(0, 0x1, 0),  // 1
	d38c(-2, 0x5, 0), // 0.5
	d38c(-76, 0x3d5b8b8f0f0f0f0f, 0x08d9ccc8831ee9f0), // 0.11764705882352941176470588235294117647
	d38c(-78, 0x597ad630a0a0a0a1, 0x0ce84aa469e26a7e), // 0.017156862745098039215686274509803921569
	d38c(-80, 0x597ad630a0a0a0a1, 0x0ce84aa469e26a7e), // 0.0017156862745098039215686274509803921569
	d38c(-82, 0x3fea06b505050505, 0x09383550ddeade5a), // 0.00012254901960784313725490196078431372549
	d38c(-86, 0x3aa301932d6f132d, 0x2f47a1d336f6185f), // 0.0000062845651080945198592257415786827551533
	d38c(-88, 0xb9835bfdb4cc3db5, 0x10e2b9cb6f0ebf8f), // 2.2444875386051856640091934209581268405E-7
	d38c(-92, 0xbce4b9cc3dd02f27, 0x266060714238caa3), // 5.1011080422845128727481668658139246375E-9
	d38c(-96, 0x0ac5403844ae6d47, 0x2aa3f9616606367d), // 5.6678978247605698586090742953488051527E-11
}

// Q(z) = P(−z): odd-index weights negated.
var padeExpQWeights = []d38{
	d38c(0, 0x1, 0),
	d38c(-1, 0x5, 0),
	d38c(-76, 0x3d5b8b8f0f0f0f0f, 0x08d9ccc8831ee9f0),
	d38c(-77, 0x597ad630a0a0a0a1, 0x0ce84aa469e26a7e),
	d38c(-80, 0x597ad630a0a0a0a1, 0x0ce84aa469e26a7e),
	d38c(-81, 0x3fea06b505050505, 0x09383550ddeade5a),
	d38c(-86, 0x3aa301932d6f132d, 0x2f47a1d336f6185f),
	d38c(-87, 0xb9835bfdb4cc3db5, 0x10e2b9cb6f0ebf8f),
	d38c(-92, 0xbce4b9cc3dd02f27, 0x266060714238caa3),
	d38c(-95, 0x0ac5403844ae6d47, 0x2aa3f9616606367d),
}

// [9,9] Padé numerator weights for 10^z.
var padeExp10PWeights = []d38{
	d38c(0, 0x1, 0), // 1
	d38c(-74, 0x807c96848fd5346e, 0x08a94f0cdd576cac), // 1.1512925464970228420089957273421821038
	d38c(-76, 0x21cc2113a7fd8d45, 0x2eed0a571e8de5b0), // 0.62375271887981153065431369277576678725
	d38c(-76, 0xf4e9363a62617bf3, 0x0fc1e71d7bcd64d7), // 0.20945220803021078496789615208818637811
	d38c(-78, 0x537b20eabe312f13, 0x2448678953f745ea), // 0.048228153190505110248516490071932743443
	d38c(-80, 0x560b758714d33fc3, 0x3bacac2d9d34d9d1), // 0.0079321018999350207382307581722084196291
	d38c(-82, 0xfd639a8b96c97b33, 0x4676e43cf59484ef), // 0.00093663279953333980407088694976192281395
	d38c(-84, 0xf0199ba9f1621e42, 0x39f250b64ef98cc1), // 0.000077024168636241020775343936951782874690
	d38c(-86, 0x966b9d1214d97312, 0x1e530442c4685593), // 0.0000040307886932288201762295127255197250322
	d38c(-88, 0x53324cc0c47f6de9, 0x07c21ca062c8cf5b), // 0.00000010312482175597367540252796938795052521
}

var padeExp10QWeights = []d38{
	d38c(0, 0x1, 0),
	d38c(-73, 0x807c96848fd5346e, 0x08a94f0cdd576cac),
	d38c(-76, 0x21cc2113a7fd8d45, 0x2eed0a571e8de5b0),
	d38c(-75, 0xf4e9363a62617bf3, 0x0fc1e71d7bcd64d7),
	d38c(-78, 0x537b20eabe312f13, 0x2448678953f745ea),
	d38c(-79, 0x560b758714d33fc3, 0x3bacac2d9d34d9d1),
	d38c(-82, 0xfd639a8b96c97b33, 0x4676e43cf59484ef),
	d38c(-83, 0xf0199ba9f1621e42, 0x39f250b64ef98cc1),
	d38c(-86, 0x966b9d1214d97312, 0x1e530442c4685593),
	d38c(-87, 0x53324cc0c47f6de9, 0x07c21ca062c8cf5b),
}
