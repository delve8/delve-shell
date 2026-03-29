package ui

import (
	"strings"

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
	if s.m.Interaction.inputHistIndex >= 0 {
		return false
	}
	if !strings.HasPrefix(inputVal, "/") || s.m.Input.LineCount() > 1 || (key != "up" && key != "down") {
		return false
	}
	_, vis, _ := s.slashSuggestionTriple(inputVal)
	if next, changed := slashview.NextSuggestIndex(s.slashSuggestIndex(), len(vis), key); changed {
		s.setSlashSuggestIndex(next)
	}
	return true
}

// handleInputHistoryNav implements bash-like previous/next submitted line.
// While browsing history (inputHistIndex >= 0), Up/Down keep moving through history even if the buffer is
// multiline or starts with / (textarea line nav and slash completion stay disabled until browsing ends).
// When not browsing, multiline drafts keep native Up/Down for moving between lines.
func (s *keySession) handleInputHistoryNav(key string, inputVal string) bool {
	if key != "up" && key != "down" {
		return false
	}
	if s.m.Input.LineCount() > 1 && s.m.Interaction.inputHistIndex < 0 {
		return false
	}
	if strings.HasPrefix(inputVal, "/") && s.m.Interaction.inputHistIndex < 0 {
		return false
	}
	h := s.m.Interaction.inputHistory
	if len(h) == 0 {
		return false
	}
	if key == "up" {
		if s.m.Interaction.inputHistIndex < 0 {
			s.m.Interaction.inputHistScratch = s.m.Input.Value()
			s.m.Interaction.inputHistIndex = len(h) - 1
		} else if s.m.Interaction.inputHistIndex > 0 {
			s.m.Interaction.inputHistIndex--
		} else {
			return true
		}
		s.setInputValue(h[s.m.Interaction.inputHistIndex])
		s.setSlashSuggestIndex(0)
		return true
	}
	// down
	if s.m.Interaction.inputHistIndex < 0 {
		return false
	}
	if s.m.Interaction.inputHistIndex < len(h)-1 {
		s.m.Interaction.inputHistIndex++
		s.setInputValue(h[s.m.Interaction.inputHistIndex])
	} else {
		s.m.Interaction.inputHistIndex = -1
		s.setInputValue(s.m.Interaction.inputHistScratch)
		s.m.Interaction.inputHistScratch = ""
	}
	s.setSlashSuggestIndex(0)
	return true
}

func (m Model) withInputHistoryCommitted(line string) Model {
	line = strings.TrimSpace(line)
	if line == "" {
		return m
	}
	h := append(m.Interaction.inputHistory, line)
	if len(h) > maxInputHistoryEntries {
		h = h[len(h)-maxInputHistoryEntries:]
	}
	m.Interaction.inputHistory = h
	m.Interaction.inputHistIndex = -1
	m.Interaction.inputHistScratch = ""
	return m
}

// appendUserSubmittedEcho appends the same "User: …" transcript line as the main Enter path.
func (m Model) appendUserSubmittedEcho(text string) Model {
	text = strings.TrimSpace(text)
	if text == "" {
		return m
	}
	m = m.withInputHistoryCommitted(text)
	w := m.contentWidth()
	sepLine := renderSeparator(w)
	m.messages = maininput.AppendUserInputLines(m.messages, i18n.T(m.getLang(), i18n.KeyUserLabel), text, w, sepLine)
	return m
}

func (m Model) appendSubmissionError(err error) (Model, tea.Cmd) {
	if err == nil {
		return m, nil
	}
	m.Interaction.WaitingForAI = false
	m = m.AppendTranscriptLines(errStyle.Render(m.delveMsg(err.Error())))
	return m.printTranscriptCmd(false)
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
	if m2, cmd, handled := mm.handleOverlayKey(key, msg); handled {
		return m2, cmd
	}

	inputVal = ks.inputValue()
	if ks.handleInputHistoryNav(key, inputVal) {
		return mm, nil
	}
	if ks.handleSlashUpDown(key, inputVal) {
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
				var errCmd tea.Cmd
				mm, errCmd = mm.appendSubmissionError(err)
				return mm, tea.Sequence(printCmd, errCmd)
			}
			mm, cmd := mm.applyLifecycleResult(res)
			return mm, tea.Sequence(printCmd, cmd)
		}
		return mm, printCmd
	}

	if mm.Interaction.inputHistIndex >= 0 && key != "up" && key != "down" {
		mm.Interaction.inputHistIndex = -1
		mm.Interaction.inputHistScratch = ""
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
	m.Interaction.inputHistIndex = -1
	m.Interaction.inputHistScratch = ""
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
				var cmd tea.Cmd
				m, cmd = m.appendSubmissionError(err)
				return m, cmd, true
			}
			trimmed := strings.TrimSpace(plan.Submission.RawText)
			// /skill … submits chat via host without going through main Enter's appendUserSubmittedEcho.
			// Mirror normal chat: print the user line to scrollback before lifecycle effects so
			// printedMessages stays aligned with tea.Println (fixes AI reply + footer drift).
			var printCmd tea.Cmd
			if strings.HasPrefix(trimmed, "/skill ") {
				m = m.appendUserSubmittedEcho(trimmed)
				m, printCmd = m.printTranscriptCmd(false)
			} else {
				// Other slash submits: no user echo line here, but still record for Up/Down recall (incl. /help, /exec …).
				m = m.withInputHistoryCommitted(trimmed)
			}
			m = m.clearSlashInput()
			returned, cmd := m.applyLifecycleResult(res)
			if printCmd != nil {
				return returned, tea.Sequence(printCmd, cmd), true
			}
			return returned, cmd, true
		}
	}
	return m, nil, false
}
