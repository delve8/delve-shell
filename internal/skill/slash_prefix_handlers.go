package skill

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/skill/store"
)

func handleSlashConfigDelSkillPrefix(rest string) inputlifecycletype.ProcessResult {
	name := strings.TrimSpace(rest)
	if name == "" {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T(i18n.KeyUsageSkillRemove)},
			}},
		})
	}

	if err := skillstore.Remove(name); err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.Tf(i18n.KeySkillRemoveFailed, err)},
			}},
		})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: i18n.Tf(i18n.KeySkillRemoved, name)},
			{Kind: inputlifecycletype.TranscriptLineBlank},
		}},
	})
}

func skillInvocationPrompt(skillName, skillContent, naturalLanguage string) string {
	const header = `[Skill invocation] Fulfill the user's request using ONLY the skill below. Use the run_skill tool with this skill's scripts and parameters; do not suggest arbitrary shell commands unless the skill documentation explicitly allows it.`
	return header + "\n\n## Skill: " + skillName + "\n\n" + skillContent + "\n\n## User request\n\n" + naturalLanguage
}
