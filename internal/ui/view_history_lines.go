package ui

import (
	"encoding/json"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

const maxSessionHistoryEvents = 500

// sessionEventsToMessages converts history events to the same display lines used in the live conversation (User:, AI:, Run:, result).
// width is used to soft-wrap long command lines; if <= 0, no wrapping is applied.
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
				out = append(out, line)
				out = append(out, "") // blank line before command or AI reply
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
				out = append(out, separatorStyle.Render(strings.Repeat("─", sepW)))
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
				out = append(out, suggestStyle.Render(skillLine))
			}
			tag := i18n.T(lang, i18n.KeyRunTagApproved)
			if p.Suggested {
				tag = i18n.T(lang, i18n.KeyRunTagSuggested)
			}
			line := i18n.T(lang, i18n.KeyRunLabel) + p.Command + " (" + tag + ")"
			if width > 0 {
				line = wrapString(line, width)
			}
			out = append(out, execStyle.Render(line))
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
				out = append(out, resultStyle.Render(result))
			}
			out = append(out, "") // blank line after command output
		}
	}
	return out
}
