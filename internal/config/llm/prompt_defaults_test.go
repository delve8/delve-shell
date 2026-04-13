package configllm

import (
	"strings"
	"testing"
)

func TestDefaultSystemPrompt_HostMemoryIsReadOnlyGuidance(t *testing.T) {
	checks := []string{
		"read persistent host memory with view_host_memory when needed",
		`The system message may include a "Host memory" block. Treat it as a useful prior, not a guarantee`,
		"Recent session history is authoritative for the current conversation",
		"Host memory is maintained outside the main conversation from persisted session history",
	}
	for _, want := range checks {
		if !strings.Contains(DefaultSystemPrompt, want) {
			t.Fatalf("DefaultSystemPrompt missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"update_host_memory",
		"Treat host memory maintenance as a default online workflow",
	} {
		if strings.Contains(DefaultSystemPrompt, unwanted) {
			t.Fatalf("DefaultSystemPrompt should not contain %q", unwanted)
		}
	}
}
