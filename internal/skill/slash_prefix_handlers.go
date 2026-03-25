package skill

import (
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/skills"
	"delve-shell/internal/ui"
)

func handleSlashConfigDelSkillPrefix(m ui.Model, rest string) ui.Model {
	lang := "en"
	name := strings.TrimSpace(rest)
	if name == "" {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeyUsageSkillRemove))))
		return m.RefreshViewport()
	}

	if err := skillsvc.Remove(name); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillRemoveFailed, err))))
	} else {
		m.Messages = append(m.Messages, suggestStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillRemoved, name))))
	}
	m = m.ClearSlashInput()
	return m.RefreshViewport()
}

func handleSlashConfigUpdateSkillPrefix(m ui.Model, rest string) ui.Model {
	lang := "en"
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeyDescConfigUpdateSkill))))
		return m.RefreshViewport()
	}

	skillName := fields[0]
	m = openUpdateSkillOverlay(m, skillName)
	m = m.ClearSlashInput()
	return m.RefreshViewport()
}

func handleSlashSkillPrefix(m ui.Model, rest string) ui.Model {
	lang := "en"
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeyUsageSkill))))
		return m
	}

	skillName := fields[0]
	naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
	if naturalLanguage == "" {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeyUsageSkill))))
		return m
	}

	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeySkillNotFound))))
		return m
	}

	skillContent, err := skills.ReadSKILLContent(skillDir)
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillInstallFailed, err))))
		return m
	}

	payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
	if m.Host.Submit(payload) {
		m.Interaction.WaitingForAI = true
	}
	m.Input.SetValue("")
	m.Input.CursorEnd()
	return m
}

func skillInvocationPrompt(skillName, skillContent, naturalLanguage string) string {
	const header = `[Skill invocation] Fulfill the user's request using ONLY the skill below. Use the run_skill tool with this skill's scripts and parameters; do not suggest arbitrary shell commands unless the skill documentation explicitly allows it.`
	return header + "\n\n## Skill: " + skillName + "\n\n" + skillContent + "\n\n## User request\n\n" + naturalLanguage
}
