package configllm

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestStreamFallbackChatModelGenerate(t *testing.T) {
	t.Run("keeps non stream response when content exists", func(t *testing.T) {
		base := &fakeToolCallingChatModel{
			generateMsg: schema.AssistantMessage("hello", nil),
			streamMsg:   schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("ignored", nil)}),
		}

		got, err := WrapToolCallingChatModelWithStreamFallback(base).Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if got == nil || got.Content != "hello" {
			t.Fatalf("Generate() content = %#v, want hello", got)
		}
		if base.streamCalls != 0 {
			t.Fatalf("streamCalls = %d, want 0", base.streamCalls)
		}
	})

	t.Run("falls back to stream when non stream response is empty", func(t *testing.T) {
		base := &fakeToolCallingChatModel{
			generateMsg: schema.AssistantMessage("", nil),
			streamMsg: schema.StreamReaderFromArray([]*schema.Message{
				schema.AssistantMessage("hel", nil),
				schema.AssistantMessage("lo", nil),
			}),
		}

		got, err := WrapToolCallingChatModelWithStreamFallback(base).Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if got == nil || got.Content != "hello" {
			t.Fatalf("Generate() content = %#v, want hello", got)
		}
		if base.streamCalls != 1 {
			t.Fatalf("streamCalls = %d, want 1", base.streamCalls)
		}
	})

	t.Run("does not fallback when response only contains tool calls", func(t *testing.T) {
		base := &fakeToolCallingChatModel{
			generateMsg: &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID:   "call_1",
					Type: "function",
				}},
			},
			streamMsg: schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("ignored", nil)}),
		}

		got, err := WrapToolCallingChatModelWithStreamFallback(base).Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if got == nil || len(got.ToolCalls) != 1 {
			t.Fatalf("Generate() tool calls = %#v, want 1 tool call", got)
		}
		if base.streamCalls != 0 {
			t.Fatalf("streamCalls = %d, want 0", base.streamCalls)
		}
	})

	t.Run("returns wrapped stream fallback errors", func(t *testing.T) {
		base := &fakeToolCallingChatModel{
			generateMsg: schema.AssistantMessage("", nil),
			streamErr:   errors.New("proxy stream failed"),
		}

		_, err := WrapToolCallingChatModelWithStreamFallback(base).Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err == nil || err.Error() != "non-stream response was empty; stream fallback failed: proxy stream failed" {
			t.Fatalf("Generate() error = %v", err)
		}
	})

	t.Run("with tools keeps wrapper behavior", func(t *testing.T) {
		child := &fakeToolCallingChatModel{
			generateMsg: schema.AssistantMessage("", nil),
			streamMsg:   schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("from stream", nil)}),
		}
		base := &fakeToolCallingChatModel{
			generateMsg: schema.AssistantMessage("unused", nil),
			withTools:   child,
		}

		wrapped, err := WrapToolCallingChatModelWithStreamFallback(base).WithTools(nil)
		if err != nil {
			t.Fatalf("WithTools() error = %v", err)
		}
		got, err := wrapped.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err != nil {
			t.Fatalf("wrapped.Generate() error = %v", err)
		}
		if got == nil || got.Content != "from stream" {
			t.Fatalf("wrapped.Generate() content = %#v, want from stream", got)
		}
	})

	t.Run("uses cached stream preference directly", func(t *testing.T) {
		const key = "test-stream-mode"
		storeStreamMode(key, streamModePreferStream)
		t.Cleanup(func() { llmStreamModeCache.Delete(key) })
		base := &fakeToolCallingChatModel{
			generateMsg: schema.AssistantMessage("should not be used", nil),
			streamMsg:   schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("stream only", nil)}),
		}

		got, err := wrapToolCallingChatModelWithStreamFallback(base, key).Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if got == nil || got.Content != "stream only" {
			t.Fatalf("Generate() content = %#v, want stream only", got)
		}
		if base.generateCalls != 0 {
			t.Fatalf("generateCalls = %d, want 0", base.generateCalls)
		}
		if base.streamCalls != 1 {
			t.Fatalf("streamCalls = %d, want 1", base.streamCalls)
		}
	})
}

func TestCollectStreamMessage(t *testing.T) {
	t.Run("returns nil when stream is empty", func(t *testing.T) {
		stream := schema.StreamReaderFromArray([]*schema.Message{})
		defer stream.Close()

		got, err := collectStreamMessage(stream)
		if err != nil {
			t.Fatalf("collectStreamMessage() error = %v", err)
		}
		if got != nil {
			t.Fatalf("collectStreamMessage() = %#v, want nil", got)
		}
	})

	t.Run("returns stream read errors", func(t *testing.T) {
		stream, writer := schema.Pipe[*schema.Message](1)
		go func() {
			defer writer.Close()
			_ = writer.Send(nil, errors.New("recv failed"))
		}()
		defer stream.Close()

		_, err := collectStreamMessage(stream)
		if !errors.Is(err, io.EOF) && (err == nil || err.Error() != "recv failed") {
			t.Fatalf("collectStreamMessage() error = %v, want recv failed", err)
		}
	})
}

func TestShouldWarmResolvedConfig(t *testing.T) {
	if ShouldWarmResolvedConfig("", "", "demo-model") {
		t.Fatal("model-only config should not warm during startup")
	}
	if !ShouldWarmResolvedConfig("http://127.0.0.1:11434/v1", "", "demo-model") {
		t.Fatal("baseURL + model should warm during startup")
	}
	if !ShouldWarmResolvedConfig("", "secret", "demo-model") {
		t.Fatal("api key + model should warm during startup")
	}
}

type fakeToolCallingChatModel struct {
	generateMsg   *schema.Message
	generateErr   error
	streamMsg     *schema.StreamReader[*schema.Message]
	streamErr     error
	withTools     model.ToolCallingChatModel
	generateCalls int
	streamCalls   int
}

func (f *fakeToolCallingChatModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	f.generateCalls++
	return f.generateMsg, f.generateErr
}

func (f *fakeToolCallingChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	f.streamCalls++
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	if f.streamMsg != nil {
		return f.streamMsg, nil
	}
	return schema.StreamReaderFromArray([]*schema.Message{}), nil
}

func (f *fakeToolCallingChatModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if f.withTools != nil {
		return f.withTools, nil
	}
	return f, nil
}
