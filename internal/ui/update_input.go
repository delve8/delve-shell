package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/inputpreflight"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"
)

// keySession narrows keyboard handling for [Model.handleKeyMsg] so update_input does not
// scatter direct Model field access.
type keySession struct {
	m *Model
}

func (s *keySession) inputValue() string { return s.m.Input.Value() }

func (s *keySession) setInputValue(v string) {
	s.m.Input.SetValue(v)
	s.m.Input.CursorEnd()
	*s.m = s.m.syncInputHeight()
}

func (s *keySession) slashSuggestIndex() int { return s.m.Interaction.slashSuggestIndex }

func (s *keySession) setSlashSuggestIndex(i int) { s.m.Interaction.slashSuggestIndex = i }

func (s *keySession) waitingForAI() bool { return s.m.Interaction.WaitingForAI }

func (s *keySession) updateViewportKey(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	s.m.Viewport, cmd = s.m.Viewport.Update(msg)
	return cmd
}

func (s *keySession) scrollViewportForKey(key string) bool {
	switch key {
	case "ctrl+u":
		s.m.Viewport.HalfPageUp()
		return true
	case "ctrl+d":
		s.m.Viewport.HalfPageDown()
		return true
	case "alt+up":
		s.m.Viewport.ScrollUp(1)
		return true
	case "alt+down":
		s.m.Viewport.ScrollDown(1)
		return true
	default:
		return false
	}
}

func (s *keySession) updateTextInput(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	s.m.Input, cmd = s.m.Input.Update(msg)
	*s.m = s.m.syncInputHeight()
	return cmd
}

func (s *keySession) slashSuggestionTriple(inputVal string) (opts []SlashOption, vis []int, viewOpts []slashview.Option) {
	return s.m.slashSuggestionContext(inputVal)
}

func (s *keySession) syncSuggestAfterInputChange(inputVal string) {
	_, vis, _ := s.m.slashSuggestionContext(inputVal)
	s.m.Interaction.slashSuggestIndex = maininput.SyncSlashSuggestIndex(maininput.SyncInput{
		InputVal:            inputVal,
		CurrentSuggestIndex: s.m.Interaction.slashSuggestIndex,
		VisibleCount:        len(vis),
	})
}

func (s *keySession) handleSlashUpDown(key string, inputVal string) bool {
	if !strings.HasPrefix(inputVal, "/") || s.m.Input.LineCount() > 1 || (key != "up" && key != "down") {
		return false
	}
	_, vis, _ := s.slashSuggestionTriple(inputVal)
	if next, changed := slashview.NextSuggestIndex(s.slashSuggestIndex(), len(vis), key); changed {
		s.setSlashSuggestIndex(next)
	}
	return true
}

// appendUserSubmittedEcho appends the same "User: …" transcript line as the main Enter path.
func (m Model) appendUserSubmittedEcho(text string) Model {
	text = strings.TrimSpace(text)
	if text == "" {
		return m
	}
	w := m.contentWidth()
	sepLine := renderSeparator(w)
	m.messages = maininput.AppendUserInputLines(m.messages, i18n.T(m.getLang(), i18n.KeyUserLabel), text, w, sepLine)
	return m.RefreshViewport()
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	mm := m
	ks := keySession{m: &mm}
	key := msg.String()

	if key == "ctrl+c" {
		res, err := mm.lifecycleEngine().SubmitControl(inputlifecycletype.ControlSignalInterrupt, inputlifecycletype.SourceKeyboardSignal)
		if err == nil {
			mm, cmd := mm.applyLifecycleResult(res)
			return mm, cmd
		}
		return mm, tea.Quit
	}

	state := mm.currentUIState()
	if state == uiStateChoiceCardAlt || state == uiStateChoiceCard {
		if handledModel, handled := mm.handlePendingChoiceKey(key); handled {
			return handledModel, nil
		}
	}

	inputVal := ks.inputValue()
	if strings.HasPrefix(inputVal, "/") && mm.Input.LineCount() == 1 {
		if key == "enter" {
			if m2, cmd, handled := mm.handleSlashEnterKey(inputVal); handled {
				return m2, cmd
			}
		}
	}

	if key == "esc" {
		res, err := mm.lifecycleEngine().SubmitControl(inputlifecycletype.ControlSignalEsc, inputlifecycletype.SourceKeyboardSignal)
		if err == nil {
			mm, cmd := mm.applyLifecycleResult(res)
			return mm, cmd
		}
	}
	if key == "ctrl+shift+c" || key == "shift+ctrl+c" {
		text := mm.selectedOrVisibleScreenText()
		if text != "" {
			_ = clipboard.WriteAll(text)
		}
		return mm, nil
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
	if ks.scrollViewportForKey(key) {
		return mm, nil
	}

	if key == "enter" {
		text := strings.TrimSpace(inputVal)
		if text == "" {
			return mm, nil
		}
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
		mm = mm.appendUserSubmittedEcho(text)
		var printCmd tea.Cmd
		mm, printCmd = mm.printTranscriptCmd(false)
		ks.setInputValue("")
		ks.setSlashSuggestIndex(0)
		if res, handled, err := mm.lifecycleEngine().SubmitEnter(text, capture.SelectedIndex); handled {
			if err != nil {
				mm = mm.AppendTranscriptLines(errStyle.Render(mm.delveMsg(err.Error())))
				mm = mm.RefreshViewport()
				mm, errCmd := mm.printTranscriptCmd(false)
				return mm, tea.Sequence(printCmd, errCmd)
			}
			mm, cmd := mm.applyLifecycleResult(res)
			return mm, tea.Sequence(printCmd, cmd)
		}
		return mm, printCmd
	}

	cmd := ks.updateTextInput(msg)
	ks.syncSuggestAfterInputChange(ks.inputValue())
	return mm, cmd
}

func (m Model) openHelpOverlay() Model {
	m = m.OpenOverlayFeature("", i18n.T(m.getLang(), i18n.KeyHelpTitle), i18n.T(m.getLang(), i18n.KeyHelpText))
	m = m.InitOverlayViewport()
	return m
}

func (m Model) clearSlashInput() Model {
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.slashSuggestIndex = 0
	return m.syncInputHeight()
}

// handleSlashEnterKey handles Enter when input starts with "/".
func (m Model) handleSlashEnterKey(inputVal string) (Model, tea.Cmd, bool) {
	if strings.TrimSpace(inputVal) == "" {
		return m, nil, false
	}
	if m.Input.LineCount() > 1 {
		return m, nil, false
	}
	_, vis, viewOpts := m.slashSuggestionContext(inputVal)
	selected, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, m.Interaction.slashSuggestIndex)
	plan := inputpreflight.PlanSlashEnter(inputVal, selected, ok, m.Interaction.slashSuggestIndex)
	switch plan.Kind {
	case inputpreflight.EnterPlanFillOnly:
		m.Input.SetValue(plan.FillValue)
		m.Input.CursorEnd()
		m.Interaction.slashSuggestIndex = 0
		return m.syncInputHeight(), nil, true
	case inputpreflight.EnterPlanSubmit:
		if res, handled, err := m.lifecycleEngine().RouteSubmission(plan.Submission); handled {
			if err != nil {
				m = m.AppendTranscriptLines(errStyle.Render(m.delveMsg(err.Error())))
				return m.RefreshViewport(), nil, true
			}
			m = m.clearSlashInput()
			returned, cmd := m.applyLifecycleResult(res)
			return returned, cmd, true
		}
	}
	return m, nil, false
}
