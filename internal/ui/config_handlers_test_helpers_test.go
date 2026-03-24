package ui

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
)

func applyTestConfigAllowlistAutoRun(m Model, value string) Model {
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
		return m.RefreshViewport()
	}

	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		return m.RefreshViewport()
	}
	cfg.AllowlistAutoRun = &on
	if on {
		cfg.Mode = "run"
	} else {
		cfg.Mode = "suggest"
	}
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		return m.RefreshViewport()
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

func applyTestConfigAllowlistUpdate(m Model) Model {
	lang := m.getLang()
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		return m.RefreshViewport()
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

func applyTestOverlayCloseFeatureResets(m Model) Model {
	m.AddRemote.Active = false
	m.AddRemote.Connecting = false
	m.AddRemote.Error = ""
	m.AddRemote.OfferOverwrite = false
	m.RemoteAuth.Connecting = false
	m.RemoteAuth.Step = ""
	m.RemoteAuth.Target = ""
	m.RemoteAuth.Error = ""
	m.RemoteAuth.Username = ""

	m.AddSkill.Active = false
	m.AddSkill.Error = ""
	m.UpdateSkill.Active = false
	m.UpdateSkill.Error = ""

	m.ConfigLLM.Active = false
	m.ConfigLLM.Checking = false
	m.ConfigLLM.Error = ""

	return m
}
