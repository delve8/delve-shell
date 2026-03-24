package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func registerTestSkillPrefixMirrors() {
	RegisterSlashPrefix("/config update-skill", SlashPrefixDispatchEntry{
		Prefix: "/config update-skill",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyDescConfigUpdateSkill))))
				mm = mm.RefreshViewport()
				return mm, nil, true
			}
			skillName := fields[0]
			mm = mm.OpenOverlay("Update skill", "")
			mm.UpdateSkill.Active = true
			mm.UpdateSkill.Name = skillName
			mm.UpdateSkill.Error = ""
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.Interaction.SlashSuggestIndex = 0
			mm = mm.RefreshViewport()
			return mm, nil, true
		},
	})
}

func registerTestConfigPrefixMirrors() {
	RegisterSlashPrefix("/config auto-run ", SlashPrefixDispatchEntry{
		Prefix: "/config auto-run ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = applyTestConfigAllowlistAutoRun(mm, strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
}

func registerTestSlashSelectedMirrors() {
	RegisterSlashSelectedProvider(func(m Model, chosen string) (Model, tea.Cmd, bool) {
		if !strings.HasPrefix(chosen, "/skill ") {
			return m, nil, false
		}
		m.Input.SetValue(chosen + " ")
		m.Input.CursorEnd()
		m.Interaction.SlashSuggestIndex = 0
		return m, nil, true
	})

	RegisterSlashSelectedProvider(func(m Model, chosen string) (Model, tea.Cmd, bool) {
		if chosen != SlashRunUsageOption {
			return m, nil, false
		}
		m.Input.SetValue("/run ")
		m.Input.CursorEnd()
		return m, nil, true
	})
}
