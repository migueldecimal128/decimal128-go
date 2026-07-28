// Cross-port SWEPT microbenchmark: decimal128-go averaged over a 4096-input
// operand distribution per op/band (bare `thru`), the headline methodology
// (BenchmarkMatrix.md; www swept cross-port work order). Complements the
// fixed-operand best-case rows in bench_test.go.
//
// Operands come from the shared string corpus decimal128-resources/swept/<profile>/
// (mounted at resources/), parsed ONCE outside the timed loop with the port's
// own MustParse — the SAME strings every port sweeps, so the numbers run over
// byte-identical operands (and identical to what the C bench sweeps, since both
// draw from the same deterministic generators). A band is a relation between the
// two operands: SQ/NQ/OQ/FQ (and P-fin's MIX) feed add AND sub, CP/WP/XP feed mul,
// CD/WD/XD/ET/PT feed div. Ops go through the public Add/Sub/Mul/Quo methods (the
// flag-free `_tte` rung — the same surface the fixed bench times).
//
// The magnitude profile (BenchmarkMatrix.md §2.1) is chosen by the SWEPT_PROFILE
// env var — P-gen (default, all 12 bands), P-fin (financial 64-bit: MIX + CP/WP +
// CD/WD/ET/PT), or P-max (wide-path stress: SQ/OQ/FQ + XP + XD); each profile
// gates which bands are feasible, so the bench sweeps only that profile's set.
//
// Each subtest self-times a MEDIAN over sweptReps fixed-workload sweeps (see
// sweptMedianNS) and publishes it via b.ReportMetric(..,"ns/op") — NOT testing.B's
// auto-scaled mean over one b.Loop run. This matches the zig/rust/swift/JVM manual
// harnesses (fixed-workload samples → median), so the cross-port estimator is
// identical. The operand indexing (bounds-checked slice load) is inherent to a swept
// bench and matches the C harness's d.dx[i] access; -benchtime=1x runs each subtest
// once (our internal reps do the sampling).
//
// Run: cd go && go test -bench='^BenchmarkSwept$' -run='^$' -benchtime=1x

package decimal128

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Manual timing (matching the zig/rust/swift/java/kotlin swept harnesses): sweep a
// fixed workload sweptReps times, each its own clock bracket, and report the MEDIAN
// of the per-rep ns/op samples — NOT testing.B's auto-scaled mean over one run. The
// median is the cross-port estimator: robust to the one-sided right tail of
// scheduler/interrupt contamination without reporting an unachievable best case,
// matching Google Benchmark's median aggregate. Odd rep count ⇒ the median is a true
// single order statistic (no averaging of the two central samples).
const sweptReps = 15

func median(xs []float64) float64 {
	sort.Float64s(xs)
	m := len(xs) / 2
	if len(xs)%2 == 1 {
		return xs[m]
	}
	return (xs[m-1] + xs[m]) / 2
}

// sweptMedianNS runs `sweep` (one full pass over the n-pair band) repeatedly: one
// warmup/calibration sweep sizes the pass count to ~30 ms/measurement, then median
// over sweptReps timed measurements. ns/op = total_ns / (passes * n).
func sweptMedianNS(n int, sweep func()) float64 {
	t0 := time.Now()
	sweep() // warmup + calibrate
	one := time.Since(t0).Nanoseconds()
	if one < 1 {
		one = 1
	}
	passes := int(30_000_000 / one)
	if passes < 1 {
		passes = 1
	}
	samples := make([]float64, sweptReps)
	for r := 0; r < sweptReps; r++ {
		s0 := time.Now()
		for p := 0; p < passes; p++ {
			sweep()
		}
		dt := time.Since(s0).Nanoseconds()
		samples[r] = float64(dt) / float64(passes*n)
	}
	return median(samples)
}

// The band set feasible under each magnitude profile (BenchmarkMatrix.md §2.1;
// see decimal128-resources/swept/README.md). P-gen = all 12; P-fin folds add/sub
// into the single realistic MIX stream and drops the >19-digit XP/XD; P-max keeps
// only the wide-path bands. Add/sub alignment bands are sign-split dataset pairs
// (AddSubSignSplitWorkOrder): <band>_ss (same-sign) / <band>_os (opposite-sign),
// identical magnitudes; each runs on BOTH add and sub so every cell is one path.
// MIX stays blended by design (workload realism).
func profileBands(tb testing.TB, profile string) []string {
	tb.Helper()
	switch profile {
	case "P-gen":
		return []string{"SQ_ss", "SQ_os", "NQ_ss", "NQ_os", "MQ_ss", "MQ_os", "OQ_ss", "OQ_os", "FQ_ss", "FQ_os", "CP", "WP", "XP", "CD", "WD", "XD", "ET", "PT"}
	case "P-fin":
		return []string{"MIX", "CP", "WP", "CD", "WD", "ET", "PT"}
	case "P-max":
		return []string{"SQ_ss", "SQ_os", "OQ_ss", "OQ_os", "FQ_ss", "FQ_os", "XP", "XD"}
	case "FMA":
		// regime-based (3-operand self+lhs*rhs): FN (near ⇒ Barrett), FF (far ⇒ fits-128)
		return []string{"FN", "FF"}
	default:
		tb.Fatalf("unknown SWEPT_PROFILE %q (expect P-gen|P-fin|P-max|FMA)", profile)
		return nil
	}
}

