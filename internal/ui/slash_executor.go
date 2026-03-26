package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"
	"delve-shell/internal/uiflow/enterflow"
)

// executeSlashSubmission runs one normalized slash submission against the local slash registry.
func (m Model) executeSlashSubmission(rawText string, selectedIndex int) (Model, tea.Cmd) {
	text := strings.TrimSpace(rawText)
	if text == "" {
		return m, nil
	}

	if m2, cmd, handled := m.dispatchSlashExact(text); handled {
		return m2, cmd
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
		return m2, cmd
	}

	_, vis, viewOpts := m.slashSuggestionContext(text)
	sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
	delRemoteNoneMsg := i18n.T(m.getLang(), i18n.KeyDelRemoteNoHosts)
	plan := enterflow.PlanAfterSlashDispatches(text, selectedIndex, viewOpts, vis, sessionNoneMsg, delRemoteNoneMsg)
	switch plan.Kind {
	case maininput.MainEnterShowSessionNone:
		m = m.AppendTranscriptLines(suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
		m = m.RefreshViewport()
		m = m.clearSlashInput()
		return m, nil
	case maininput.MainEnterShowDelRemoteNone:
		m = m.AppendTranscriptLines(suggestStyle.Render(m.delveMsg(delRemoteNoneMsg)))
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
		m = m.AppendTranscriptLines(errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUnknownCmd))))
		m = m.RefreshViewport()
		return m, nil
	}

	if m.EmitChatSubmitIntent(text, inputlifecycletype.SourceMainEnter) {
		m.Interaction.WaitingForAI = true
	}
	return m, nil
}

// executeSlashEarlySubmission runs slash-mode Enter after lifecycle submission routing.
func (m Model) executeSlashEarlySubmission(inputLine string) (Model, tea.Cmd, bool) {
	trimmed := strings.TrimSpace(inputLine)
	if trimmed == "" {
		return m, nil, false
	}
	_, vis, viewOpts := m.slashSuggestionContext(inputLine)
	selected, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, m.Interaction.slashSuggestIndex)
	result := slashflow.EvaluateSlashEnter(inputLine, trimmed, selected, ok)
	switch result.Action {
	case slashflow.EnterKeyDispatchExactChosen:
		if _, regOK := slashExactDispatchRegistry.Get(selected.Cmd); regOK {
			m = m.appendUserSubmittedEcho(trimmed)
		}
		if m2, cmd, handled := m.dispatchSlashExact(selected.Cmd); handled {
			return m2, cmd, true
		}
	case slashflow.EnterKeyFillOnly:
		m.Input.SetValue(result.Fill)
		m.Input.CursorEnd()
		m.Interaction.slashSuggestIndex = 0
		return m, nil, true
	}
	if _, regOK := slashExactDispatchRegistry.Get(trimmed); regOK {
		m = m.appendUserSubmittedEcho(trimmed)
	}
	if m2, cmd, handled := m.dispatchSlashExact(trimmed); handled {
		return m2, cmd, true
	}
	return m, nil, false
}

// dispatchSlashExact routes exact slash commands through a single table-driven path.
// clearInput controls whether the slash input is consumed after execution.
func (m Model) dispatchSlashExact(cmd string) (Model, tea.Cmd, bool) {
	entry, ok := slashExactDispatchRegistry.Get(cmd)
	if !ok {
		return m, nil, false
	}
	m.requestSlashDispatchAction(cmd)
	m, outCmd := entry.Handle(m)
	if entry.ClearInput {
		m = m.clearSlashInput()
	}
	m.traceSlashEnteredAction(cmd)
	return m, outCmd, true
}

// dispatchSlashPrefix handles slash commands with arguments.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry.Entries() {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			m.requestSlashDispatchAction(text)
			m2, outCmd, handled := e.Handle(m, rest)
			if handled {
				m2 = m2.clearSlashInput()
				m2.traceSlashEnteredAction(text)
			}
			return m2, outCmd, handled
		}
	}
	return m, nil, false
}

// handleSlashSelectedFallback handles suggestion-selected slash commands
// that are intentionally not routed through exact/prefix dispatcher.
func (m Model) handleSlashSelectedFallback(chosen string) (Model, tea.Cmd, bool) {
	m.requestSlashDispatchAction(chosen)
	for _, p := range slashSelectedProviderChain.List() {
		if m2, cmd, handled := p(m, chosen); handled {
			m2.traceSlashEnteredAction(chosen)
			return m2, cmd, true
		}
	}
	return m, nil, false
}
