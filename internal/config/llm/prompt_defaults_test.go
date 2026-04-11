package configllm

import (
	"strings"
	"testing"
)

func TestDefaultSystemPrompt_StrongHostMemoryGuidance(t *testing.T) {
	checks := []string{
		"Treat host memory maintenance as a default online workflow, not an optional extra",
		"The main goal is to remember what this machine is, what it is for, and what it can reliably do",
		"Use update_host_memory for stable, reusable facts: machine role, responsibilities, capabilities",
		"Record durable capabilities and responsibilities when you can infer them with reasonable confidence",
		"Trigger update_host_memory proactively when command output shows strong evidence",
		"prefer to call update_host_memory before your final answer for that turn",
	}
	for _, want := range checks {
		if !strings.Contains(DefaultSystemPrompt, want) {
			t.Fatalf("DefaultSystemPrompt missing %q", want)
		}
	}
}
