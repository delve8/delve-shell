package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"
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

		var selectedOpt slashOption
		if selected, ok := slashview.SelectedByVisibleIndex(toSlashViewOptions(opts), vis, slashSelectedIndex); ok {
			selectedOpt = slashOption{Cmd: selected.Cmd, Path: selected.Path}
		}

		sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
		outcome := slashflow.EvaluateMainEnter(text, slashflow.EnterInput{
			HasSlashPrefix:      true,
			SelectedPath:        slashSelectedPath,
			SelectedCmd:         selectedOpt.Cmd,
			VisibleOptionCount:  len(vis),
			IsSessionNoneOption: selectedOpt.Path == "" && selectedOpt.Cmd == sessionNoneMsg,
		})

		if m2, cmd, handled := m.handleSlashOutcome(outcome, slashSelectedPath, selectedOpt.Cmd, sessionNoneMsg); handled {
			return m2, cmd
		}
	}

	if m.Ports.SubmitChan != nil {
		m.Ports.SubmitChan <- text
		m.Interaction.WaitingForAI = true
	}
	return m, nil
}

func (m Model) handleSlashOutcome(outcome slashflow.Outcome, selectedPath string, chosen string, sessionNoneMsg string) (Model, tea.Cmd, bool) {
	switch outcome {
	case slashflow.OutcomeSwitchSession:
		if m.Ports.SessionSwitchChan != nil {
			select {
			case m.Ports.SessionSwitchChan <- selectedPath:
			default:
			}
		}
		m = m.clearSlashInput()
		m = m.RefreshViewport()
		return m, nil, true
	case slashflow.OutcomeShowSessionNone:
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
		m = m.RefreshViewport()
		m = m.clearSlashInput()
		return m, nil, true
	case slashflow.OutcomeResolveSelected:
		return m.resolveSelectedSlash(chosen)
	case slashflow.OutcomeUnknownSlash:
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUnknownCmd))))
		m = m.RefreshViewport()
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m Model) resolveSelectedSlash(chosen string) (Model, tea.Cmd, bool) {
	// Let feature providers decide fill-only slash options first.
	if m2, cmd, handled := m.handleSlashSelectedFallback(chosen); handled {
		return m2, cmd, true
	}
	if m2, cmd, handled := m.dispatchSlashExact(chosen); handled {
		return m2, cmd, true
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(chosen); handled {
		return m2, cmd, true
	}
	return m, nil, false
}
