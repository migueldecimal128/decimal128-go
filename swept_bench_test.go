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
// Each timed b.Loop() body is ONE op indexed by a counter cycling through the
// 4096 pairs, so Go's reported ns/op is the swept per-op average directly. The
// operand indexing (bounds-checked slice load) is inherent to a swept bench and
// matches the C harness's d.dx[i] access.
//
// Run: cd go && go test -bench='^BenchmarkSwept$' -run='^$'   (add -count=N for reps)

package decimal128

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The band set feasible under each magnitude profile (BenchmarkMatrix.md §2.1;
// see decimal128-resources/swept/README.md). P-gen = all 12; P-fin folds add/sub
// into the single realistic MIX stream and drops the >19-digit XP/XD; P-max keeps
// only the wide-path bands.
func profileBands(tb testing.TB, profile string) []string {
	tb.Helper()
	switch profile {
	case "P-gen":
		return []string{"SQ", "NQ", "MQ", "OQ", "FQ", "CP", "WP", "XP", "CD", "WD", "XD", "ET", "PT"}
	case "P-fin":
		return []string{"MIX", "CP", "WP", "CD", "WD", "ET", "PT"}
	case "P-max":
		return []string{"SQ", "OQ", "FQ", "XP", "XD"}
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
				i := 0
				for b.Loop() {
					benchSink = ss[i].AddProduct(ls[i], rs[i])
					if i++; i == n {
						i = 0
					}
				}
			})
			continue
		}
		xs, ys := loadSweptBand(b, profile, band)
		n := len(xs)
		switch band {
		case "SQ", "NQ", "MQ", "OQ", "FQ", "MIX":
			b.Run("add/"+band, func(b *testing.B) {
				i := 0
				for b.Loop() {
					benchSink = xs[i].Add(ys[i])
					if i++; i == n {
						i = 0
					}
				}
			})
			b.Run("sub/"+band, func(b *testing.B) {
				i := 0
				for b.Loop() {
					benchSink = xs[i].Sub(ys[i])
					if i++; i == n {
						i = 0
					}
				}
			})
		case "CP", "WP", "XP":
			b.Run("mul/"+band, func(b *testing.B) {
				i := 0
				for b.Loop() {
					benchSink = xs[i].Mul(ys[i])
					if i++; i == n {
						i = 0
					}
				}
			})
		case "CD", "WD", "XD", "ET", "PT":
			b.Run("div/"+band, func(b *testing.B) {
				i := 0
				for b.Loop() {
					benchSink = xs[i].Quo(ys[i])
					if i++; i == n {
						i = 0
					}
				}
			})
		}
	}
}
