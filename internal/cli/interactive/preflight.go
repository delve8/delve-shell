package interactive

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"delve-shell/internal/config"
	configllm "delve-shell/internal/config/llm"
	"delve-shell/internal/history"
)

// PreflightResult holds early startup outputs shared by the interactive loop.
type PreflightResult struct {
	Config          *config.Config
	NeedConfigModel bool
	RulesText       string
	InitialSession  *history.Session
}

// RunPreflight ensures config root, optional history prune, loads rules, and allocates the first session.
func RunPreflight() (*PreflightResult, error) {
	if err := config.EnsureRootDir(); err != nil {
		return nil, err
	}
	cfg, _ := config.LoadEnsured()
	needConfigModel := NeedsConfigModelOverlay(cfg)
	warmLLMStreamMode(cfg)

	if cfg != nil {
		if err := history.Prune(cfg); err != nil {
			log.Printf("[warn] history prune: %v", err)
		}
	}
	rulesText, err := config.LoadRules()
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}
	initialSession, err := history.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &PreflightResult{
		Config:          cfg,
		NeedConfigModel: needConfigModel,
		RulesText:       rulesText,
		InitialSession:  initialSession,
	}, nil
}

// NeedsConfigModelOverlay reports whether the first layout should open the model config overlay.
func NeedsConfigModelOverlay(cfg *config.Config) bool {
	return cfg == nil || strings.TrimSpace(cfg.LLM.Model) == ""
}

func warmLLMStreamMode(cfg *config.Config) {
	if cfg == nil || NeedsConfigModelOverlay(cfg) {
		return
	}
	baseURL, apiKey, model := cfg.LLMResolved()
	if !configllm.ShouldWarmResolvedConfig(baseURL, apiKey, model) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := configllm.WarmResolvedConfigStreamMode(ctx, baseURL, apiKey, model); err != nil {
		log.Printf("[warn] llm stream mode warmup skipped: %v", err)
	}
}
