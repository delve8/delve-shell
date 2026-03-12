package llmtest

import (
	"context"
	"strings"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/config"
)

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
