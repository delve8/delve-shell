package ui

import (
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/skills"
)

// SlashConfigDelSkillPrefix handles prefix command: "/config del-skill <name>".
// It is exported so feature packages can implement slash routing without
// re-creating ui-specific message styling.
func (m Model) SlashConfigDelSkillPrefix(rest string) Model {
	name := strings.TrimSpace(rest)
	if name == "" {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkillRemove))))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}

	if err := skillsvc.Remove(name); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoveFailed, err))))
	} else {
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoved, name))))
	}
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// SlashConfigUpdateSkillPrefix handles prefix command: "/config update-skill <skill-name>".
// It is exported so feature packages can implement slash routing without
// re-creating ui-specific message styling.
func (m Model) SlashConfigUpdateSkillPrefix(rest string) Model {
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyDescConfigUpdateSkill))))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}

	skillName := fields[0]
	m = m.openUpdateSkillOverlay(skillName)
	m = m.ClearSlashInput()
	m = m.RefreshViewport()
	return m
}

// SlashSkillPrefix handles prefix command: "/skill <skill-name> <detail...>".
// It is exported so feature packages can implement slash routing without
// re-creating ui-specific error styling.
func (m Model) SlashSkillPrefix(rest string) Model {
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkill))))
		return m
	}

	skillName := fields[0]
	naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
	if naturalLanguage == "" {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkill))))
		return m
	}

	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeySkillNotFound))))
		return m
	}

	skillContent, err := skills.ReadSKILLContent(skillDir)
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillInstallFailed, err))))
		return m
	}

	payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
	if m.SubmitChan != nil {
		m.SubmitChan <- payload
		m.WaitingForAI = true
	}
	m.Input.SetValue("")
	m.Input.CursorEnd()
	return m
}

// ApplyConfigAddRemoteArgs exposes applyConfigAddRemote to feature packages.
func (m Model) ApplyConfigAddRemoteArgs(args string) Model {
	return m.applyConfigAddRemote(args)
}

// ApplyConfigRemoveRemote exposes applyConfigRemoveRemote to feature packages.
func (m Model) ApplyConfigRemoveRemote(nameOrTarget string) Model {
	return m.applyConfigRemoveRemote(nameOrTarget)
}
