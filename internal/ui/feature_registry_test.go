package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

// Test-only fallback registrations so internal/ui unit tests can run
// without importing feature packages (which would create import cycles).
func init() {
	registerSlashExact("/config add-remote", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openAddRemoteOverlay(true, false), nil
		},
		ClearInput: true,
	})
	registerSlashExact("/remote on", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openAddRemoteOverlay(false, true), nil
		},
		ClearInput: true,
	})
	registerSlashPrefix("/config update-skill", SlashPrefixDispatchEntry{
		Prefix: "/config update-skill",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyDescConfigUpdateSkill))))
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil, true
			}
			skillName := fields[0]
			mm.OverlayActive = true
			mm.OverlayTitle = "Update skill"
			mm.UpdateSkillActive = true
			mm.UpdateSkillName = skillName
			mm.UpdateSkillError = ""
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.SlashSuggestIndex = 0
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})
}
