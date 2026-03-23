package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
)

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	switch {
	case strings.HasPrefix(text, "/run "):
		cmd := strings.TrimSpace(text[len("/run "):])
		if m.ExecDirectChan != nil && cmd != "" {
			m.ExecDirectChan <- cmd
		} else if cmd == "" {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageRun))))
		}
		return m, nil, true
	case strings.HasPrefix(text, "/sessions "):
		id := strings.TrimSpace(strings.TrimPrefix(text, "/sessions "))
		if id == "" {
			return m, nil, true
		}
		if m.SessionSwitchChan != nil {
			sessionPath := filepath.Join(config.HistoryDir(), id+".jsonl")
			select {
			case m.SessionSwitchChan <- sessionPath:
			default:
			}
		}
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil, true
	case strings.HasPrefix(text, "/config del-remote "):
		nameOrTarget := strings.TrimSpace(strings.TrimPrefix(text, "/config del-remote "))
		if nameOrTarget == "" {
			return m, nil, true
		}
		return m.applyConfigRemoveRemote(nameOrTarget), nil, true
	case strings.HasPrefix(text, "/config del-skill "):
		name := strings.TrimSpace(strings.TrimPrefix(text, "/config del-skill "))
		if name == "" {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkillRemove))))
			return m, nil, true
		}
		if err := skillsvc.Remove(name); err != nil {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoveFailed, err))))
		} else {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoved, name))))
		}
		m.Messages = append(m.Messages, "")
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil, true
	case strings.HasPrefix(text, "/remote on "):
		target := strings.TrimSpace(strings.TrimPrefix(text, "/remote on "))
		if target == "" {
			return m, nil, true
		}
		if m.RemoteOnChan != nil {
			select {
			case m.RemoteOnChan <- target:
			default:
			}
		}
		return m, nil, true
	}
	return m, nil, false
}
