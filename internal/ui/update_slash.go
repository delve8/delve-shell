package ui

import (
	"strings"

	"delve-shell/internal/host/route"
	"delve-shell/internal/i18n"
	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) openHelpOverlay() Model {
	m = m.OpenOverlay(i18n.T(m.getLang(), i18n.KeyHelpTitle), i18n.T(m.getLang(), i18n.KeyHelpText))
	m = m.InitOverlayViewport()
	return m
}

func (m Model) clearSlashInput() Model {
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.slashSuggestIndex = 0
	return m
}

// dispatchSlashExact routes exact slash commands through a single table-driven path.
// clearInput controls whether the slash input is consumed after execution.
func (m Model) dispatchSlashExact(cmd string) (Model, tea.Cmd, bool) {
	entry, ok := slashExactDispatchRegistry.Get(cmd)
	if !ok {
		return m, nil, false
	}
	m.Host.RequestSlashDispatch(cmd)
	m, outCmd := entry.Handle(m)
	if entry.ClearInput {
		m = m.clearSlashInput()
	}
	m.Host.TraceSlashEntered(cmd)
	return m, outCmd, true
}

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry.Entries() {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			m.Host.RequestSlashDispatch(text)
			m2, outCmd, handled := e.Handle(m, rest)
			if handled {
				m2.Host.TraceSlashEntered(text)
			}
			return m2, outCmd, handled
		}
	}
	return m, nil, false
}

// handleSlashEnterKey handles Enter when input starts with "/".
// Apply the highlighted suggestion (fill or exact) before dispatchSlashExact(trimmed),
// so a row like "/config del-remote host" is not overridden by exact "/config del-remote".
// Lists that mix a parent row with longer rows should put the parent first (see remote slash options).
func (m Model) handleSlashEnterKey(inputVal string) (Model, tea.Cmd, bool) {
	if strings.TrimSpace(inputVal) == "" {
		return m, nil, false
	}
	if m.Host.TryRelaySlashSubmit(route.SlashSubmitPayload{
		RawLine:            strings.TrimSpace(inputVal),
		SlashSelectedIndex: m.Interaction.slashSuggestIndex,
		InputLine:          inputVal,
	}) {
		return m, nil, true
	}
	return m.execSlashEnterKeyLocal(inputVal)
}

// execSlashEnterKeyLocal runs slash-mode Enter without bus relay (after SlashSubmitRelayMsg with InputLine, or when relay is unwired).
func (m Model) execSlashEnterKeyLocal(inputVal string) (Model, tea.Cmd, bool) {
	trimmed := strings.TrimSpace(inputVal)
	if trimmed == "" {
		return m, nil, false
	}
	_, vis, viewOpts := m.slashSuggestionContext(inputVal)
	selected, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, m.Interaction.slashSuggestIndex)
	result := slashflow.EvaluateSlashEnter(inputVal, trimmed, selected, ok)
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

// handleSlashSelectedFallback handles suggestion-selected slash commands
// that are intentionally not routed through exact/prefix dispatcher.
func (m Model) handleSlashSelectedFallback(chosen string) (Model, tea.Cmd, bool) {
	m.Host.RequestSlashDispatch(chosen)
	for _, p := range slashSelectedProviderChain.List() {
		if m2, cmd, handled := p(m, chosen); handled {
			m2.Host.TraceSlashEntered(chosen)
			return m2, cmd, true
		}
	}
	return m, nil, false
}
