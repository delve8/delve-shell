package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

func registerTestSessionMessageMirror() {
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
				msgs := sessionEventsToMessages(events, lang, m.Layout.Width)
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
}

func registerTestSessionSlashOptionsMirror() {
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
