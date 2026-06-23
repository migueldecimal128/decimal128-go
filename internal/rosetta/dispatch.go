package rosetta

// Outcome is the result of running one Case through the harness.
type Outcome int

const (
	// Pass: the engine reproduced the expected value and flags exactly.
	Pass Outcome = iota
	// Fail: the engine ran but produced a mismatching value or flags.
	Fail
	// Unsupported: no dispatch arm is wired for the Case's operation yet. In
	// Phase 0 every Case is Unsupported. Unsupported cases are not failures —
	// they record coverage still to be earned as operators land.
	Unsupported
)

func (o Outcome) String() string {
	switch o {
	case Pass:
		return "PASS"
	case Fail:
		return "FAIL"
	case Unsupported:
		return "UNSUPPORTED"
	default:
		return "?"
	}
}

// Dispatch runs one Case against the engine and reports its Outcome plus a short
// diagnostic. It is the single canonical dispatch point; per-operation arms
// (keyed on Case.Operation) are added as each operator lands in Phases 2–6.
//
// Phase 0: no arms are wired, so every Case is Unsupported. A future invariant
// (mirroring the Rust harness) is that the native corpus admits no Unsupported —
// a missing arm there becomes a hard error rather than a silent skip — but that
// is enforced once the operator set is in place.
func Dispatch(c *Case) (Outcome, string) {
	_ = c
	return Unsupported, "no dispatch arm wired (phase 0)"
}
