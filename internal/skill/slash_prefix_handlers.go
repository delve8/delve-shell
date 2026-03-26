package skill

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func handleSlashConfigDelSkillPrefix(rest string) inputlifecycletype.ProcessResult {
	lang := "en"
	name := strings.TrimSpace(rest)
	if name == "" {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputMessage,
			Message: &inputlifecycletype.MessagePayload{
				Value: ui.TranscriptAppendMsg{Lines: []uivm.Line{
					{Kind: uivm.LineSystemError, Text: i18n.T(lang, i18n.KeyUsageSkillRemove)},
				}},
			},
		})
	}

	if err := skillsvc.Remove(name); err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputMessage,
			Message: &inputlifecycletype.MessagePayload{
				Value: ui.TranscriptAppendMsg{Lines: []uivm.Line{
					{Kind: uivm.LineSystemError, Text: i18n.Tf(lang, i18n.KeySkillRemoveFailed, err)},
				}},
			},
		})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{
			Value: ui.TranscriptAppendMsg{Lines: []uivm.Line{
				{Kind: uivm.LineSystemSuggest, Text: i18n.Tf(lang, i18n.KeySkillRemoved, name)},
			}},
		},
	})
}

func skillInvocationPrompt(skillName, skillContent, naturalLanguage string) string {
	const header = `[Skill invocation] Fulfill the user's request using ONLY the skill below. Use the run_skill tool with this skill's scripts and parameters; do not suggest arbitrary shell commands unless the skill documentation explicitly allows it.`
	return header + "\n\n## Skill: " + skillName + "\n\n" + skillContent + "\n\n## User request\n\n" + naturalLanguage
}
