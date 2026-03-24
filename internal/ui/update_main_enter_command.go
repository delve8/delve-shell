package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
)

func (m Model) handleMainEnterCommand(text string, slashSelectedPath string, slashSelectedIndex int) (Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		if m2, cmd, handled := m.dispatchSlashExact(text); handled {
			return m2, cmd
		}
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
		return m2, cmd
	}

	if strings.HasPrefix(text, "/") {
		opts := getSlashOptionsForInput(text, m.getLang(), m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
		vis := visibleSlashOptions(text, opts)
		sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
		plan := maininput.PlanMainEnter(maininput.MainEnterInput{
			Text:               text,
			SlashSelectedPath:  slashSelectedPath,
			SlashSelectedIndex: slashSelectedIndex,
			Options:            toSlashViewOptions(opts),
			Visible:            vis,
			SessionNoneMsg:     sessionNoneMsg,
		})
		switch plan.Kind {
		case maininput.MainEnterSwitchSession:
			if m.Ports.SessionSwitchChan != nil {
				select {
				case m.Ports.SessionSwitchChan <- slashSelectedPath:
				default:
				}
			}
			m = m.clearSlashInput()
			m = m.RefreshViewport()
			return m, nil
		case maininput.MainEnterShowSessionNone:
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
			m = m.RefreshViewport()
			m = m.clearSlashInput()
			return m, nil
		case maininput.MainEnterResolveSelected:
			if m2, cmd, handled := m.handleSlashSelectedFallback(plan.Chosen); handled {
				return m2, cmd
			}
			if m2, cmd, handled := m.dispatchSlashExact(plan.Chosen); handled {
				return m2, cmd
			}
			if m2, cmd, handled := m.dispatchSlashPrefix(plan.Chosen); handled {
				return m2, cmd
			}
		case maininput.MainEnterUnknownSlash:
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUnknownCmd))))
			m = m.RefreshViewport()
			return m, nil
		}
	}

	if m.Ports.SubmitChan != nil {
		m.Ports.SubmitChan <- text
		m.Interaction.WaitingForAI = true
	}
	return m, nil
}
