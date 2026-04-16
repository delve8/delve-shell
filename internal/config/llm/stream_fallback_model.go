package configllm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type streamMode uint8

const (
	streamModeUnknown streamMode = iota
	streamModeGenerate
	streamModePreferStream
)

var llmStreamModeCache sync.Map

// NewToolCallingChatModel builds an OpenAI-compatible chat model and falls back to
// streaming only when a successful non-stream response is effectively empty.
func NewToolCallingChatModel(ctx context.Context, cfg *openaimodel.ChatModelConfig) (model.ToolCallingChatModel, error) {
	base, err := openaimodel.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return wrapToolCallingChatModelWithStreamFallback(base, streamModeCacheKey(chatModelConfigKeyParts(cfg))), nil
}

// WrapToolCallingChatModelWithStreamFallback makes Generate retry via Stream when
// the non-stream response succeeds but returns no usable content.
func WrapToolCallingChatModelWithStreamFallback(base model.ToolCallingChatModel) model.ToolCallingChatModel {
	return wrapToolCallingChatModelWithStreamFallback(base, "")
}

func wrapToolCallingChatModelWithStreamFallback(base model.ToolCallingChatModel, modeKey string) model.ToolCallingChatModel {
	if base == nil {
		return nil
	}
	if wrapped, ok := base.(*streamFallbackChatModel); ok {
		if modeKey != "" {
			wrapped.modeKey = modeKey
		}
		return wrapped
	}
	return &streamFallbackChatModel{base: base, modeKey: modeKey}
}

type streamFallbackChatModel struct {
	base    model.ToolCallingChatModel
	modeKey string
}

func (m *streamFallbackChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.currentMode() == streamModePreferStream {
		return m.generateFromStream(ctx, input, opts...)
	}

	msg, err := m.base.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	if !emptyAssistantResponse(msg) {
		m.rememberMode(streamModeGenerate)
		return msg, nil
	}

	streamMsg, err := m.generateFromStream(ctx, input, opts...)
	if err != nil {
		return nil, fmt.Errorf("non-stream response was empty; stream fallback failed: %w", err)
	}
	if emptyAssistantResponse(streamMsg) {
		return msg, nil
	}
	m.rememberMode(streamModePreferStream)
	return streamMsg, nil
}

func (m *streamFallbackChatModel) generateFromStream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	stream, err := m.base.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	streamMsg, err := collectStreamMessage(stream)
	if err != nil {
		return nil, err
	}
	return streamMsg, nil
}

func (m *streamFallbackChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m.base.Stream(ctx, input, opts...)
}

func (m *streamFallbackChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	next, err := m.base.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return wrapToolCallingChatModelWithStreamFallback(next, m.modeKey), nil
}

func collectStreamMessage(stream *schema.StreamReader[*schema.Message]) (*schema.Message, error) {
	var parts []*schema.Message
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if chunk == nil {
			continue
		}
		parts = append(parts, chunk)
	}
	switch len(parts) {
	case 0:
		return nil, nil
	case 1:
		return parts[0], nil
	default:
		return schema.ConcatMessages(parts)
	}
}

func emptyAssistantResponse(msg *schema.Message) bool {
	if msg == nil {
		return true
	}
	if strings.TrimSpace(msg.Content) != "" {
		return false
	}
	if strings.TrimSpace(msg.ReasoningContent) != "" {
		return false
	}
	if len(msg.ToolCalls) > 0 {
		return false
	}
	if len(msg.AssistantGenMultiContent) > 0 {
		return false
	}
	if len(msg.MultiContent) > 0 {
		return false
	}
	return true
}

func (m *streamFallbackChatModel) currentMode() streamMode {
	if m == nil || m.modeKey == "" {
		return streamModeUnknown
	}
	return loadStreamMode(m.modeKey)
}

func (m *streamFallbackChatModel) rememberMode(mode streamMode) {
	if m == nil || m.modeKey == "" || mode == streamModeUnknown {
		return
	}
	storeStreamMode(m.modeKey, mode)
}

func loadStreamMode(key string) streamMode {
	if key == "" {
		return streamModeUnknown
	}
	v, ok := llmStreamModeCache.Load(key)
	if !ok {
		return streamModeUnknown
	}
	mode, _ := v.(streamMode)
	return mode
}

func storeStreamMode(key string, mode streamMode) {
	if key == "" || mode == streamModeUnknown {
		return
	}
	llmStreamModeCache.Store(key, mode)
}

func WarmResolvedConfigStreamMode(ctx context.Context, baseURL, apiKey, modelName string) error {
	baseURL, apiKey, modelName = normalizeModelConfig(baseURL, apiKey, modelName)
	if strings.TrimSpace(modelName) == "" {
		return nil
	}
	key := streamModeCacheKey(baseURL, apiKey, modelName)
	if mode := loadStreamMode(key); mode == streamModeGenerate || mode == streamModePreferStream {
		return nil
	}
	timeout := modelWarmupTimeout(ctx)
	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      modelName,
		HTTPClient: NewLLMHTTPClient(timeout),
	})
	if err != nil {
		return err
	}
	msg, err := chatModel.Generate(ctx, []*schema.Message{schema.UserMessage("hello")})
	if err != nil {
		return err
	}
	if !emptyAssistantResponse(msg) {
		storeStreamMode(key, streamModeGenerate)
		return nil
	}
	stream, err := chatModel.Stream(ctx, []*schema.Message{schema.UserMessage("hello")})
	if err != nil {
		return fmt.Errorf("non-stream response was empty; stream probe failed: %w", err)
	}
	defer stream.Close()
	streamMsg, err := collectStreamMessage(stream)
	if err != nil {
		return fmt.Errorf("non-stream response was empty; stream probe failed: %w", err)
	}
	if emptyAssistantResponse(streamMsg) {
		return fmt.Errorf("received empty response from model")
	}
	storeStreamMode(key, streamModePreferStream)
	return nil
}

func ShouldWarmResolvedConfig(baseURL, apiKey, modelName string) bool {
	baseURL, apiKey, modelName = normalizeModelConfig(baseURL, apiKey, modelName)
	if strings.TrimSpace(modelName) == "" {
		return false
	}
	return strings.TrimSpace(baseURL) != "" || strings.TrimSpace(apiKey) != ""
}

func normalizeModelConfig(baseURL, apiKey, modelName string) (string, string, string) {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")
	apiKey = strings.TrimSpace(apiKey)
	modelName = strings.TrimSpace(modelName)
	if baseURL == "" && apiKey != "" {
		baseURL = "https://api.openai.com/v1"
	}
	return baseURL, apiKey, modelName
}

func chatModelConfigKeyParts(cfg *openaimodel.ChatModelConfig) (string, string, string) {
	if cfg == nil {
		return "", "", ""
	}
	return normalizeModelConfig(cfg.BaseURL, cfg.APIKey, cfg.Model)
}

func streamModeCacheKey(baseURL, apiKey, modelName string) string {
	baseURL, apiKey, modelName = normalizeModelConfig(baseURL, apiKey, modelName)
	sum := sha256.Sum256([]byte(baseURL + "\x00" + apiKey + "\x00" + modelName))
	return hex.EncodeToString(sum[:])
}

func modelWarmupTimeout(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d > 0 {
			return d
		}
	}
	return 15 * time.Second
}
