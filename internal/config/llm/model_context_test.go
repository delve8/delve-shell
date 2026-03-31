package configllm

import "testing"

func TestKnownModelContextTokensByPrefix(t *testing.T) {
	tests := []struct {
		model string
		min   int // at least this many tokens (exact match varies by table)
	}{
		{"gpt-5.4-2026-03-01", 500000},
		{"gpt-5.4-mini-2026-03-01", 350000},
		{"gpt-5.4-nano", 350000},
		{"gpt-5.2", 200000},
		{"gpt-4o-mini", 120000},
		{"claude-sonnet-4-6-20260301", 500000},
		{"anthropic.claude-opus-4-6-v1:0", 500000},
		{"qwen2.5-72b-instruct", 120000},
		{"qwen3-coder-plus-2025-09-23", 500000},
		{"qwen-long-latest", 5000000},
		{"moonshot-v1-8k-foo", 7000},
		{"glm-4-airx", 120000},
		{"glm-5-2026-01-01", 190000},
		{"kimi-k2.5", 250000},
		{"kimi-k2-thinking-turbo", 250000},
		{"deepseek-chat", 120000},
		{"grok-4-1-fast-reasoning", 1000000},
		{"grok-3-beta", 120000},
	}
	for _, tc := range tests {
		n := knownModelContextTokensByPrefix(tc.model)
		if n < tc.min {
			t.Fatalf("%q: got %d, want >= %d", tc.model, n, tc.min)
		}
	}
}
