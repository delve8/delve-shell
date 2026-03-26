package configllm

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/config"
)

// LLMTester is used to check LLM connectivity.
// It is injected in tests; production uses TestConnection.
type LLMTester func(ctx context.Context, baseURL, apiKey, model string) error

// SaveLLMFromOverlay validates and writes the LLM config fields.
// It does not run connectivity checks; caller should call CheckLLMAndMaybeAutoCorrect.
func SaveLLMFromOverlay(baseURL, apiKey, model, maxMessages, maxChars string) error {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
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

	if s := strings.TrimSpace(maxMessages); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			cfg.LLM.MaxContextMessages = n
		}
	} else {
		cfg.LLM.MaxContextMessages = 0
	}
	if s := strings.TrimSpace(maxChars); s != "" {
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
		tester = TestConnection
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

// TestConnection sends a minimal "hello" request to the LLM and returns nil if the response is received.
// baseURL, apiKey, model are used as-is after env expansion and trim; empty baseURL is left empty (client default).
func TestConnection(ctx context.Context, baseURL, apiKey, model string) error {
	baseURL = strings.TrimSpace(config.ExpandEnv(baseURL))
	baseURL = strings.TrimRight(baseURL, "/")
	apiKey = strings.TrimSpace(config.ExpandEnv(apiKey))
	model = strings.TrimSpace(config.ExpandEnv(model))
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" && apiKey != "" {
		baseURL = "https://api.openai.com/v1"
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	timeout := 15 * time.Second
	if d, ok := ctx.Deadline(); ok {
		timeout = time.Until(d)
		if timeout <= 0 {
			return context.DeadlineExceeded
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return err
	}
	_, err = chatModel.Generate(ctx, []*schema.Message{
		schema.UserMessage("hello"),
	})
	return err
}
