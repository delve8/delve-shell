package ui

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
)

// applyConfigLLM sets one llm field in config.yaml and writes back; value supports $VAR env expansion.
func (m Model) applyConfigLLM(field, value string) Model {
	value = strings.TrimSpace(value)
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
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
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigUnknownField)+field))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigSaved, field)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigLanguage sets config.yaml language and writes back.
func (m Model) applyConfigLanguage(value string) Model {
	value = strings.TrimSpace(value)
	if value == "" {
		lang := m.getLang()
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigLanguageRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	cfg.Language = value
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Input.Placeholder = i18n.T(value, i18n.KeyPlaceholderInput)
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigSavedLanguage, value)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// showConfig displays current config path and LLM summary (api_key masked) in the conversation area.
func (m Model) showConfig() Model {
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(config.ConfigPath()+"\n"+cfg.LLMSummary()))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// applyAllowlistAutoRunSwitch sets runtime allowlist auto-run (on -> true, off -> false) and sends to AllowlistAutoRunChangeChan; does not write config.
func (m Model) applyAllowlistAutoRunSwitch(value string) Model {
	value = strings.TrimSpace(strings.ToLower(value))
	lang := m.getLang()
	var on bool
	switch value {
	case "on":
		on = true
	case "off":
		on = false
	default:
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigAutoRunRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if m.AllowlistAutoRunChangeChan != nil {
		select {
		case m.AllowlistAutoRunChangeChan <- on:
		default:
		}
	}
	display := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyAllowlistAutoRunSetTo, display)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// applyConfigAllowlistAutoRun sets allowlist_auto_run in config and writes; next startup will use it.
// value: "list-only" -> on, "disable" -> off.
func (m Model) applyConfigAllowlistAutoRun(value string) Model {
	value = strings.TrimSpace(strings.ToLower(value))
	var on bool
	switch value {
	case "list-only":
		on = true
	case "disable":
		on = false
	default:
		lang := m.getLang()
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigAutoRunRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	cfg.AllowlistAutoRun = &on
	if on {
		cfg.Mode = "run"
	} else {
		cfg.Mode = "suggest"
	}
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	display := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigSavedAllowlistAutoRun, display)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigAllowlistUpdate merges built-in default allowlist into current allowlist.yaml, appending only missing patterns.
func (m Model) applyConfigAllowlistUpdate() Model {
	lang := m.getLang()
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyAllowlistUpdateDone, added)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}
