package configsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"delve-shell/internal/config"
)

func TestCheckLLMAndMaybeAutoCorrect_AppendsV1OnRetrySuccess(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)

	cfg := config.Default()
	cfg.LLM.BaseURL = "https://example.com"
	cfg.LLM.APIKey = "x"
	cfg.LLM.Model = "m"
	if err := config.Write(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}

	calls := 0
	tester := func(ctx context.Context, baseURL, apiKey, model string) error {
		_ = ctx
		calls++
		if baseURL == "https://example.com/v1" {
			return nil
		}
		return os.ErrInvalid
	}

	corrected, err := CheckLLMAndMaybeAutoCorrect(context.Background(), tester)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if corrected != "https://example.com/v1" {
		t.Fatalf("expected corrected base url, got %q", corrected)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}

	// Ensure config written back.
	cfg2, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg2.LLM.BaseURL != "https://example.com/v1" {
		t.Fatalf("expected base_url updated, got %q", cfg2.LLM.BaseURL)
	}
	// Ensure config path exists under root.
	if _, err := os.Stat(filepath.Join(root, "config.yaml")); err != nil {
		t.Fatalf("expected config.yaml exists: %v", err)
	}
}

func TestSaveLLMFromOverlay_RequiresModel(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	err := SaveLLMFromOverlay(SaveLLMParams{BaseURL: "x", APIKey: "y", Model: ""})
	if err == nil {
		t.Fatalf("expected error for empty model")
	}
}
