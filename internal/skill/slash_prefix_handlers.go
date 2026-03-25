package skill

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/ui"
)

func handleSlashConfigDelSkillPrefix(m ui.Model, rest string) ui.Model {
	lang := "en"
	name := strings.TrimSpace(rest)
	if name == "" {
		m = m.AppendTranscriptLines(errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeyUsageSkillRemove))))
		return m.RefreshViewport()
	}

	if err := skillsvc.Remove(name); err != nil {
		m = m.AppendTranscriptLines(errStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillRemoveFailed, err))))
	} else {
		m = m.AppendTranscriptLines(suggestStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillRemoved, name))))
	}
	m = m.ClearSlashInput()
	return m.RefreshViewport()
}

func skillInvocationPrompt(skillName, skillContent, naturalLanguage string) string {
	const header = `[Skill invocation] Fulfill the user's request using ONLY the skill below. Use the run_skill tool with this skill's scripts and parameters; do not suggest arbitrary shell commands unless the skill documentation explicitly allows it.`
	return header + "\n\n## Skill: " + skillName + "\n\n" + skillContent + "\n\n## User request\n\n" + naturalLanguage
}
