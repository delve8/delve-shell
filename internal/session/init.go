package session

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

var sessionSwitchedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Italic(true)

const maxSessionHistoryEvents = 500

func init() {
	ui.RegisterSlashExact("/new", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.Ports.SubmitChan != nil {
				m.Ports.SubmitChan <- "/new"
			}
			// /new consumes input and refreshes content (keep old behavior).
			m = m.ClearSlashInput()
			m = m.RefreshViewport()
			return m, nil
		},
		ClearInput: false,
	})

	ui.RegisterSlashPrefix("/sessions ", ui.SlashPrefixDispatchEntry{
		Prefix: "/sessions ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			id := strings.TrimSpace(rest)
			if id == "" {
				return m, nil, true
			}
			if m.Ports.SessionSwitchChan != nil {
				sessionPath := filepath.Join(config.HistoryDir(), id+".jsonl")
				select {
				case m.Ports.SessionSwitchChan <- sessionPath:
				default:
				}
			}
			return m.RefreshViewport(), nil, true
		},
	})

	// Delegate session switched message to ui handler.
	ui.RegisterMessageProvider(func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
		switch t := msg.(type) {
		case ui.SessionSwitchedMsg:
			lang := "en"
			m.CurrentSessionPath = t.Path
			sessionID := ""
			if t.Path != "" {
				sessionID = strings.TrimSuffix(filepath.Base(t.Path), ".jsonl")
			}
			switchedLine := sessionSwitchedStyle.Render(i18n.T(lang, i18n.KeyDelveLabel) + " " + i18n.Tf(lang, i18n.KeySessionSwitchedTo, sessionID))
			if t.Path != "" {
				events, _ := history.ReadRecent(t.Path, maxSessionHistoryEvents)
				msgs := ui.SessionEventsToMessages(events, lang, m.Width)
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

	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		currentSessionPath string,
		_ []string,
		_ []string,
		_ bool,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)

		if normalizedLower == "sessions" || strings.HasPrefix(normalizedLower, "sessions ") {
			filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "sessions"))
			return getSessionSlashOptions(filter, currentSessionPath), true
		}

		_ = lang
		return nil, false
	})
}

const maxSessionsInSlash = 20

func getSessionSlashOptions(filter string, currentSessionPath string) []ui.SlashOption {
	summaries, err := history.ListSessionsWithSummary(maxSessionsInSlash)
	if err != nil || len(summaries) == 0 {
		return []ui.SlashOption{{Cmd: i18n.T("en", i18n.KeySessionNone), Desc: "", Path: ""}}
	}

	filterLower := strings.ToLower(filter)
	var opts []ui.SlashOption
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

		cmd := "/sessions " + s.ID
		desc := s.Snippet
		opts = append(opts, ui.SlashOption{Cmd: cmd, Desc: desc, Path: s.Path})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: i18n.T("en", i18n.KeySessionNone), Desc: "", Path: ""}} // Path empty = no session to switch to
	}
	return opts
}
