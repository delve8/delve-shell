// Package historytui bridges persisted session history to the Bubble Tea shell: /history slash
// suggestions, transcript line shaping for previews and switches, and the active session path
// for marking the current session in the picker. Core jsonl read/write remains in package history.
package historytui

import (
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const maxSessionsInSlash = 20

// Register wires the /history slash option provider and related UI hooks. Call from [bootstrap.Install].
func Register() {
	ui.RegisterSlashOptionsProvider(historySlashOptionsProvider)
}

func historySlashOptionsProvider(inputVal string, lang string) ([]ui.SlashOption, bool) {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.TrimSpace(normalized)
	normalizedLower := strings.ToLower(normalized)

	if normalizedLower == "history" || strings.HasPrefix(normalizedLower, "history ") {
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "history"))
		return getSessionSlashOptions(filter), true
	}

	_ = lang
	return nil, false
}

func getSessionSlashOptions(filter string) []ui.SlashOption {
	summaries, err := history.ListSessionsWithSummary(maxSessionsInSlash)
	if err != nil || len(summaries) == 0 {
		return []ui.SlashOption{{Cmd: i18n.T(i18n.KeySessionNone), Desc: ""}}
	}

	filterLower := strings.ToLower(filter)
	currentSessionPath := getCurrentSessionPath()
	var opts []ui.SlashOption
	for _, s := range summaries {
		if filterLower != "" {
			line := s.ID
			if s.Path == currentSessionPath {
				line += i18n.T(i18n.KeyHistorySessionCurrentSuffix)
			}
			if s.Snippet != "" {
				line += " " + s.Snippet
			}
			if !strings.Contains(strings.ToLower(line), filterLower) {
				continue
			}
		}

		cmd := "/history " + s.ID
		if s.Path == currentSessionPath {
			cmd += i18n.T(i18n.KeyHistorySessionCurrentSuffix)
		}
		desc := s.Snippet
		opts = append(opts, ui.SlashOption{Cmd: cmd, Desc: desc})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: i18n.T(i18n.KeySessionNone), Desc: ""}}
	}
	return opts
}
