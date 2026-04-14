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
		"A single execute_command may still be a readable multi-line shell script or pipeline",
	}
	for _, want := range checks {
		if !strings.Contains(DefaultSystemPrompt, want) {
			t.Fatalf("DefaultSystemPrompt missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"update_host_memory",
		"Treat host memory maintenance as a default online workflow",
		`"cmd1 && cmd2 && cmd3"`,
	} {
		if strings.Contains(DefaultSystemPrompt, unwanted) {
			t.Fatalf("DefaultSystemPrompt should not contain %q", unwanted)
		}
	}
}

func TestOfflineManualRelayAppend_RequestsReadableMultilineCommands(t *testing.T) {
	for _, want := range []string{
		"Prefer one combined shell command or pipeline per execute_command",
		"one multi-line command string with real newline characters",
		`trailing \ for long pipelines or argument lists`,
		"only keep it on one line when it is genuinely short and easy to review",
	} {
		if !strings.Contains(OfflineManualRelayAppend, want) {
			t.Fatalf("OfflineManualRelayAppend missing %q", want)
		}
	}
}
