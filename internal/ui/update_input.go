package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/input/maininput"
	"delve-shell/internal/input/preflight"
	"delve-shell/internal/slash/view"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui/uivm"
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
	s.m.syncInputHeight()
}

func (s *keySession) slashSuggestIndex() int { return s.m.Interaction.slashSuggestIndex }

func (s *keySession) setSlashSuggestIndex(i int) { s.m.Interaction.slashSuggestIndex = i }

func (s *keySession) waitingForAI() bool { return s.m.Interaction.WaitingForAI }

func (s *keySession) updateTextInput(msg tea.KeyMsg) tea.Cmd {
	// bubbles/textarea repositions its inner viewport using the current height. While
	// height is still 1, InsertNewline moves the cursor to logical line 2 and triggers
	// ScrollDown so only that line fits—then we grow the widget and the first line stays
	// off-screen until Up. Expand height before Update so repositionView sees the final size.
	if key.Matches(msg, s.m.Input.KeyMap.InsertNewline) && s.m.Input.LineCount() == 1 {
		s.m.Input.SetHeight(inputTextareaMaxHeight)
	}
	var cmd tea.Cmd
	s.m.Input, cmd = s.m.Input.Update(msg)
	s.m.syncInputHeight()
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
	if !strings.HasPrefix(inputVal, "/") || s.m.Input.LineCount() > 1 || (key != teakey.Up && key != teakey.Down) {
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
	if key != teakey.Up && key != teakey.Down {
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
	if key == teakey.Up {
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

func (m *Model) withInputHistoryCommitted(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	h := append(m.Interaction.inputHistory, line)
	if len(h) > maxInputHistoryEntries {
		h = h[len(h)-maxInputHistoryEntries:]
	}
	m.Interaction.inputHistory = h
	m.Interaction.inputHistIndex = -1
	m.Interaction.inputHistScratch = ""
}

// appendUserSubmittedEcho appends the same user transcript echo ("> …" with band style) as the main Enter path.
func (m *Model) appendUserSubmittedEcho(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	m.withInputHistoryCommitted(text)
	m.appendUserTranscriptLine(text)
}

// appendUserTranscriptLine appends the user transcript row only (no input history).
// Used when [withInputHistoryCommitted] or equivalent already ran (e.g. slash Enter before /bash quit).
func (m *Model) appendUserTranscriptLine(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	lines := make([]uivm.Line, 0, 3)
	if len(m.messages) > 0 && !isRenderedShortSeparator(m.messages[len(m.messages)-1]) {
		lines = append(lines, uivm.Line{Kind: uivm.LineSeparator})
	}
	lines = append(lines,
		uivm.Line{Kind: uivm.LineUser, Text: text},
		uivm.Line{Kind: uivm.LineBlank},
	)
	m.appendSemanticTranscriptLines(lines...)
}

func (m *Model) appendSubmissionError(err error) (*Model, tea.Cmd) {
	if err == nil {
		return m, nil
	}
	m.Interaction.WaitingForAI = false
	m.appendSemanticTranscriptLines(uivm.Line{Kind: uivm.LineSystemError, Text: err.Error()})
	return m, m.printTranscriptCmd(false)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (*Model, tea.Cmd) {
	ks := keySession{m: m}
	key := msg.String()

	if key == "ctrl+c" {
		res, err := m.lifecycleEngine().SubmitControl(inputlifecycletype.ControlSignalInterrupt, inputlifecycletype.SourceKeyboardSignal)
		if err == nil {
			return m.applyLifecycleResult(res)
		}
		return m, tea.Quit
	}

	if m.currentUIState() == uiStateOfflinePaste {
		return m.handleOfflinePasteKeyMsg(msg)
	}

	state := m.currentUIState()
	if state == uiStateChoiceCardAlt || state == uiStateChoiceCard {
		if handledModel, cmd, handled := m.handlePendingChoiceKey(msg); handled {
			return handledModel, cmd
		}
	}

	if m.Interaction.CommandExecuting {
		if key == teakey.Esc {
			res, err := m.lifecycleEngine().SubmitControl(inputlifecycletype.ControlSignalEsc, inputlifecycletype.SourceKeyboardSignal)
			if err == nil {
				return m.applyLifecycleResult(res)
			}
		}
		return m, nil
	}

	inputVal := ks.inputValue()
	if strings.HasPrefix(inputVal, "/") && m.Input.LineCount() == 1 {
		if key == teakey.Enter {
			if m2, cmd, handled := m.handleSlashEnterKey(inputVal); handled {
				return m2, cmd
			}
		}
		if key == teakey.Tab {
			if m2, cmd, handled := m.handleSlashTabKey(inputVal); handled {
				return m2, cmd
			}
		}
	}

	if key == teakey.Esc {
		res, err := m.lifecycleEngine().SubmitControl(inputlifecycletype.ControlSignalEsc, inputlifecycletype.SourceKeyboardSignal)
		if err == nil {
			return m.applyLifecycleResult(res)
		}
	}
	if m2, cmd, handled := m.handleOverlayKey(key, msg); handled {
		return m2, cmd
	}

	inputVal = ks.inputValue()
	if ks.handleInputHistoryNav(key, inputVal) {
		return m, nil
	}
	if ks.handleSlashUpDown(key, inputVal) {
		return m, nil
	}

	if key == teakey.Enter {
		text := strings.TrimSpace(inputVal)
		if text == "" {
			return m, nil
		}
		if ks.waitingForAI() && !strings.HasPrefix(text, "/") {
			return m, nil
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
			return m, nil
		}
		m.appendUserSubmittedEcho(text)
		printCmd := m.printTranscriptCmd(false)
		ks.setInputValue("")
		ks.setSlashSuggestIndex(0)
		if res, handled, err := m.lifecycleEngine().SubmitEnter(text, capture.SelectedIndex); handled {
			if err != nil {
				var errCmd tea.Cmd
				m, errCmd = m.appendSubmissionError(err)
				return m, tea.Sequence(printCmd, errCmd)
			}
			m2, cmd := m.applyLifecycleResult(res)
			return m2, tea.Sequence(printCmd, cmd)
		}
		return m, printCmd
	}

	if m.Interaction.inputHistIndex >= 0 && key != teakey.Up && key != teakey.Down {
		m.Interaction.inputHistIndex = -1
		m.Interaction.inputHistScratch = ""
	}

	cmd := ks.updateTextInput(msg)
	ks.syncSuggestAfterInputChange(ks.inputValue())
	return m, cmd
}

func (m *Model) clearSlashInput() {
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.slashSuggestIndex = 0
	m.Interaction.inputHistIndex = -1
	m.Interaction.inputHistScratch = ""
	m.syncInputHeight()
}

// handleSlashTabKey applies slash completion fill when Tab is pressed; it never submits.
// When no fill applies, the key is still consumed so a literal tab is not inserted.
func (m *Model) handleSlashTabKey(inputVal string) (*Model, tea.Cmd, bool) {
	if strings.TrimSpace(inputVal) == "" {
		return m, nil, false
	}
	if m.Input.LineCount() > 1 {
		return m, nil, false
	}
	_, vis, viewOpts := m.slashSuggestionContext(inputVal)
	selected, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, m.Interaction.slashSuggestIndex)
	plan := inputpreflight.PlanSlashEnter(inputVal, selected, ok, m.Interaction.slashSuggestIndex)
	if plan.Kind == inputpreflight.EnterPlanFillOnly {
		m.Input.SetValue(plan.FillValue)
		m.Input.CursorEnd()
		m.Interaction.slashSuggestIndex = 0
		m.syncInputHeight()
		return m, nil, true
	}
	return m, nil, true
}

// handleSlashEnterKey handles Enter when input starts with "/".
func (m *Model) handleSlashEnterKey(inputVal string) (*Model, tea.Cmd, bool) {
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
		m.syncInputHeight()
		return m, nil, true
	case inputpreflight.EnterPlanSubmit:
		trimmed := strings.TrimSpace(plan.Submission.RawText)
		var printCmd tea.Cmd
		// Mirror normal Enter: slash submissions echo before execution so all runtime paths,
		// including /quit and /bash shell snapshots, observe the same transcript/history state.
		if plan.Submission.Kind == inputlifecycletype.SubmissionSlash {
			m.appendUserSubmittedEcho(trimmed)
			printCmd = m.printTranscriptCmd(false)
		}
		if res, handled, err := m.lifecycleEngine().RouteSubmission(plan.Submission); handled {
			if err != nil {
				var cmd tea.Cmd
				m, cmd = m.appendSubmissionError(err)
				if printCmd != nil {
					return m, tea.Sequence(printCmd, cmd), true
				}
				return m, cmd, true
			}
			m.clearSlashInput()
			returned, cmd := m.applyLifecycleResult(res)
			if printCmd != nil {
				return returned, tea.Sequence(printCmd, cmd), true
			}
			return returned, cmd, true
		}
	}
	return m, nil, false
}
