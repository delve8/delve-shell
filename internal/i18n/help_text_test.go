package i18n

import (
	"strings"
	"testing"
)

func TestEnglishHelpText_UsesExpectedSectionsAndOrder(t *testing.T) {
	got := englishHelpText()
	if strings.Contains(got, "<name>") || strings.Contains(got, "<cmd>") {
		t.Fatalf("help text must avoid raw angle-bracket placeholders")
	}
	order := []string{
		"### /help",
		"### /config",
		"### /access",
		"### /new",
		"### /history",
		"### /skill",
		"### /quit",
	}
	last := -1
	for _, marker := range order {
		idx := strings.Index(got, marker)
		if idx < 0 {
			t.Fatalf("missing section %q", marker)
		}
		if idx <= last {
			t.Fatalf("section %q out of order", marker)
		}
		last = idx
	}
}

func TestEnglishHelpText_UsesDetailedCopyDistinctFromSlashRows(t *testing.T) {
	got := englishHelpText()
	snippets := []string{
		"Switch the execution target.",
		"Use an installed skill for the current turn.",
		"Browse and switch sessions.",
		"Configure model settings.",
	}
	for _, snippet := range snippets {
		if !strings.Contains(got, snippet) {
			t.Fatalf("help text missing %q", snippet)
		}
	}
}
