package verify

import "testing"

func TestDemandPassesWhenConditionHolds(t *testing.T) {
	Demand("must not fire", true)
}

func TestDemandAbortsWhenConditionFails(t *testing.T) {
	defer func() {
		if r := recover(); r != "boom" {
			t.Fatalf("expected panic %q, got %v", "boom", r)
		}
	}()
	Demand("boom", false)
	t.Fatal("Demand did not abort")
}

func TestDemandEmptyMessageRendersDefault(t *testing.T) {
	defer func() {
		if r := recover(); r != "demand failed" {
			t.Fatalf("expected %q, got %v", "demand failed", r)
		}
	}()
	Demand("", false)
	t.Fatal("Demand did not abort")
}

func TestImpossibleAlwaysAborts(t *testing.T) {
	defer func() {
		if r := recover(); r != "unreachable here" {
			t.Fatalf("expected %q, got %v", "unreachable here", r)
		}
	}()
	Impossible("unreachable here")
	t.Fatal("Impossible returned")
}

func TestVerifyIsNoopWhenDisabled(t *testing.T) {
	if Enabled {
		t.Skip("verify is enabled; failure path is covered only when disabled")
	}
	// cond is false, but with Enabled == false Verify must not abort.
	Verify("should not fire while disabled", false)
}