// Load one FMA band file (3 space-separated canonical strings per line: self lhs
// rhs) into parallel operand slices for self + lhs*rhs.
func loadSweptBand3(tb testing.TB, profile, band string) (ss, ls, rs []Decimal128) {
	tb.Helper()
	path := filepath.Join("resources", "swept", profile, band+".txt")
	f, err := os.Open(path)
	if err != nil {
		tb.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			tb.Fatalf("%s: expected 3 operands, got %q", band, line)
		}
		ss = append(ss, MustParse(fields[0]))
		ls = append(ls, MustParse(fields[1]))
		rs = append(rs, MustParse(fields[2]))
	}
	if err := sc.Err(); err != nil {
		tb.Fatalf("scan %s: %v", path, err)
	}
	return ss, ls, rs
}

// Load one band file: resources/swept/<profile>/<band>.txt, parsing each data
// line's two space-separated canonical strings into parallel operand slices.
// Comment (`#`) and blank lines are skipped. Fails the bench on a parse miss so a
// format regression surfaces loudly.
func loadSweptBand(tb testing.TB, profile, band string) (xs, ys []Decimal128) {
	tb.Helper()
	path := filepath.Join("resources", "swept", profile, band+".txt")
	f, err := os.Open(path)
	if err != nil {
		tb.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			tb.Fatalf("%s: expected 2 operands, got %q", band, line)
		}
		xs = append(xs, MustParse(fields[0]))
		ys = append(ys, MustParse(fields[1]))
	}
	if err := sc.Err(); err != nil {
		tb.Fatalf("scan %s: %v", path, err)
	}
	return xs, ys
}

func BenchmarkSwept(b *testing.B) {
	profile := os.Getenv("SWEPT_PROFILE")
	if profile == "" {
		profile = "P-gen"
	}
	// Each band's category (BenchmarkMatrix.md) fixes which op(s) it feeds; the
	// alignment/MIX bands feed both add and sub off one load. Direct method calls
	// (not func values) so cheap bands aren't taxed by call overhead.
	for _, band := range profileBands(b, profile) {
		// FMA bands (FN/FF) are 3-operand (self lhs rhs); handle before loadSweptBand.
		if band == "FN" || band == "FF" {
			ss, ls, rs := loadSweptBand3(b, profile, band)
			n := len(ss)
			b.Run("fma/"+band, func(b *testing.B) {
				b.ReportMetric(sweptMedianNS(n, func() {
					for i := 0; i < n; i++ {
						benchSink = ss[i].FMA(ls[i], rs[i])
					}
				}), "ns/op")
			})
			continue
		}
		xs, ys := loadSweptBand(b, profile, band)
		n := len(xs)
		// Store cat = band file name minus the underscore ("SQ_ss" -> "SQss").
		cat := strings.ReplaceAll(band, "_", "")
		addsub := band == "MIX"
		for _, pfx := range []string{"SQ", "NQ", "MQ", "OQ", "FQ"} {
			if strings.HasPrefix(band, pfx) && len(band) > 2 {
				addsub = true
			}
		}
		switch {
		case addsub:
			b.Run("add/"+cat, func(b *testing.B) {
				b.ReportMetric(sweptMedianNS(n, func() {
					for i := 0; i < n; i++ {
						benchSink = xs[i].Add(ys[i])
					}
				}), "ns/op")
			})
			b.Run("sub/"+cat, func(b *testing.B) {
				b.ReportMetric(sweptMedianNS(n, func() {
					for i := 0; i < n; i++ {
						benchSink = xs[i].Sub(ys[i])
					}
				}), "ns/op")
			})
		case band == "CP" || band == "WP" || band == "XP":
			b.Run("mul/"+cat, func(b *testing.B) {
				b.ReportMetric(sweptMedianNS(n, func() {
					for i := 0; i < n; i++ {
						benchSink = xs[i].Mul(ys[i])
					}
				}), "ns/op")
			})
		case band == "CD" || band == "WD" || band == "XD" || band == "ET" || band == "PT":
			b.Run("div/"+cat, func(b *testing.B) {
				b.ReportMetric(sweptMedianNS(n, func() {
					for i := 0; i < n; i++ {
						benchSink = xs[i].Quo(ys[i])
					}
				}), "ns/op")
			})
		}
	}
}
