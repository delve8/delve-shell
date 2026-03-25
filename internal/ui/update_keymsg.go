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
	mm := m
	ks := keySession{m: &mm}
	key := msg.String()

	// Always allow ctrl+c to quit, even during pending approvals or sensitive prompts.
	if key == "ctrl+c" {
		return mm, tea.Quit
	}

	state := mm.currentUIState()
	if state == uiStatePendingSensitive || state == uiStatePendingApproval {
		if handledModel, handled := mm.handlePendingChoiceKey(key); handled {
			return handledModel, nil
		}
	}

	// Slash dropdown navigation should work even if other key paths evolve.
	// Handle it before overlay/key-to-input processing so Up/Down/Enter remain reliable.
	inputVal := ks.inputValue()
	if strings.HasPrefix(inputVal, "/") {
		if key == "enter" {
			if m2, cmd, handled := mm.handleSlashEnterKey(inputVal); handled {
				return m2, cmd
			}
		}
	}

	if m2, cmd, handled := mm.handleOverlayKey(key, msg); handled {
		return m2, cmd
	}

	inputVal = ks.inputValue()
	if ks.handleSlashUpDown(key, inputVal) {
		return mm, nil
	}
	if key == "pgup" || key == "pgdown" {
		return mm, ks.updateViewportKey(msg)
	}

	if key == "enter" {
		text := strings.TrimSpace(inputVal)
		if text == "" {
			return mm, nil
		}
		// WaitingForAI only blocks submitting new messages; slash commands starting with / always run
		if ks.waitingForAI() && !strings.HasPrefix(text, "/") {
			return mm, nil
		}
		_, vis, viewOpts := ks.slashSuggestionTriple(inputVal)
		selected, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, ks.slashSuggestIndex())
		capture := maininput.CaptureSlashSelection(maininput.CaptureInput{
			InputVal:     inputVal,
			Text:         text,
			SuggestIndex: ks.slashSuggestIndex(),
			Selected:     selected,
			HasSelected:  ok,
		})
		if capture.FillOnly {
			ks.setInputValue(capture.FillValue)
			ks.setSlashSuggestIndex(0)
			return mm, nil
		}
		if maininput.IsNewSessionCommand(text) {
			mm = mm.appendUserSubmittedEcho(text)
			_ = mm.Host.Submit(text)
			ks.setInputValue("")
			ks.setSlashSuggestIndex(0)
			mm = mm.RefreshViewport()
			return mm, nil
		}
		mm = mm.appendUserSubmittedEcho(text)
		ks.setInputValue("")
		ks.setSlashSuggestIndex(0)
		return mm.handleMainEnterCommand(text, capture.SelectedIndex)
	}

	cmd := ks.updateTextInput(msg)
	ks.syncSuggestAfterInputChange(ks.inputValue())
	return mm, cmd
}
