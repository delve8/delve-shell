package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			return e.Handle(m, rest)
		}
	}
	return m, nil, false
}

func init() {
	// NOTE: order matters for prefix overlaps. Keep it explicit and deterministic.
	registerSlashPrefix("/run ", SlashPrefixDispatchEntry{
		Prefix: "/run ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			cmd := strings.TrimSpace(rest)
			if mm.ExecDirectChan != nil && cmd != "" {
				mm.ExecDirectChan <- cmd
			} else if cmd == "" {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageRun))))
			}
			return mm, nil, true
		},
	})

	registerSlashPrefix("/config llm base_url ", SlashPrefixDispatchEntry{
		Prefix: "/config llm base_url ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigLLM("base_url", strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config llm api_key ", SlashPrefixDispatchEntry{
		Prefix: "/config llm api_key ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigLLM("api_key", strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config llm model ", SlashPrefixDispatchEntry{
		Prefix: "/config llm model ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigLLM("model", strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config auto-run ", SlashPrefixDispatchEntry{
		Prefix: "/config auto-run ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigAllowlistAutoRun(strings.TrimSpace(rest))
			return mm, nil, true
		},
	})

	// Keep update-skill prefix registration in ui so ui unit tests
	// can run without relying on feature package init() registration.
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
			mm = mm.openUpdateSkillOverlay(skillName)
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.SlashSuggestIndex = 0
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})

	// NOTE: skill/remote/session prefix handlers moved to feature packages.
}
