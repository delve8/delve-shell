package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

// registerTestSlashExactMirrors mirrors exact handlers registered by non-ui packages
// so internal/ui tests can run without importing those packages.
func registerTestSlashExactMirrors() {
	RegisterSlashExact("/help", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openHelpOverlay(), nil
		},
		ClearInput: true,
	})

	appendConfigHint := func(m Model) (Model, tea.Cmd) {
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
		m = m.RefreshViewport()
		return m, nil
	}
	RegisterSlashExact("/config show", SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: false,
	})
	RegisterSlashExact("/config", SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: false,
	})
	RegisterSlashExact("/config update auto-run list", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return applyTestConfigAllowlistUpdate(m), nil
		},
		ClearInput: true,
	})
	reloadConfig := func(m Model) (Model, tea.Cmd) {
		if m.Ports.ConfigUpdatedChan != nil {
			select {
			case m.Ports.ConfigUpdatedChan <- struct{}{}:
			default:
			}
		}
		return m, nil
	}
	RegisterSlashExact("/config reload", SlashExactDispatchEntry{
		Handle:     reloadConfig,
		ClearInput: true,
	})
	RegisterSlashExact("/reload", SlashExactDispatchEntry{
		Handle:     reloadConfig,
		ClearInput: true,
	})
	RegisterSlashExact("/cancel", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.Interaction.WaitingForAI && m.Ports.CancelRequestChan != nil {
				select {
				case m.Ports.CancelRequestChan <- struct{}{}:
				default:
				}
				m.Interaction.WaitingForAI = false
				return m, nil
			}
			lang := m.getLang()
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyNoRequestInProgress))))
			m = m.RefreshViewport()
			return m, nil
		},
		ClearInput: false,
	})
	RegisterSlashExact("/q", SlashExactDispatchEntry{
		Handle:     func(m Model) (Model, tea.Cmd) { return m, tea.Quit },
		ClearInput: false,
	})
	RegisterSlashExact("/sh", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.Ports.ShellRequestedChan != nil {
				msgs := make([]string, len(m.Messages))
				copy(msgs, m.Messages)
				select {
				case m.Ports.ShellRequestedChan <- msgs:
				default:
				}
			}
			return m, tea.Quit
		},
		ClearInput: false,
	})
}

func registerTestStaticSlashOptionsMirror() {
	RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		_ string,
		localRunCommands []string,
		remoteRunCommands []string,
		remoteActive bool,
	) ([]SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)
		if normalizedLower != "run" && !strings.HasPrefix(normalizedLower, "run ") {
			return nil, false
		}
		if normalizedLower == "run" {
			return []SlashOption{{Cmd: SlashRunUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun), Path: ""}}, true
		}
		rest := ""
		if len(normalized) >= 3 {
			rest = strings.TrimSpace(normalized[3:])
		}
		if strings.Contains(rest, " ") || strings.Contains(rest, "\t") {
			return []SlashOption{}, true
		}
		prefix := strings.ToLower(rest)
		cands := localRunCommands
		if cands == nil {
			cands = LocalRunCommands()
		}
		if remoteActive && len(remoteRunCommands) > 0 {
			cands = remoteRunCommands
		}
		const maxRunCands = 50
		opts := make([]SlashOption, 0, 12)
		for _, c := range cands {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(c), prefix) {
				continue
			}
			opts = append(opts, SlashOption{Cmd: "/run " + c, Desc: "", Path: ""})
			if len(opts) >= maxRunCands {
				break
			}
		}
		return opts, true
	})

	RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		_ string,
		_ []string,
		_ []string,
		_ bool,
	) ([]SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)
		if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
			return testConfigSlashOptions(lang), true
		}
		return nil, false
	})

	RegisterRootSlashOptionProvider(func(lang string) []SlashOption {
		return testRootSlashOptions(lang)
	})
}

func testRootSlashOptions(lang string) []SlashOption {
	return []SlashOption{
		{Cmd: "/help", Desc: i18n.T(lang, i18n.KeyDescHelp), Path: ""},
		{Cmd: "/cancel", Desc: i18n.T(lang, i18n.KeyDescCancel), Path: ""},
		{Cmd: "/config", Desc: i18n.T(lang, i18n.KeyDescConfig), Path: ""},
		{Cmd: "/remote", Desc: i18n.T(lang, i18n.KeyDescRemoteOn), Path: ""},
		{Cmd: "/new", Desc: i18n.T(lang, i18n.KeySessionNew), Path: ""},
		{Cmd: "/sessions", Desc: i18n.T(lang, i18n.KeyDescSessions), Path: ""},
		{Cmd: "/skill <skill-name> [detail]", Desc: i18n.T(lang, i18n.KeyDescSkill), Path: ""},
		{Cmd: SlashRunUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun), Path: ""},
		{Cmd: "/sh", Desc: i18n.T(lang, i18n.KeyDescSh), Path: ""},
		{Cmd: "/q", Desc: i18n.T(lang, i18n.KeyDescExit), Path: ""},
	}
}

func testConfigSlashOptions(lang string) []SlashOption {
	return []SlashOption{
		{Cmd: "/config add-remote", Desc: i18n.T(lang, i18n.KeyDescConfigAddRemote), Path: ""},
		{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyDescConfigRemoveRemote), Path: ""},
		{Cmd: "/config add-skill", Desc: i18n.T(lang, i18n.KeyDescSkillInstall), Path: ""},
		{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeyDescSkillRemove), Path: ""},
		{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeyDescConfigUpdateSkill), Path: ""},
		{Cmd: "/config auto-run list-only", Desc: i18n.T(lang, i18n.KeyDescAutoRunListOnly), Path: ""},
		{Cmd: "/config auto-run disable", Desc: i18n.T(lang, i18n.KeyDescAutoRunDisable), Path: ""},
		{Cmd: "/config update auto-run list", Desc: i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate), Path: ""},
		{Cmd: "/config llm", Desc: i18n.T(lang, i18n.KeyDescConfigLLM), Path: ""},
		{Cmd: "/config reload", Desc: i18n.T(lang, i18n.KeyDescReload), Path: ""},
	}
}
