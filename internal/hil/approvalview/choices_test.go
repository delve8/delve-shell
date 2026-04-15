package approvalview

import (
	"testing"

	"delve-shell/internal/i18n"
)

func TestChoiceCount(t *testing.T) {
	if got := ChoiceCount(true, false); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
	if got := ChoiceCount(false, true); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
	if got := ChoiceCount(false, false); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestInputPlaceholder(t *testing.T) {
	i18n.SetLang("en")
	if got := InputPlaceholder(true, false); got == "" {
		t.Fatal("expected non-empty placeholder for pending")
	}
	if got := InputPlaceholder(false, true); got == "" {
		t.Fatal("expected non-empty placeholder for sensitive")
	}
	if got := InputPlaceholder(false, false); got == "" {
		t.Fatal("expected default placeholder")
	}
}

func TestChoiceOptionsPendingUsesGuidanceInsteadOfCopy(t *testing.T) {
	i18n.SetLang("en")
	opts := ChoiceOptions(true, false)
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	if opts[1].Label != i18n.T(i18n.KeyChoiceGuide) {
		t.Fatalf("option 2=%q want %q", opts[1].Label, i18n.T(i18n.KeyChoiceGuide))
	}
	if opts[2].Label != i18n.T(i18n.KeyChoiceDismiss) {
		t.Fatalf("option 3=%q want %q", opts[2].Label, i18n.T(i18n.KeyChoiceDismiss))
	}
}
