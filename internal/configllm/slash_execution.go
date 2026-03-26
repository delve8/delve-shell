package configllm

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/i18n"
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
			return saveConfigLLMField("base_url", strings.TrimSpace(strings.TrimPrefix(text, "/config llm base_url ")), req.CommandSender), true, nil
		case strings.HasPrefix(text, "/config llm api_key "):
			return saveConfigLLMField("api_key", strings.TrimSpace(strings.TrimPrefix(text, "/config llm api_key ")), req.CommandSender), true, nil
		case strings.HasPrefix(text, "/config llm model "):
			return saveConfigLLMField("model", strings.TrimSpace(strings.TrimPrefix(text, "/config llm model ")), req.CommandSender), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}

func saveConfigLLMField(field, value string, sender ui.CommandSender) inputlifecycletype.ProcessResult {
	value = strings.TrimSpace(value)
	lang := "en"
	cfg, err := config.Load()
	if err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{
				Lines: []inputlifecycletype.TranscriptLine{{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T(lang, i18n.KeyConfigPrefix) + err.Error()}},
			},
		})
	}
	switch field {
	case "base_url":
		cfg.LLM.BaseURL = value
	case "api_key":
		cfg.LLM.APIKey = value
	case "model":
		cfg.LLM.Model = value
	default:
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{
				Lines: []inputlifecycletype.TranscriptLine{{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T(lang, i18n.KeyConfigPrefix) + i18n.T(lang, i18n.KeyConfigUnknownField) + field}},
			},
		})
	}
	if err := config.Write(cfg); err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{
				Lines: []inputlifecycletype.TranscriptLine{{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T(lang, i18n.KeyConfigPrefix) + err.Error()}},
			},
		})
	}
	if sender != nil {
		_ = sender.Send(hostcmd.ConfigUpdated{})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{
			Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: i18n.Tf(lang, i18n.KeyConfigSaved, field)},
				{Kind: inputlifecycletype.TranscriptLineBlank},
			},
		},
	})
}
