package decimal128

import "testing"

func TestDemandPassesWhenConditionHolds(t *testing.T) {
	demand("must not fire", true)
}

func TestDemandAbortsWhenConditionFails(t *testing.T) {
	defer func() {
		if r := recover(); r != "boom" {
			t.Fatalf("expected panic %q, got %v", "boom", r)
		}
	}()
	demand("boom", false)
	t.Fatal("demand did not abort")
}

func TestDemandEmptyMessageRendersDefault(t *testing.T) {
	defer func() {
		if r := recover(); r != "demand failed" {
			t.Fatalf("expected %q, got %v", "demand failed", r)
		}
	}()
	demand("", false)
	t.Fatal("demand did not abort")
}

func TestImpossibleAlwaysAborts(t *testing.T) {
	defer func() {
		if r := recover(); r != "unreachable here" {
			t.Fatalf("expected %q, got %v", "unreachable here", r)
		}
	}()
	impossible("unreachable here")
	t.Fatal("impossible returned")
}

func TestVerifyIsNoopWhenDisabled(t *testing.T) {
	if verifyEnabled {
		t.Skip("verify is enabled; the failure path is covered only when disabled")
	}
	verify("should not fire while disabled", func() bool { return false })
}
