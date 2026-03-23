package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
)

type slashPrefixEntry struct {
	prefix string
	handle func(Model, string) (Model, tea.Cmd, bool) // rest after prefix
}

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	entries := []slashPrefixEntry{
		{
			prefix: "/run ",
			handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
				cmd := strings.TrimSpace(rest)
				if mm.ExecDirectChan != nil && cmd != "" {
					mm.ExecDirectChan <- cmd
				} else if cmd == "" {
					mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageRun))))
				}
				return mm, nil, true
			},
		},
		{
			prefix: "/sessions ",
			handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
				id := strings.TrimSpace(rest)
				if id == "" {
					return mm, nil, true
				}
				if mm.SessionSwitchChan != nil {
					sessionPath := filepath.Join(config.HistoryDir(), id+".jsonl")
					select {
					case mm.SessionSwitchChan <- sessionPath:
					default:
					}
				}
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil, true
			},
		},
		{
			prefix: "/config del-remote ",
			handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
				nameOrTarget := strings.TrimSpace(rest)
				if nameOrTarget == "" {
					return mm, nil, true
				}
				return mm.applyConfigRemoveRemote(nameOrTarget), nil, true
			},
		},
		{
			prefix: "/config del-skill ",
			handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
				name := strings.TrimSpace(rest)
				if name == "" {
					mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageSkillRemove))))
					return mm, nil, true
				}
				if err := skillsvc.Remove(name); err != nil {
					mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.Tf(mm.getLang(), i18n.KeySkillRemoveFailed, err))))
				} else {
					mm.Messages = append(mm.Messages, suggestStyle.Render(mm.delveMsg(i18n.Tf(mm.getLang(), i18n.KeySkillRemoved, name))))
				}
				mm.Messages = append(mm.Messages, "")
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil, true
			},
		},
		{
			prefix: "/remote on ",
			handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
				target := strings.TrimSpace(rest)
				if target == "" {
					return mm, nil, true
				}
				if mm.RemoteOnChan != nil {
					select {
					case mm.RemoteOnChan <- target:
					default:
					}
				}
				return mm, nil, true
			},
		},
	}

	for _, e := range entries {
		if strings.HasPrefix(text, e.prefix) {
			rest := strings.TrimPrefix(text, e.prefix)
			return e.handle(m, rest)
		}
	}
	return m, nil, false
}
