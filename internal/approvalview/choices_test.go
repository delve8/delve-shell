package approvalview

import "testing"

func TestChoiceCount(t *testing.T) {
	if got := ChoiceCount(true, false, true); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if got := ChoiceCount(true, false, false); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
	if got := ChoiceCount(false, true, true); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
	if got := ChoiceCount(false, false, true); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestInputPlaceholder(t *testing.T) {
	if got := InputPlaceholder("en", true, false, true); got == "" {
		t.Fatal("expected non-empty placeholder for pending")
	}
	if got := InputPlaceholder("en", false, true, true); got == "" {
		t.Fatal("expected non-empty placeholder for sensitive")
	}
	if got := InputPlaceholder("en", false, false, true); got == "" {
		t.Fatal("expected default placeholder")
	}
}
