package session

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const maxSessionHistoryEvents = 500

// Register wires session slash commands and the session-switched message provider. Call from [bootstrap.Install].
func Register() {
	ui.RegisterSlashExact("/new", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			_ = m.Submit("/new")
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
			m.TrySubmitNonBlocking("/sessions " + id)
			return m.RefreshViewport(), nil, true
		},
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
