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
	if r.Decision != DecisionGuide {
		t.Fatalf("expected guide, got %#v", r)
	}
	r = Evaluate(ChoiceKey3, true, false, 0, 3)
	if r.Decision != DecisionDismiss {
		t.Fatalf("expected dismiss, got %#v", r)
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
	if r.Decision != DecisionGuide {
		t.Fatalf("enter should map to option 2: %#v", r)
	}
	r = Evaluate(teakey.Down, true, false, 0, 3)
	if !r.ChoiceChanged || r.ChoiceIndex != 1 {
		t.Fatalf("down should move selection: %#v", r)
	}
}

func TestEvaluate_fullwidthDigitApprove(t *testing.T) {
	r := Evaluate("１", true, false, 0, 3)
	if r.Decision != DecisionApprove {
		t.Fatalf("fullwidth 1: %#v", r)
	}
}

func TestEvaluate_bracketPastedDigitApprove(t *testing.T) {
	r := Evaluate("[1]", true, false, 0, 3)
	if r.Decision != DecisionApprove {
		t.Fatalf("pasted [1]: %#v", r)
	}
}

func TestEvaluate_shiftEnterConfirmsLikeEnter(t *testing.T) {
	r := Evaluate(teakey.ShiftEnter, true, false, 0, 3)
	if r.Decision != DecisionApprove {
		t.Fatalf("shift+enter: %#v", r)
	}
}

func TestEvaluate_ctrlJConfirmsLikeEnter(t *testing.T) {
	r := Evaluate(teakey.CtrlJ, true, false, 1, 3)
	if r.Decision != DecisionGuide {
		t.Fatalf("ctrl+j: %#v", r)
	}
}

func TestEvaluate_crlfRunesNormalizeToEnter(t *testing.T) {
	r := Evaluate("\r", true, false, 0, 3)
	if r.Decision != DecisionApprove {
		t.Fatalf("\\r: %#v", r)
	}
	r = Evaluate("\n", true, false, 2, 3)
	if r.Decision != DecisionDismiss {
		t.Fatalf("\\n: %#v", r)
	}
}
