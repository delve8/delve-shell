package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/route"
	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
)

func (m Model) handleMainEnterCommand(text string, slashSelectedIndex int) (Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		if m.Host.TryRelaySlashSubmit(route.SlashSubmitPayload{
			RawLine:            text,
			SlashSelectedIndex: slashSelectedIndex,
		}) {
			return m, nil
		}
	}
	return m.executeMainEnterCommandNoRelay(text, slashSelectedIndex)
}

// executeMainEnterCommandNoRelay runs the main Enter path without the bus relay (used after SlashSubmitRelayMsg).
func (m Model) executeMainEnterCommandNoRelay(text string, slashSelectedIndex int) (Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		if m2, cmd, handled := m.dispatchSlashExact(text); handled {
			return m2, cmd
		}
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
		return m2, cmd
	}

	if strings.HasPrefix(text, "/") {
		_, vis, viewOpts := m.slashSuggestionContext(text)
		sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
		delRemoteNoneMsg := i18n.T(m.getLang(), i18n.KeyDelRemoteNoHosts)
		plan := maininput.PlanMainEnter(maininput.MainEnterInput{
			Text:               text,
			SlashSelectedIndex: slashSelectedIndex,
			Options:            viewOpts,
			Visible:            vis,
			SessionNoneMsg:     sessionNoneMsg,
			DelRemoteNoneMsg:   delRemoteNoneMsg,
		})
		switch plan.Kind {
		case maininput.MainEnterShowSessionNone:
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
			m = m.RefreshViewport()
			m = m.clearSlashInput()
			return m, nil
		case maininput.MainEnterShowDelRemoteNone:
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(delRemoteNoneMsg)))
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

	if m.Host.Submit(text) {
		m.Interaction.WaitingForAI = true
	}
	return m, nil
}
