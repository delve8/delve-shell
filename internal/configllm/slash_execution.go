package configllm

import (
	"strings"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == "/config llm":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputOverlayOpen,
				Overlay: &inputlifecycletype.OverlayPayload{
					Key: "config_llm",
				},
			}), true, nil
		case strings.HasPrefix(text, "/config llm base_url "):
			return applyConfigLLMFieldResult("base_url", strings.TrimSpace(strings.TrimPrefix(text, "/config llm base_url ")), req.CommandSender), true, nil
		case strings.HasPrefix(text, "/config llm api_key "):
			return applyConfigLLMFieldResult("api_key", strings.TrimSpace(strings.TrimPrefix(text, "/config llm api_key ")), req.CommandSender), true, nil
		case strings.HasPrefix(text, "/config llm model "):
			return applyConfigLLMFieldResult("model", strings.TrimSpace(strings.TrimPrefix(text, "/config llm model ")), req.CommandSender), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
