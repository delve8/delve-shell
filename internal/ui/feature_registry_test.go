package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

// Test-only fallback registrations so internal/ui unit tests can run without importing
// remote/session (avoids heavy deps). internal/run cannot be imported here: run → ui
// would form a cycle with ui tests; mirror SlashRunUsageOption handling below instead.
func init() {
	// Overlay close feature resets: see overlay_close_feature_reset.go (package ui init).

	// Mirror remote package title-bar segment (internal/ui tests do not import remote).
	RegisterTitleBarFragmentProvider(func(m Model) (string, bool) {
		if !m.Context.RemoteActive {
			return "", false
		}
		if m.Context.RemoteLabel != "" {
			return "Remote " + m.Context.RemoteLabel, true
		}
		return "Remote", true
	})

	registerSlashExact("/config add-remote", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Overlay.Active = true
			m.Overlay.Title = i18n.T(m.getLang(), i18n.KeyAddRemoteTitle)
			m.AddRemote.Active = true
			m.AddRemote.Error = ""
			m.AddRemote.OfferOverwrite = false
			m.AddRemote.Save = true
			m.AddRemote.Connect = false
			m.PathCompletion.Candidates = nil
			m.PathCompletion.Index = -1
			m.AddRemote.FieldIndex = 0
			m.AddRemote.HostInput = textinput.New()
			m.AddRemote.HostInput.Placeholder = "host or host:22"
			m.AddRemote.HostInput.Focus()
			m.AddRemote.UserInput = textinput.New()
			m.AddRemote.UserInput.Placeholder = "e.g. root"
			m.AddRemote.UserInput.SetValue("root")
			m.AddRemote.NameInput = textinput.New()
			m.AddRemote.NameInput.Placeholder = "name (optional)"
			m.AddRemote.KeyInput = textinput.New()
			m.AddRemote.KeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
			return m, nil
		},
		ClearInput: true,
	})
	registerSlashExact("/remote on", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Overlay.Active = true
			m.Overlay.Title = i18n.T(m.getLang(), i18n.KeyAddRemoteTitle)
			m.AddRemote.Active = true
			m.AddRemote.Error = ""
			m.AddRemote.OfferOverwrite = false
			m.AddRemote.Save = false
			m.AddRemote.Connect = true
			m.PathCompletion.Candidates = nil
			m.PathCompletion.Index = -1
			m.AddRemote.FieldIndex = 0
			m.AddRemote.HostInput = textinput.New()
			m.AddRemote.HostInput.Placeholder = "host or host:22"
			m.AddRemote.HostInput.Focus()
			m.AddRemote.UserInput = textinput.New()
			m.AddRemote.UserInput.Placeholder = "e.g. root"
			m.AddRemote.UserInput.SetValue("root")
			m.AddRemote.NameInput = textinput.New()
			m.AddRemote.NameInput.Placeholder = "name (optional)"
			m.AddRemote.KeyInput = textinput.New()
			m.AddRemote.KeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
			return m, nil
		},
		ClearInput: true,
	})
	registerSlashExact("/config llm", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Overlay.Active = true
			m.Overlay.Title = i18n.T(m.getLang(), i18n.KeyConfigLLMTitle)
			m.ConfigLLM.Active = true
			m.ConfigLLM.Checking = false
			m.ConfigLLM.Error = ""
			m.ConfigLLM.FieldIndex = 0
			m.ConfigLLM.BaseURLInput = textinput.New()
			m.ConfigLLM.BaseURLInput.Focus()
			m.ConfigLLM.ApiKeyInput = textinput.New()
			m.ConfigLLM.ApiKeyInput.Blur()
			m.ConfigLLM.ModelInput = textinput.New()
			m.ConfigLLM.ModelInput.Blur()
			m.ConfigLLM.MaxMessagesInput = textinput.New()
			m.ConfigLLM.MaxMessagesInput.Blur()
			m.ConfigLLM.MaxCharsInput = textinput.New()
			m.ConfigLLM.MaxCharsInput.Blur()
			return m, nil
		},
		ClearInput: true,
	})
	registerSlashExact("/config del-remote", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Input.SetValue("/config del-remote ")
			m.Input.CursorEnd()
			m.Interaction.SlashSuggestIndex = 0
			return m, nil
		},
		ClearInput: false,
	})
	registerSlashPrefix("/config update-skill", SlashPrefixDispatchEntry{
		Prefix: "/config update-skill",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyDescConfigUpdateSkill))))
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil, true
			}
			skillName := fields[0]
			mm.Overlay.Active = true
			mm.Overlay.Title = "Update skill"
			mm.UpdateSkill.Active = true
			mm.UpdateSkill.Name = skillName
			mm.UpdateSkill.Error = ""
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.Interaction.SlashSuggestIndex = 0
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})

	RegisterMessageProvider(func(m Model, msg tea.Msg) (Model, tea.Cmd, bool) {
		switch t := msg.(type) {
		case SessionSwitchedMsg:
			lang := m.getLang()
			m.Context.CurrentSessionPath = t.Path
			sessionID := ""
			if t.Path != "" {
				sessionID = strings.TrimSuffix(filepath.Base(t.Path), ".jsonl")
			}
			switchedLine := sessionSwitchedStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeySessionSwitchedTo, sessionID)))
			if t.Path != "" {
				events, _ := history.ReadRecent(t.Path, maxSessionHistoryEvents)
				msgs := sessionEventsToMessages(events, lang, m.Width)
				m.Messages = make([]string, 0, len(msgs)+2)
				m.Messages = append(m.Messages, msgs...)
				m.Messages = append(m.Messages, switchedLine)
			} else {
				m.Messages = []string{switchedLine}
			}
			m.Messages = append(m.Messages, "")
			m = m.RefreshViewport()
			return m, nil, true
		default:
			return m, nil, false
		}
	})

	RegisterSlashSelectedProvider(func(m Model, chosen string) (Model, tea.Cmd, bool) {
		if !strings.HasPrefix(chosen, "/skill ") {
			return m, nil, false
		}
		m.Input.SetValue(chosen + " ")
		m.Input.CursorEnd()
		m.Interaction.SlashSuggestIndex = 0
		return m, nil, true
	})

	RegisterSlashSelectedProvider(func(m Model, chosen string) (Model, tea.Cmd, bool) {
		if chosen != SlashRunUsageOption {
			return m, nil, false
		}
		m.Input.SetValue("/run ")
		m.Input.CursorEnd()
		return m, nil, true
	})

	RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		currentSessionPath string,
		_ []string,
		_ []string,
		_ bool,
	) ([]SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)
		if normalizedLower != "sessions" && !strings.HasPrefix(normalizedLower, "sessions ") {
			return nil, false
		}
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "sessions"))
		summaries, err := history.ListSessionsWithSummary(20)
		if err != nil || len(summaries) == 0 {
			return []SlashOption{{Cmd: i18n.T(lang, i18n.KeySessionNone), Desc: "", Path: ""}}, true
		}
		filterLower := strings.ToLower(filter)
		var opts []SlashOption
		for _, s := range summaries {
			if s.Path == currentSessionPath {
				continue
			}
			if filterLower != "" {
				line := s.ID
				if s.Snippet != "" {
					line += " " + s.Snippet
				}
				if !strings.Contains(strings.ToLower(line), filterLower) {
					continue
				}
			}
			opts = append(opts, SlashOption{Cmd: "/sessions " + s.ID, Desc: s.Snippet, Path: s.Path})
		}
		if len(opts) == 0 {
			return []SlashOption{{Cmd: i18n.T(lang, i18n.KeySessionNone), Desc: "", Path: ""}}, true
		}
		return opts, true
	})
}
