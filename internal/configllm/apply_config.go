package configllm

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/service/configsvc"
	"delve-shell/internal/ui"
)

func applyConfigLLMFromOverlayStart(m ui.Model, baseURL, apiKey, model, maxMessagesStr, maxCharsStr string) ui.Model {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	lang := "en"
	if model == "" {
		return m
	}
	if err := configsvc.SaveLLMFromOverlay(configsvc.SaveLLMParams{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		Model:       model,
		MaxMessages: maxMessagesStr,
		MaxChars:    maxCharsStr,
	}); err != nil {
		m = m.AppendTranscriptLines(ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix) + err.Error()))
		m = m.RefreshViewport()
		return m
	}
	st := getOverlayState()
	st.Error = ""
	st.Checking = true
	setOverlayState(st)
	return m
}

func applyConfigLLMFieldResult(field, value string, sender ui.CommandSender) inputlifecycletype.ProcessResult {
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
