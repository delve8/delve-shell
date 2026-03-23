package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
		if m2, cmd, handled := m.handleSlashNavigationKey(key, msg, inputVal); handled {
			return m2, cmd
		}
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
	if m2, cmd, handled := m.handleMainScrollKey(key, msg, inputVal); handled {
		return m2, cmd
	}

	if key == "enter" {
		text := strings.TrimSpace(inputVal)
		if text == "" {
			return m, nil
		}
		// WaitingForAI only blocks submitting new messages; slash commands starting with / always run
		if m.WaitingForAI && !strings.HasPrefix(text, "/") {
			return m, nil
		}
		// Save selected slash option before any state change; Enter handler resets SlashSuggestIndex below.
		// Use inputVal (not text) so we match what the view shows and get correct opts/vis with trailing space.
		var slashSelectedPath string
		var slashSelectedIndex int
		var filled bool
		m, slashSelectedPath, slashSelectedIndex, filled = m.captureSlashSelectionForEnter(inputVal, text)
		if filled {
			return m, nil
		}
		if m2, handled := m.handleNewSessionCommandIfNeeded(text); handled {
			return m2, nil
		}

		m = m.appendUserInputLine(text)
		return m.handleMainEnterCommand(text, slashSelectedPath, slashSelectedIndex)
	}

	return m.handleMainInputUpdate(msg)
}
