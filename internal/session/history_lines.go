package session

import (
	"encoding/json"
	"strings"
	"unicode"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	sessionExecStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	sessionResultStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	sessionSuggestStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	sessionSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// sessionEventsToMessages converts history events to display lines used by session replay.
func sessionEventsToMessages(events []history.Event, lang string, width int) []string {
	var out []string
	for _, ev := range events {
		switch ev.Type {
		case "user_input":
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Text != "" {
				line := i18n.T(lang, i18n.KeyUserLabel) + p.Text
				if width > 0 {
					line = wrapString(line, width)
				}
				out = append(out, line, "")
			}
		case "llm_response":
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Reply != "" {
				line := i18n.T(lang, i18n.KeyAILabel) + p.Reply
				if width > 0 {
					line = wrapString(line, width)
				}
				out = append(out, line)
				sepW := width
				if sepW <= 0 {
					sepW = 40
				}
				out = append(out, sessionSeparatorStyle.Render(strings.Repeat("─", sepW)))
			}
		case "command":
			var p struct {
				Command   string `json:"command"`
				Approved  bool   `json:"approved"`
				Suggested bool   `json:"suggested"`
				Kind      string `json:"kind"`
				SkillName string `json:"skill_name"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil || p.Command == "" {
				continue
			}
			if p.Kind == "skill" && strings.TrimSpace(p.SkillName) != "" {
				skillLine := i18n.Tf(lang, i18n.KeySkillLine, strings.TrimSpace(p.SkillName))
				if width > 0 {
					skillLine = wrapString(skillLine, width)
				}
				out = append(out, sessionSuggestStyle.Render(skillLine))
			}
			tag := i18n.T(lang, i18n.KeyRunTagApproved)
			if p.Suggested {
				tag = i18n.T(lang, i18n.KeyRunTagSuggested)
			}
			line := i18n.T(lang, i18n.KeyRunLabel) + p.Command + " (" + tag + ")"
			if width > 0 {
				line = wrapString(line, width)
			}
			out = append(out, sessionExecStyle.Render(line))
		case "command_result":
			var p struct {
				Command  string `json:"command"`
				Stdout   string `json:"stdout"`
				Stderr   string `json:"stderr"`
				ExitCode int    `json:"exit_code"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil {
				continue
			}
			result := strings.TrimSpace(p.Stdout)
			if p.Stderr != "" {
				if result != "" {
					result += "\n"
				}
				result += strings.TrimSpace(p.Stderr)
			}
			if result != "" {
				if width > 0 {
					result = wrapString(result, width)
				}
				out = append(out, sessionResultStyle.Render(result))
			}
			out = append(out, "")
		}
	}
	return out
}

// wrapString breaks s into lines of at most maxWidth terminal cells.
func wrapString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	start := 0
	cellWidth := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			b.WriteString(string(runes[start : i+1]))
			start = i + 1
			cellWidth = 0
			continue
		}
		w := runewidth.RuneWidth(r)
		if cellWidth+w > maxWidth && cellWidth > 0 {
			breakAt := i
			for j := i - 1; j >= start; j-- {
				if unicode.IsSpace(runes[j]) {
					breakAt = j + 1
					break
				}
			}
			b.WriteString(string(runes[start:breakAt]))
			b.WriteByte('\n')
			start = breakAt
			for start < len(runes) && unicode.IsSpace(runes[start]) {
				start++
			}
			cellWidth = 0
			i = start - 1
			continue
		}
		cellWidth += w
	}
	if start < len(runes) {
		b.WriteString(string(runes[start:]))
	}
	return b.String()
}
