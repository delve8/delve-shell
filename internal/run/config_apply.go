package run

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func applyConfigAllowlistAutoRun(value string, sender ui.CommandSender) inputlifecycletype.ProcessResult {
	value = strings.TrimSpace(strings.ToLower(value))
	var on bool
	switch value {
	case "list-only":
		on = true
	case "disable":
		on = false
	default:
		return transcriptErrorResult(i18n.T("en", i18n.KeyConfigPrefix) + i18n.T("en", i18n.KeyConfigAutoRunRequired))
	}

	cfg, err := config.Load()
	if err != nil {
		return transcriptErrorResult(i18n.T("en", i18n.KeyConfigPrefix) + err.Error())
	}
	cfg.AllowlistAutoRun = &on
	if err := config.Write(cfg); err != nil {
		return transcriptErrorResult(i18n.T("en", i18n.KeyConfigPrefix) + err.Error())
	}
	display := i18n.T("en", i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T("en", i18n.KeyAutoRunNone)
	}
	if sender != nil {
		_ = sender.Send(hostcmd.AllowlistAutoRun{Enabled: on})
	}
	return transcriptSuggestResult(
		i18n.Tf("en", i18n.KeyConfigSavedAllowlistAutoRun, display),
		true,
	)
}

func applyConfigAllowlistUpdate(sender ui.CommandSender) inputlifecycletype.ProcessResult {
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		return transcriptErrorResult(i18n.T("en", i18n.KeyConfigPrefix) + err.Error())
	}
	if sender != nil {
		_ = sender.Send(hostcmd.ConfigUpdated{})
	}
	return transcriptSuggestResult(
		i18n.Tf("en", i18n.KeyAllowlistUpdateDone, added),
		true,
	)
}
