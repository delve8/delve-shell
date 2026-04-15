package historytui

import (
	"encoding/json"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
)

func historyRunLinePrefix(suggested, autoAllowed bool, approved *bool, execution string, offlineMode bool, executionTarget string) string {
	var prefix string
	if offlineMode || strings.EqualFold(strings.TrimSpace(execution), "offline_manual") {
		prefix = i18n.T(i18n.KeyRunLineOfflineManual)
	} else if suggested {
		prefix = i18n.T(i18n.KeyRunLineSuggested)
	} else if autoAllowed {
		prefix = i18n.T(i18n.KeyRunLineAutoAllowed)
	} else if approved != nil && !*approved {
		prefix = i18n.T(i18n.KeyRunLineNotApproved)
	} else {
		prefix = i18n.T(i18n.KeyRunLineApproved)
	}
	if label := historyExecutionLabel(execution, offlineMode, executionTarget); label != "" {
		prefix = insertRunPrefixTarget(prefix, label)
	}
	return prefix
}

func historyExecutionLabel(execution string, offlineMode bool, executionTarget string) string {
	execution = strings.TrimSpace(strings.ToLower(execution))
	target := strings.TrimSpace(executionTarget)
	if offlineMode || execution == history.ExecutionOfflineManual {
		if target != "" {
			return target
		}
		return i18n.T(i18n.KeyRemoteTitleBarOffline)
	}
	switch execution {
	case history.ExecutionLocal:
		if target != "" {
			return target
		}
		return i18n.T(i18n.KeyTitleBarLocal)
	case history.ExecutionRemote:
		if target != "" {
			return i18n.T(i18n.KeyRemoteTitleBarRemote) + " " + target
		}
		return i18n.T(i18n.KeyRemoteTitleBarRemote)
	default:
		return target
	}
}

func insertRunPrefixTarget(prefix, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return prefix
	}
	trimmed := strings.TrimRight(prefix, " \t")
	if strings.HasSuffix(trimmed, ":") {
		return strings.TrimSuffix(trimmed, ":") + " @ " + target + ": "
	}
	return prefix + "@ " + target + " "
}

// EventsToTranscriptLinesForHistoryPreview converts session events into transcript lines for the
// /history read-only preview: full command text and full stdout/stderr (no Run-line width cap;
// command output not trimmed away beyond normal JSON payload).
func EventsToTranscriptLinesForHistoryPreview(events []history.Event) []uivm.Line {
	out := make([]uivm.Line, 0, len(events)*2)
	for _, ev := range events {
		switch ev.Type {
		case history.EventTypeUserInput:
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Text) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineUser, Text: p.Text})
				out = append(out, uivm.Line{Kind: uivm.LineBlank})
			}
		case history.EventTypeLLMResponse:
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Reply) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineAI, Text: p.Reply})
			}
		case history.EventTypeCommand:
			var p struct {
				Command     string `json:"command"`
				Suggested   bool   `json:"suggested"`
				Guidance    string `json:"guidance"`
				Kind        string `json:"kind"`
				SkillName   string `json:"skill_name"`
				Execution   string `json:"execution"`
				ExecTarget  string `json:"execution_target"`
				OfflineMode bool   `json:"offline_mode"`
				AutoAllowed bool   `json:"auto_allowed"`
				Approved    *bool  `json:"approved"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil || strings.TrimSpace(p.Command) == "" {
				continue
			}
			if p.Kind == history.CommandPayloadKindSkill && strings.TrimSpace(p.SkillName) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineSystemSuggest, Text: "Skill: " + strings.TrimSpace(p.SkillName)})
			}
			prefix := historyRunLinePrefix(p.Suggested, p.AutoAllowed, p.Approved, p.Execution, p.OfflineMode, p.ExecTarget)
			out = append(out, uivm.Line{Kind: uivm.LineExec, Text: ui.FormatRunTranscriptLineFull(prefix, p.Command)})
			if g := strings.TrimSpace(p.Guidance); g != "" {
				out = append(out, uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.T(i18n.KeyApprovalUserGuidance) + " " + g})
			}
		case history.EventTypeCommandResult:
			var p struct {
				Stdout string `json:"stdout"`
				Stderr string `json:"stderr"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil {
				continue
			}
			var result string
			if p.Stdout != "" {
				result = p.Stdout
			}
			if p.Stderr != "" {
				if result != "" {
					result += "\n"
				}
				result += p.Stderr
			}
			if result != "" {
				out = append(out, uivm.Line{Kind: uivm.LineResult, Text: result})
			}
			out = append(out, uivm.Line{Kind: uivm.LineBlank})
		}
	}
	return out
}

// EventsToTranscriptLines converts session history events into semantic transcript lines.
// Rendering and wrapping are owned by internal/ui.
func EventsToTranscriptLines(events []history.Event) []uivm.Line {
	out := make([]uivm.Line, 0, len(events)*2)
	for _, ev := range events {
		switch ev.Type {
		case history.EventTypeUserInput:
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Text) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineUser, Text: p.Text})
				out = append(out, uivm.Line{Kind: uivm.LineBlank})
			}
		case history.EventTypeLLMResponse:
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Reply) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineAI, Text: p.Reply})
			}
		case history.EventTypeCommand:
			var p struct {
				Command     string `json:"command"`
				Suggested   bool   `json:"suggested"`
				Guidance    string `json:"guidance"`
				Kind        string `json:"kind"`
				SkillName   string `json:"skill_name"`
				Execution   string `json:"execution"`
				ExecTarget  string `json:"execution_target"`
				OfflineMode bool   `json:"offline_mode"`
				AutoAllowed bool   `json:"auto_allowed"`
				Approved    *bool  `json:"approved"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil || strings.TrimSpace(p.Command) == "" {
				continue
			}
			if p.Kind == history.CommandPayloadKindSkill && strings.TrimSpace(p.SkillName) != "" {
				out = append(out, uivm.Line{Kind: uivm.LineSystemSuggest, Text: "Skill: " + strings.TrimSpace(p.SkillName)})
			}
			prefix := historyRunLinePrefix(p.Suggested, p.AutoAllowed, p.Approved, p.Execution, p.OfflineMode, p.ExecTarget)
			out = append(out, uivm.Line{Kind: uivm.LineExec, Text: ui.FormatRunTranscriptLine(prefix, p.Command)})
			if g := strings.TrimSpace(p.Guidance); g != "" {
				out = append(out, uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.T(i18n.KeyApprovalUserGuidance) + " " + g})
			}
		case history.EventTypeCommandResult:
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
