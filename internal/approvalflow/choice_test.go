package approvalflow

import "testing"

func TestEvaluatePendingTwoOptions(t *testing.T) {
	r := Evaluate("1", true, false, true, 0, 2)
	if !r.Handled || r.Decision != DecisionApprove {
		t.Fatalf("unexpected result: %#v", r)
	}
	r = Evaluate("2", true, false, true, 0, 2)
	if r.Decision != DecisionReject {
		t.Fatalf("expected reject, got %#v", r)
	}
}

func TestEvaluatePendingThreeOptions(t *testing.T) {
	r := Evaluate("2", true, false, false, 0, 3)
	if r.Decision != DecisionCopy {
		t.Fatalf("expected copy, got %#v", r)
	}
	r = Evaluate("3", true, false, false, 0, 3)
	if r.Decision != DecisionDismiss {
		t.Fatalf("expected dismiss, got %#v", r)
	}
}

func TestEvaluateSensitive(t *testing.T) {
	r := Evaluate("3", false, true, true, 0, 3)
	if r.Decision != DecisionSensitiveRunNoStore {
		t.Fatalf("unexpected sensitive decision: %#v", r)
	}
}

func TestEvaluateEnterAndArrow(t *testing.T) {
	r := Evaluate("enter", true, false, true, 1, 2)
	if r.Decision != DecisionReject {
		t.Fatalf("enter should map to option 2: %#v", r)
	}
	r = Evaluate("down", true, false, true, 0, 2)
	if !r.ChoiceChanged || r.ChoiceIndex != 1 {
		t.Fatalf("down should move selection: %#v", r)
	}
}
