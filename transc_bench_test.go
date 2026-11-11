package decimal128

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Transcendental benchmark (D48GoWorkOrder.md G-12; Swift Decimal128Transc,
// C benchmark/transc, Rust benches/transc.rs): exp, exp10, ln, log10, pow,
// pown n > 0, pown n < 0, compound (CompoundD48OnePlusXWorkOrder.md) and lnp1 / log2p1 /
// log10p1 / expm1 / exp10m1 (LogP1ExpM1WorkOrder.md), each over its 64-input pool from the shared
// resources/d48/bench_pool.txt, ties-to-even through the wrapper's flag-free
// methods. Each iteration is one call; the pool is cycled in file order
// (realistic branch history), so ns/op is the mean over the pool.
//
// The op-benchmark suite doesn't cover these functions (BenchmarkMatrix.md),
// so this is where the D48 cost ceiling (2× D38 per op) is measured.
//
// Run: go test -run '^$' -bench Transc -count 3

const transcPoolSize = 64

type transcArg struct {
	x, y Decimal128
	n    int
}

// loadTranscPool returns the pool for one op: its `op x [y|n]` lines in file
// order.
func loadTranscPool(b *testing.B, op string) []transcArg {
	b.Helper()
	f, err := os.Open(filepath.Join("resources", "d48", "bench_pool.txt"))
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	var pool []transcArg
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] != op {
			continue
		}
		var a transcArg
		if a.x, err = Parse(fields[1]); err != nil {
			b.Fatalf("bad operand: %s", line)
		}
		a.y = a.x
		if len(fields) > 2 {
			if op == "pow" {
				if a.y, err = Parse(fields[2]); err != nil {
					b.Fatalf("bad operand: %s", line)
				}
			} else if a.n, err = strconv.Atoi(fields[2]); err != nil {
				b.Fatalf("bad n: %s", line)
			}
		}
		pool = append(pool, a)
	}
	if err := sc.Err(); err != nil {
		b.Fatal(err)
	}
	if len(pool) != transcPoolSize {
		b.Fatalf("%s: %d inputs, want %d", op, len(pool), transcPoolSize)
	}
	return pool
}

func BenchmarkTransc(b *testing.B) {
	ops := []struct {
		name string
		fn   func(transcArg) Decimal128
	}{
		{"exp", func(a transcArg) Decimal128 { return a.x.Exp() }},
		{"exp10", func(a transcArg) Decimal128 { return a.x.Exp10() }},
		{"ln", func(a transcArg) Decimal128 { return a.x.Ln() }},
		{"log10", func(a transcArg) Decimal128 { return a.x.Log10() }},
		{"pow", func(a transcArg) Decimal128 { return a.x.Pow(a.y) }},
		{"pown_pos", func(a transcArg) Decimal128 { return a.x.PowInt(a.n) }},
		{"pown_neg", func(a transcArg) Decimal128 { return a.x.PowInt(a.n) }},
		{"compound", func(a transcArg) Decimal128 { return a.x.Compound(a.n) }},
		{"lnp1", func(a transcArg) Decimal128 { return a.x.LnP1() }},
		{"log2p1", func(a transcArg) Decimal128 { return a.x.Log2P1() }},
		{"log10p1", func(a transcArg) Decimal128 { return a.x.Log10P1() }},
		{"expm1", func(a transcArg) Decimal128 { return a.x.ExpM1() }},
		{"exp10m1", func(a transcArg) Decimal128 { return a.x.Exp10M1() }},
	}
	for _, op := range ops {
		b.Run(op.name, func(b *testing.B) {
			pool := loadTranscPool(b, op.name)
			fn := op.fn
			var sink Decimal128
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sink = fn(pool[i&(transcPoolSize-1)])
			}
			benchSink = sink
		})
	}
}
