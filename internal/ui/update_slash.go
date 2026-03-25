package ui

import (
	"strings"

	"delve-shell/internal/hostnotify"
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
	m.Interaction.SlashSuggestIndex = 0
	return m
}

// dispatchSlashExact routes exact slash commands through a single table-driven path.
// clearInput controls whether the slash input is consumed after execution.
func (m Model) dispatchSlashExact(cmd string) (Model, tea.Cmd, bool) {
	entry, ok := slashExactDispatchRegistry.Get(cmd)
	if !ok {
		return m, nil, false
	}
	m, outCmd := entry.Handle(m)
	if entry.ClearInput {
		m = m.clearSlashInput()
	}
	return m, outCmd, true
}

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry.Entries() {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			return e.Handle(m, rest)
		}
	}
	return m, nil, false
}

// handleSlashEnterKey handles Enter when input starts with "/".
// Apply the highlighted suggestion (fill or exact) before dispatchSlashExact(trimmed),
// so a row like "/config del-remote host" is not overridden by exact "/config del-remote".
// Lists that mix a parent row with longer rows should put the parent first (see remote slash options).
func (m Model) handleSlashEnterKey(inputVal string) (Model, tea.Cmd, bool) {
	trimmed := strings.TrimSpace(inputVal)
	if trimmed == "" {
		return m, nil, false
	}
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, hostnotify.RemoteActive())
	vis := visibleSlashOptions(inputVal, opts)
	selected, ok := slashview.SelectedByVisibleIndex(toSlashViewOptions(opts), vis, m.Interaction.SlashSuggestIndex)
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
		m.Interaction.SlashSuggestIndex = 0
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
	for _, p := range slashSelectedProviderChain.List() {
		if m2, cmd, handled := p(m, chosen); handled {
			return m2, cmd, true
		}
	}
	return m, nil, false
}
