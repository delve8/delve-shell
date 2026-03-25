package session

import (
	"encoding/json"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/uivm"
)

// EventsToTranscriptLines converts session history events into semantic transcript lines.
// Rendering and wrapping are owned by internal/ui.
func EventsToTranscriptLines(events []history.Event) []uivm.Line {
	out := make([]uivm.Line, 0, len(events)*2)
	for _, ev := range events {
		switch ev.Type {
		case "user_input":
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Text) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineUser, Text: p.Text})
				out = append(out, uivm.Line{Kind: uivm.LineBlank})
			}
		case "llm_response":
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Reply) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineAI, Text: p.Reply})
				out = append(out, uivm.Line{Kind: uivm.LineSeparator})
			}
		case "command":
			var p struct {
				Command   string `json:"command"`
				Suggested bool   `json:"suggested"`
				Kind      string `json:"kind"`
				SkillName string `json:"skill_name"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil || strings.TrimSpace(p.Command) == "" {
				continue
			}
			if p.Kind == "skill" && strings.TrimSpace(p.SkillName) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineSystemSuggest, Text: "Skill: " + strings.TrimSpace(p.SkillName)})
			}
			tag := "approved"
			if p.Suggested {
				tag = "suggested"
			}
			out = append(out, uivm.Line{Kind: uivm.LineExec, Text: "Run: " + p.Command + " (" + tag + ")"})
		case "command_result":
			var p struct {
				Stdout string `json:"stdout"`
				Stderr string `json:"stderr"`
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
				out = append(out, uivm.Line{Kind: uivm.LineResult, Text: result})
			}
			out = append(out, uivm.Line{Kind: uivm.LineBlank})
		}
	}
	return out
}

