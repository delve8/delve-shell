package configllm

import "testing"

func TestKnownModelContextTokensByPrefix(t *testing.T) {
	tests := []struct {
		model string
		min   int // at least this many tokens (exact match varies by table)
	}{
		{"gpt-5.4-2026-03-01", 500000},
		{"gpt-5.2", 200000},
		{"gpt-4o-mini", 120000},
		{"qwen2.5-72b-instruct", 120000},
		{"qwen-long-latest", 5000000},
		{"moonshot-v1-8k-foo", 7000},
		{"glm-4-airx", 120000},
		{"deepseek-chat", 60000},
		{"grok-3-beta", 120000},
	}
	for _, tc := range tests {
		n := knownModelContextTokensByPrefix(tc.model)
		if n < tc.min {
			t.Fatalf("%q: got %d, want >= %d", tc.model, n, tc.min)
		}
	}
}
