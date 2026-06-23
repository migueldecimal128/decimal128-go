// The 128-bit reciprocal-multiply divPow10 (1..38): x / 10^pow10 with a
// classified residue, via a single 128×128→256 reciprocal multiply. 10^k =
// 2^(k-1) · (2 · 5^k): strip k-1 low bits (tracked as sticky), multiply by the
// tabulated 5^k reciprocal M, then shift by the tabulated S and read the round
// and sticky bits to classify the residue.

package decimal128

// rmp10_128_m[pow10] (lo/hi dwords) is the 128-bit reciprocal multiplier; index 0
// is unused (pow10 is 1-based).
var rmp10_128_m_lo = [39]int64{
	0, -3689348814741910323, -8116567392432202711, -5607810198407703691, -796899343984252629,
	7059937891890119595, -5966710051105796727, 8681362423783412279, -433607690457090823,
	3342462662376237665, -2352363749791415257, 6908224879525537595, 1837231088878519753,
	-2219563943639094521, -4577174392197548131, 7406306930467692465, -1104043714324185915,
	-5455818757660607787, 6287533877951699089, -5705333293747935177, 4930221287926428315,
	-6359384767086891331, -5087507813669513065, 5343694504016015421, -1551871013135504155,
	-310374202627100831, -7626996991585501311, 2163949416424810061, 1731159533139848049,
	-5993770002971942207, -2211334375271286885, 6494163879375305893, -6729281241546290057,
	-3388152356990243445, -677630471398048689, -542104377118438951, 6511330626094318325,
	-8463514193788003639, -5382051653499511051,
}

var rmp10_128_m_hi = [39]int64{
	0, -3689348814741910324, 2951479051793528258, 4722366482869645213, 3777893186295716170,
	1511157274518286468, 302231454903657293, 120892581961462917, 96714065569170333,
	77371252455336267, 30948500982134506, 6189700196426901, 4951760157141521, 3961408125713216,
	1584563250285286, 1267650600228229, 126765060022822, 202824096036516, 40564819207303,
	129807421463370, 3245185536584, 10384593717069, 8307674973655, 3323069989462, 1329227995784,
	265845599156, 212676479325, 42535295865, 34028236692, 27222589353, 43556142965, 17422457186,
	1742245718, 2787593149, 557518629, 446014903, 713623846, 285449538, 57089907,
}

// rmp10_128_s128[pow10] = S - 128 (the post-multiply shift in the high half).
var rmp10_128_s128 = [39]int{
	0, 2, 2, 5, 7, 8, 8, 9, 11, 13, 14, 14, 16, 18, 19, 21, 20, 23, 23, 27, 24, 28, 30, 31, 32,
	32, 34, 34, 36, 38, 41, 42, 41, 44, 44, 46, 49, 50, 50,
}

func divPow10_q128res_128_recipmul(xDw0, xDw1 int64, pow10 int) quot128Residue {
	demand("recipmul: pow10 outside [1, 38]", pow10 > 0 && pow10 <= 38)
	k := pow10
	mDw0 := rmp10_128_m_lo[pow10]
	mDw1 := rmp10_128_m_hi[pow10]
	s128 := rmp10_128_s128[pow10]

	// Strip k-1 low bits (the 2^(k-1) factor of 10^k); track them as sticky.
	dividendAlfa := shr_128_128(xDw0, xDw1, k-1)
	stickyAlfa := false
	if k-1 > 0 {
		one := shl_128_128(1, 0, k-1)
		alfaMask := sub_128_128x128(one.dw0, one.dw1, 1, 0)
		masked := and_128_128x128(xDw0, xDw1, alfaMask.dw0, alfaMask.dw1)
		stickyAlfa = !isZero_bool_128(masked.dw0, masked.dw1)
	}

	// Multiply the stripped dividend by the 5^k reciprocal -> 256-bit product.
	p := mul_256_128x128(dividendAlfa.dw0, dividendAlfa.dw1, mDw0, mDw1)
	pHi := p.hi()
	pLo := p.lo()

	// quotPlusRound = prod >> S = pHi >> s128.
	quotPlusRound := shr_128_128(pHi.dw0, pHi.dw1, s128)
	q := shr_128_128(quotPlusRound.dw0, quotPlusRound.dw1, 1)
	roundBit := (quotPlusRound.dw0 & 1) != 0

	// stickyBeta = (prod mod 2^S != 0) || (pLo >= M), no 256-bit compare.
	stickyBeta := false
	if s128 > 0 {
		one := shl_128_128(1, 0, s128)
		fracMask := sub_128_128x128(one.dw0, one.dw1, 1, 0)
		highFrac := and_128_128x128(pHi.dw0, pHi.dw1, fracMask.dw0, fracMask.dw1)
		stickyBeta = !isZero_bool_128(highFrac.dw0, highFrac.dw1)
	}
	if !stickyBeta && cmp_int_128x128(pLo.dw0, pLo.dw1, mDw0, mDw1) >= 0 {
		stickyBeta = true
	}

	sticky := stickyAlfa || stickyBeta

	res := RESIDUE_EXACT
	if roundBit && sticky {
		res = RESIDUE_GT_HALF
	} else if roundBit {
		res = RESIDUE_HALF
	} else if sticky {
		res = RESIDUE_LT_HALF
	}
	return quot128Residue{q.dw0, q.dw1, res}
}
