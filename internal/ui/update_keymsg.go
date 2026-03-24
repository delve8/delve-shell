package ui

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"

	tea "github.com/charmbracelet/bubbletea"
)

// appendUserSubmittedEcho appends the same "User: …" transcript line as the main Enter path.
func (m Model) appendUserSubmittedEcho(text string) Model {
	text = strings.TrimSpace(text)
	if text == "" {
		return m
	}
	w := m.contentWidth()
	sepLine := renderSeparator(w)
	m.Messages = maininput.AppendUserInputLines(m.Messages, i18n.T(m.getLang(), i18n.KeyUserLabel), text, w, sepLine)
	return m.RefreshViewport()
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	// Always allow ctrl+c to quit, even during pending approvals or sensitive prompts.
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	state := m.currentUIState()
	if state == uiStatePendingSensitive || state == uiStatePendingApproval {
		if handledModel, handled := m.handlePendingChoiceKey(key); handled {
			return handledModel, nil
		}
	}

	// Slash dropdown navigation should work even if other key paths evolve.
	// Handle it before overlay/key-to-input processing so Up/Down/Enter remain reliable.
	inputVal := m.Input.Value()
	inSlash := strings.HasPrefix(inputVal, "/")
	if inSlash {
		if key == "enter" {
			if m2, cmd, handled := m.handleSlashEnterKey(inputVal); handled {
				return m2, cmd
			}
		}
	}

	if m2, cmd, handled := m.handleOverlayKey(key, msg); handled {
		return m2, cmd
	}

	inputVal = m.Input.Value()
	inSlash = strings.HasPrefix(inputVal, "/")
	if inSlash && (key == "up" || key == "down") {
		opts := getSlashOptionsForInput(inputVal, m.getLang(), m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
		vis := visibleSlashOptions(inputVal, opts)
		if next, changed := slashview.NextSuggestIndex(m.Interaction.SlashSuggestIndex, len(vis), key); changed {
			m.Interaction.SlashSuggestIndex = next
		}
		return m, nil
	}
	if key == "pgup" || key == "pgdown" {
		var cmd tea.Cmd
		m.Viewport, cmd = m.Viewport.Update(msg)
		return m, cmd
	}

	if key == "enter" {
		text := strings.TrimSpace(inputVal)
		if text == "" {
			return m, nil
		}
		// WaitingForAI only blocks submitting new messages; slash commands starting with / always run
		if m.Interaction.WaitingForAI && !strings.HasPrefix(text, "/") {
			return m, nil
		}
		opts := getSlashOptionsForInput(inputVal, m.getLang(), m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
		vis := visibleSlashOptions(inputVal, opts)
		selected, ok := slashview.SelectedByVisibleIndex(toSlashViewOptions(opts), vis, m.Interaction.SlashSuggestIndex)
		capture := maininput.CaptureSlashSelection(maininput.CaptureInput{
			InputVal:     inputVal,
			Text:         text,
			SuggestIndex: m.Interaction.SlashSuggestIndex,
			Selected:     selected,
			HasSelected:  ok,
		})
		if capture.FillOnly {
			m.Input.SetValue(capture.FillValue)
			m.Input.CursorEnd()
			m.Interaction.SlashSuggestIndex = 0
			return m, nil
		}
		if maininput.IsNewSessionCommand(text) {
			m = m.appendUserSubmittedEcho(text)
			if m.Ports.SubmitChan != nil {
				m.Ports.SubmitChan <- text
			}
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.Interaction.SlashSuggestIndex = 0
			m = m.RefreshViewport()
			return m, nil
		}
		m = m.appendUserSubmittedEcho(text)
		m.Input.SetValue("")
		m.Input.CursorEnd()
		m.Interaction.SlashSuggestIndex = 0
		return m.handleMainEnterCommand(text, capture.SelectedIndex)
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	inputVal = m.Input.Value()
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	m.Interaction.SlashSuggestIndex = maininput.SyncSlashSuggestIndex(maininput.SyncInput{
		InputVal:            inputVal,
		CurrentSuggestIndex: m.Interaction.SlashSuggestIndex,
		VisibleCount:        len(vis),
	})
	return m, cmd
}
