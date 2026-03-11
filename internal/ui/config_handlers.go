package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
)

// openConfigLLMOverlay opens the Config LLM overlay with current config values pre-filled.
func (m Model) openConfigLLMOverlay() Model {
	cfg, err := config.Load()
	if err != nil {
		lang := m.getLang()
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.OverlayActive = true
	m.OverlayTitle = i18n.T(m.getLang(), i18n.KeyConfigLLMTitle)
	m.ConfigLLMActive = true
	m.ConfigLLMError = ""
	m.ConfigLLMFieldIndex = 0
	m.ConfigLLMBaseURLInput = textinput.New()
	m.ConfigLLMBaseURLInput.Placeholder = "https://api.openai.com/v1 (optional)"
	m.ConfigLLMBaseURLInput.SetValue(cfg.LLM.BaseURL)
	m.ConfigLLMBaseURLInput.Focus()
	m.ConfigLLMApiKeyInput = textinput.New()
	m.ConfigLLMApiKeyInput.Placeholder = "sk-... or $API_KEY"
	m.ConfigLLMApiKeyInput.EchoMode = textinput.EchoPassword
	m.ConfigLLMApiKeyInput.SetValue(cfg.LLM.APIKey)
	m.ConfigLLMApiKeyInput.Blur()
	m.ConfigLLMModelInput = textinput.New()
	m.ConfigLLMModelInput.Placeholder = "gpt-4o-mini (optional)"
	m.ConfigLLMModelInput.SetValue(cfg.LLM.Model)
	m.ConfigLLMModelInput.Blur()
	return m
}

// applyConfigLLMFromOverlay writes all three llm fields (base_url, api_key, model) to config at once. api_key is required.
func (m Model) applyConfigLLMFromOverlay(baseURL, apiKey, model string) Model {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	lang := m.getLang()
	if apiKey == "" {
		return m // caller sets ConfigLLMError
	}
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	cfg.LLM.BaseURL = baseURL
	cfg.LLM.APIKey = apiKey
	cfg.LLM.Model = model
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyConfigSavedLLM)))
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

// applyConfigAddRemote adds a remote via /config add-remote <user@host> [name] [identity_file]. Name is optional.
func (m Model) applyConfigAddRemote(args string) Model {
	lang := m.getLang()
	parts := strings.Fields(args)
	if len(parts) < 1 {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+"Usage: /config add-remote <user@host> [name] [identity_file]"))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	target := parts[0]
	name := ""
	identityFile := ""
	if len(parts) >= 2 {
		name = parts[1]
	}
	if len(parts) >= 3 {
		identityFile = parts[2]
	}
	if err := config.AddRemote(target, name, identityFile); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	display := target
	if name != "" {
		display = name + " (" + target + ")"
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)))
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

// applyConfigRemoveRemote removes a remote via /config remove-remote <name-or-target> (name or target from list).
func (m Model) applyConfigRemoveRemote(nameOrTarget string) Model {
	lang := m.getLang()
	nameOrTarget = strings.TrimSpace(nameOrTarget)
	if nameOrTarget == "" {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+"Usage: select a remote from /config remove-remote list"))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if err := config.RemoveRemoteByName(nameOrTarget); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigRemoteRemoved, nameOrTarget)))
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
