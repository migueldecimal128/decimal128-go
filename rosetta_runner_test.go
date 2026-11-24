package decimal128

// Rosetta — the core runner (RosettaHarnessAlignmentWorkOrder.md). One test per
// source; each runs every file the shared data selects (files.index, or
// native.index minus native.exclude), logs the ROSETTA report lines, and checks
// every file's counts against resources/rosetta/expected.tsv. Green = no
// failures and every count exactly as expected.

import "testing"

func TestRosettaRunnerRosettaCoreDectest(t *testing.T) { rosettaSource(t, "core", "dectest", dispatch) }
func TestRosettaRunnerRosettaCoreFptest(t *testing.T)  { rosettaSource(t, "core", "fptest", dispatch) }
func TestRosettaRunnerRosettaCoreIntel(t *testing.T)   { rosettaSource(t, "core", "intel", dispatch) }

// The Cairo vectors (8.9M cases): opt-in, ROSETTA_CAIRO=1
// (CairoVectorsWorkOrder.md D-3). Without it the test skips.
func TestRosettaRunnerRosettaCoreCairo(t *testing.T) {
	cairoEnabled(t)
	rosettaSource(t, "core", "cairo", dispatch)
}

// The native files run are exactly expected.tsv's native rows.
func TestRosettaRunnerRosettaCoreNativeSelection(t *testing.T) { rosettaNativeSelection(t, "core") }

// Every native file, each checked against its own row.
func TestRosettaRunnerRosettaCoreNative(t *testing.T) { rosettaNative(t, "core", dispatch) }
