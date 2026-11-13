// SPDX-License-Identifier: MIT
// Result digests (PortParityCheckWorkOrder.md, check C1). Every core computes
// the digests in resources/rosetta/digests.tsv from the same generated inputs:
// a hash of the BID128 result and the exception flags of every call. A
// difference in any result, in any port, changes the digest. The generator,
// operands and hash are specified in resources/rosetta/digests.md.
//
// Lines whose digest is "-" are computed and printed, for recording. Set
// PARITY_DIGEST_RECORD=1 to print every line instead of checking it.

package decimal128

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// digestRng is SplitMix64, as in the u320 test and digests.md.
type digestRng struct {
	state uint64
}

// digestMix is the SplitMix64 output function.
func digestMix(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

func digestNext(rng *digestRng) uint64 {
	rng.state += 0x9E3779B97F4A7C15
	return digestMix(rng.state)
}

// digestDecimalString is a decimal operand string (digests.md, "Operands").
func digestDecimalString(rng *digestRng) string {
	c := digestNext(rng) % 16
	if c == 0 {
		k := digestNext(rng) % 6
		if k == 0 {
			return "NaN"
		}
		if k == 1 {
			return "-NaN"
		}
		if k == 2 {
			return "sNaN"
		}
		if k == 3 {
			return "Inf"
		}
		if k == 4 {
			return "-Inf"
		}
		s := digestNext(rng) % 2
		e := int(digestNext(rng)%41) - 20
		sign := ""
		if s == 1 {
			sign = "-"
		}
		return sign + "0E" + strconv.Itoa(e)
	}
	s := digestNext(rng) % 2
	n := 1 + int(digestNext(rng)%34)
	var b strings.Builder
	if s == 1 {
		b.WriteByte('-')
	}
	i := 0
	for i < n {
		b.WriteByte(byte('0' + digestNext(rng)%10))
		i += 1
	}
	q := 0
	if c <= 2 {
		q = int(digestNext(rng)%12288) - 6176
	} else {
		e := int(digestNext(rng)%41) - 20
		q = e - (n - 1)
	}
	b.WriteString("E" + strconv.Itoa(q))
	return b.String()
}

func digestDecimal(rng *digestRng) d128 {
	text := digestDecimalString(rng)
	x, ok := d128_parse(text)
	if !ok {
		panic("digest operand did not parse: " + text)
	}
	return x
}

// digestInteger is an integer operand in [-1000, 1000].
func digestInteger(rng *digestRng) int {
	return int(digestNext(rng)%2001) - 1000
}

// digestCall runs one call of op, drawing its operands in argument order.
func digestCall(op string, rng *digestRng, mode round, ctx *context) d128 {
	if op == "sqrt" {
		return d128_sqrt_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "exp" {
		return d128_exp_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "exp10" {
		return d128_exp10_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "expm1" {
		return d128_expm1_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "exp10m1" {
		return d128_exp10m1_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "ln" {
		return d128_ln_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "lnp1" {
		return d128_lnp1_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "log10" {
		return d128_log10_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "log10p1" {
		return d128_log10p1_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if op == "log2p1" {
		return d128_log2p1_rnd_ctxnull(digestDecimal(rng), mode, ctx)
	}
	if (op == "compound") || (op == "pown") || (op == "rootn") {
		x := digestDecimal(rng)
		n := digestInteger(rng)
		if op == "compound" {
			return d128_compound_rnd_ctxnull(x, n, mode, ctx)
		}
		if op == "pown" {
			return d128_pown_rnd_ctxnull(x, n, mode, ctx)
		}
		return d128_rootn_rnd_ctxnull(x, n, mode, ctx)
	}
	x := digestDecimal(rng)
	y := digestDecimal(rng)
	if op == "add" {
		return d128_add_rnd_ctxnull(x, y, mode, ctx)
	}
	if op == "subtract" {
		return d128_subtract_rnd_ctxnull(x, y, mode, ctx)
	}
	if op == "multiply" {
		return d128_multiply_rnd_ctxnull(x, y, mode, ctx)
	}
	if op == "divide" {
		return d128_divide_rnd_ctxnull(x, y, mode, ctx)
	}
	if op == "pow" {
		return d128_pow_rnd_ctxnull(x, y, mode, ctx)
	}
	if op == "quantize" {
		return d128_quantize_rnd_ctxnull(x, y, mode, ctx)
	}
	if op == "fma" {
		z := digestDecimal(rng)
		return d128_fma_rnd_ctxnull(x, y, z, mode, ctx)
	}
	panic("digests.tsv: unknown op " + op)
}

var digestModes = [5]round{
	const_ROUND_TIES_TO_EVEN,
	const_ROUND_TIES_TO_AWAY,
	const_ROUND_TOWARD_ZERO,
	const_ROUND_TOWARD_POSITIVE,
	const_ROUND_TOWARD_NEGATIVE,
}

// digestLine is the digest of one table line (digests.md, "Digest").
func digestLine(op string, mode int, count int, seed uint64) uint64 {
	rng := digestRng{state: seed}
	h := uint64(0)
	i := 0
	for i < count {
		ctx := newContext()
		r := d128_to_bid128(digestCall(op, &rng, digestModes[mode], ctx))
		lo := uint64(r.dw0)
		hi := uint64(r.dw1)
		f := uint64(ctx.exceptionFlagsInt())
		h = digestMix(h ^ lo)
		h = digestMix(h ^ hi)
		h = digestMix(h ^ f)
		i += 1
	}
	return h
}

func TestResultDigests(t *testing.T) {
	record := os.Getenv("PARITY_DIGEST_RECORD") == "1"
	data, err := os.ReadFile(filepath.Join(resourcesDir(), "digests.tsv"))
	if err != nil {
		t.Fatalf("cannot read digests.tsv: %v", err)
	}
	checked := 0
	for _, line := range strings.Split(string(data), "\n") {
		if (line == "") || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 5 {
			t.Errorf("digests.tsv: bad line %s", line)
			continue
		}
		mode, _ := strconv.Atoi(f[1])
		count, _ := strconv.Atoi(f[2])
		seed, _ := strconv.ParseUint(f[3], 10, 64)
		got := fmt.Sprintf("%016x", digestLine(f[0], mode, count, seed))
		if record || (f[4] == "-") {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n", f[0], f[1], f[2], f[3], got)
			continue
		}
		if got != f[4] {
			t.Errorf("digest %s mode %s: expected %s, got %s", f[0], f[1], f[4], got)
		}
		checked += 1
	}
	if !record && (checked == 0) {
		t.Errorf("digests.tsv: no recorded digests")
	}
}
