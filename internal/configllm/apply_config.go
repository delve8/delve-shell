package configllm

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
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

func applyConfigLLMField(m ui.Model, field, value string) ui.Model {
	value = strings.TrimSpace(value)
	lang := "en"
	cfg, err := config.Load()
	if err != nil {
		m = m.AppendTranscriptLines(ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix) + err.Error()))
		m = m.RefreshViewport()
		return m
	}
	switch field {
	case "base_url":
		cfg.LLM.BaseURL = value
	case "api_key":
		cfg.LLM.APIKey = value
	case "model":
		cfg.LLM.Model = value
	default:
		m = m.AppendTranscriptLines(ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix) + i18n.T(lang, i18n.KeyConfigUnknownField) + field))
		m = m.RefreshViewport()
		return m
	}
	if err := config.Write(cfg); err != nil {
		m = m.AppendTranscriptLines(ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix) + err.Error()))
		m = m.RefreshViewport()
		return m
	}
	prefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
	m = m.AppendTranscriptLines(
		ui.SuggestStyleRender(prefix+i18n.Tf(lang, i18n.KeyConfigSaved, field)),
		"",
	)
	m = m.RefreshViewport()
	m.Host.NotifyConfigUpdated()
	return m
}
