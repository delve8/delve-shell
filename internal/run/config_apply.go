package run

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func applyConfigAllowlistAutoRun(m ui.Model, value string) ui.Model {
	value = strings.TrimSpace(strings.ToLower(value))
	var on bool
	switch value {
	case "list-only":
		on = true
	case "disable":
		on = false
	default:
		m = m.AppendTranscriptLines(errStyle.Render(i18n.T("en", i18n.KeyConfigPrefix) + i18n.T("en", i18n.KeyConfigAutoRunRequired)))
		return m.RefreshViewport()
	}

	cfg, err := config.Load()
	if err != nil {
		m = m.AppendTranscriptLines(errStyle.Render(i18n.T("en", i18n.KeyConfigPrefix) + err.Error()))
		return m.RefreshViewport()
	}
	cfg.AllowlistAutoRun = &on
	if on {
		cfg.Mode = "run"
	} else {
		cfg.Mode = "suggest"
	}
	if err := config.Write(cfg); err != nil {
		m = m.AppendTranscriptLines(errStyle.Render(i18n.T("en", i18n.KeyConfigPrefix) + err.Error()))
		return m.RefreshViewport()
	}
	display := i18n.T("en", i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T("en", i18n.KeyAutoRunNone)
	}
	m = m.AppendTranscriptLines(
		delveMsg("en", i18n.Tf("en", i18n.KeyConfigSavedAllowlistAutoRun, display)),
		"",
	)
	m = m.RefreshViewport()
	m.EmitAllowlistAutoRunSyncIntent(on)
	return m
}

func applyConfigAllowlistUpdate(m ui.Model) ui.Model {
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		m = m.AppendTranscriptLines(errStyle.Render(i18n.T("en", i18n.KeyConfigPrefix) + err.Error()))
		return m.RefreshViewport()
	}
	m = m.AppendTranscriptLines(
		delveMsg("en", i18n.Tf("en", i18n.KeyAllowlistUpdateDone, added)),
		"",
	)
	m = m.RefreshViewport()
	m.EmitConfigUpdatedIntent()
	return m
}
