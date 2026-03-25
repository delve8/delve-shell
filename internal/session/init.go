package session

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

var sessionSwitchedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Italic(true)

const maxSessionHistoryEvents = 500

// Register wires session slash commands and the session-switched message provider. Call from [bootstrap.Install].
func Register() {
	ui.RegisterSlashExact("/new", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			_ = m.Host.Submit("/new")
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
			m.Host.TrySubmitNonBlocking("/sessions " + id)
			return m.RefreshViewport(), nil, true
		},
	})

	// Delegate session switched message to ui handler.
	ui.RegisterMessageProvider(func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
		switch t := msg.(type) {
		case ui.SessionSwitchedMsg:
			lang := "en"
			_ = t
			path := getCurrentSessionPath()
			sessionID := ""
			if path != "" {
				sessionID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
			}
			switchedLine := sessionSwitchedStyle.Render(i18n.T(lang, i18n.KeyDelveLabel) + " " + i18n.Tf(lang, i18n.KeySessionSwitchedTo, sessionID))
			if path != "" {
				events, _ := history.ReadRecent(path, maxSessionHistoryEvents)
				msgs := sessionEventsToMessages(events, lang, m.LayoutWidth())
				lines := make([]string, 0, len(msgs)+2)
				lines = append(lines, msgs...)
				lines = append(lines, switchedLine)
				m = m.WithTranscriptLines(lines)
			} else {
				m = m.WithTranscriptLines([]string{switchedLine})
			}
			m = m.AppendTranscriptLines("")
			m = m.RefreshViewport()
			return m, nil, true
		default:
			return m, nil, false
		}
	})

	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)

		if normalizedLower == "sessions" || strings.HasPrefix(normalizedLower, "sessions ") {
			filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "sessions"))
			return getSessionSlashOptions(filter), true
		}

		_ = lang
		return nil, false
	})
}

const maxSessionsInSlash = 20

func getSessionSlashOptions(filter string) []ui.SlashOption {
	summaries, err := history.ListSessionsWithSummary(maxSessionsInSlash)
	if err != nil || len(summaries) == 0 {
		return []ui.SlashOption{{Cmd: i18n.T("en", i18n.KeySessionNone), Desc: ""}}
	}

	filterLower := strings.ToLower(filter)
	currentSessionPath := getCurrentSessionPath()
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
		opts = append(opts, ui.SlashOption{Cmd: cmd, Desc: desc})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: i18n.T("en", i18n.KeySessionNone), Desc: ""}}
	}
	return opts
}
