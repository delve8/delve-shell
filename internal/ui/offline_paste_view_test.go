package ui

import (
	"strings"
	"testing"
)

func TestOfflineCommandReviewTextBreaksLongSingleLineCompoundCommand(t *testing.T) {
	command := `kubectl get pods -A --no-headers | awk '$4 != "Running" {print $1, $2, $4}' | sort -u && kubectl get nodes -o wide --no-headers | head -n 20`

	got := offlineCommandReviewText(command, 80)
	if !strings.Contains(got, "|\n  awk") {
		t.Fatalf("expected pipeline break before awk, got %q", got)
	}
	if !strings.Contains(got, "&&\n  kubectl get nodes") {
		t.Fatalf("expected chain break before second kubectl, got %q", got)
	}
	if strings.Contains(got, `"$4 !=`) {
		t.Fatalf("should not break quoted awk source, got %q", got)
	}
}

func TestOfflineCommandReviewTextLeavesExistingMultilineCommand(t *testing.T) {
	command := "kubectl get pods -A |\n  head -n 20"
	if got := offlineCommandReviewText(command, 80); got != command {
		t.Fatalf("got %q want unchanged %q", got, command)
	}
}
