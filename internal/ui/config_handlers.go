package ui

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
)

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
		m = m.RefreshViewport()
		return m
	}
	if m.Ports.AllowlistAutoRunChangeChan != nil {
		select {
		case m.Ports.AllowlistAutoRunChangeChan <- on:
		default:
		}
	}
	display := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyAllowlistAutoRunSetTo, display))))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
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
		m = m.RefreshViewport()
		return m
	}
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m = m.RefreshViewport()
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
		m = m.RefreshViewport()
		return m
	}
	display := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigSavedAllowlistAutoRun, display))))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	if m.Ports.ConfigUpdatedChan != nil {
		select {
		case m.Ports.ConfigUpdatedChan <- struct{}{}:
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
		m = m.RefreshViewport()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyAllowlistUpdateDone, added))))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	if m.Ports.ConfigUpdatedChan != nil {
		select {
		case m.Ports.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}
