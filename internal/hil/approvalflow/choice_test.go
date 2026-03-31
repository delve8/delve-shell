package approvalflow

import (
	"testing"

	"delve-shell/internal/teakey"
)

func TestEvaluatePendingThreeOptions(t *testing.T) {
	r := Evaluate(ChoiceKey1, true, false, 0, 3)
	if !r.Handled || r.Decision != DecisionApprove {
		t.Fatalf("unexpected result: %#v", r)
	}
	r = Evaluate("2", true, false, 0, 3)
	if r.Decision != DecisionDismiss {
		t.Fatalf("expected dismiss, got %#v", r)
	}
	r = Evaluate(ChoiceKey3, true, false, 0, 3)
	if r.Decision != DecisionCopy {
		t.Fatalf("expected copy, got %#v", r)
	}
}

func TestEvaluateSensitive(t *testing.T) {
	r := Evaluate(ChoiceKey3, false, true, 0, 3)
	if r.Decision != DecisionSensitiveRunNoStore {
		t.Fatalf("unexpected sensitive decision: %#v", r)
	}
}

func TestEvaluateEnterAndArrow(t *testing.T) {
	r := Evaluate(teakey.Enter, true, false, 1, 3)
	if r.Decision != DecisionDismiss {
		t.Fatalf("enter should map to option 2: %#v", r)
	}
	r = Evaluate(teakey.Down, true, false, 0, 3)
	if !r.ChoiceChanged || r.ChoiceIndex != 1 {
		t.Fatalf("down should move selection: %#v", r)
	}
}
