package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

// Test-only fallback registrations so internal/ui unit tests can run
// without importing feature packages (which would create import cycles).
func init() {
	registerSlashExact("/config add-remote", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openAddRemoteOverlay(true, false), nil
		},
		ClearInput: true,
	})
	registerSlashExact("/remote on", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openAddRemoteOverlay(false, true), nil
		},
		ClearInput: true,
	})
	registerSlashExact("/config llm", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.OverlayActive = true
			m.OverlayTitle = i18n.T(m.getLang(), i18n.KeyConfigLLMTitle)
			m.ConfigLLMActive = true
			m.ConfigLLMChecking = false
			m.ConfigLLMError = ""
			m.ConfigLLMFieldIndex = 0
			m.ConfigLLMBaseURLInput = textinput.New()
			m.ConfigLLMBaseURLInput.Focus()
			m.ConfigLLMApiKeyInput = textinput.New()
			m.ConfigLLMApiKeyInput.Blur()
			m.ConfigLLMModelInput = textinput.New()
			m.ConfigLLMModelInput.Blur()
			m.ConfigLLMMaxMessagesInput = textinput.New()
			m.ConfigLLMMaxMessagesInput.Blur()
			m.ConfigLLMMaxCharsInput = textinput.New()
			m.ConfigLLMMaxCharsInput.Blur()
			return m, nil
		},
		ClearInput: true,
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
			mm.OverlayActive = true
			mm.OverlayTitle = "Update skill"
			mm.UpdateSkillActive = true
			mm.UpdateSkillName = skillName
			mm.UpdateSkillError = ""
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.SlashSuggestIndex = 0
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})

	RegisterMessageProvider(func(m Model, msg tea.Msg) (Model, tea.Cmd, bool) {
		switch t := msg.(type) {
		case SessionSwitchedMsg:
			lang := m.getLang()
			m.CurrentSessionPath = t.Path
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
