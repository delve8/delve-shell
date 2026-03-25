package configsvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/llmtest"
)

// LLMTester is used to check LLM connectivity.
// It is injected in tests; production uses llmtest.TestConnection.
type LLMTester func(ctx context.Context, baseURL, apiKey, model string) error

func defaultLLMTester(ctx context.Context, baseURL, apiKey, model string) error {
	return llmtest.TestConnection(ctx, baseURL, apiKey, model)
}

// LoadOrDefault loads config.yaml; if not found/invalid, returns config.Default().
func LoadOrDefault() *config.Config {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return config.Default()
	}
	return cfg
}

type SaveLLMParams struct {
	BaseURL     string
	APIKey      string
	Model       string
	MaxMessages string // empty means 0
	MaxChars    string // empty means 0
}

// SaveLLMFromOverlay validates and writes the LLM config fields.
// It does not run connectivity checks; caller should call CheckLLMAndMaybeAutoCorrect.
func SaveLLMFromOverlay(p SaveLLMParams) error {
	baseURL := strings.TrimSpace(p.BaseURL)
	apiKey := strings.TrimSpace(p.APIKey)
	model := strings.TrimSpace(p.Model)
	if model == "" {
		return fmt.Errorf("llm.model is required")
	}

	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = config.Default()
		if err := config.EnsureRootDir(); err != nil {
			return err
		}
	}

	cfg.LLM.BaseURL = baseURL
	cfg.LLM.APIKey = apiKey
	cfg.LLM.Model = model

	if s := strings.TrimSpace(p.MaxMessages); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			cfg.LLM.MaxContextMessages = n
		}
	} else {
		cfg.LLM.MaxContextMessages = 0
	}
	if s := strings.TrimSpace(p.MaxChars); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			cfg.LLM.MaxContextChars = n
		}
	} else {
		cfg.LLM.MaxContextChars = 0
	}

	return config.Write(cfg)
}

// CheckLLMAndMaybeAutoCorrect checks LLM connectivity using resolved config.
// If it fails and base_url does not end with /v1, it retries with /v1 and writes back on success.
// Returns correctedBaseURL when auto-correction happened.
func CheckLLMAndMaybeAutoCorrect(ctx context.Context, tester LLMTester) (correctedBaseURL string, err error) {
	if tester == nil {
		tester = defaultLLMTester
	}
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return "", err
	}
	resolvedBaseURL, resolvedAPIKey, resolvedModel := cfg.LLMResolved()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx1, cancel1 := context.WithTimeout(ctx, 15*time.Second)
	defer cancel1()
	checkErr := tester(ctx1, resolvedBaseURL, resolvedAPIKey, resolvedModel)
	if checkErr == nil {
		return "", nil
	}
	if resolvedBaseURL == "" || strings.HasSuffix(resolvedBaseURL, "/v1") {
		return "", checkErr
	}

	tryURL := resolvedBaseURL + "/v1"
	ctx2, cancel2 := context.WithTimeout(ctx, 15*time.Second)
	defer cancel2()
	retryErr := tester(ctx2, tryURL, resolvedAPIKey, resolvedModel)
	if retryErr != nil {
		return "", checkErr
	}

	cfg.LLM.BaseURL = tryURL
	if writeErr := config.Write(cfg); writeErr != nil {
		return "", checkErr
	}
	return tryURL, nil
}
