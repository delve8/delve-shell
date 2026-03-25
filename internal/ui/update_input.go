package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/inputpreflight"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"
	"delve-shell/internal/uiflow/enterflow"
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
}

func (s *keySession) slashSuggestIndex() int { return s.m.Interaction.slashSuggestIndex }

func (s *keySession) setSlashSuggestIndex(i int) { s.m.Interaction.slashSuggestIndex = i }

func (s *keySession) waitingForAI() bool { return s.m.Interaction.WaitingForAI }

func (s *keySession) updateViewportKey(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	s.m.Viewport, cmd = s.m.Viewport.Update(msg)
	return cmd
}

func (s *keySession) updateTextInput(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	s.m.Input, cmd = s.m.Input.Update(msg)
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
	if !strings.HasPrefix(inputVal, "/") || (key != "up" && key != "down") {
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

	if key == "esc" {
		inputVal = ks.inputValue()
		if strings.HasPrefix(inputVal, "/") && strings.TrimSpace(inputVal) != "" {
			mm = mm.clearSlashInput()
			return mm, nil
		}
		if ks.waitingForAI() {
			res, err := mm.lifecycleEngine().SubmitControl(inputlifecycletype.ControlSignalEsc, inputlifecycletype.SourceKeyboardSignal)
			if err == nil {
				mm.Interaction.WaitingForAI = false
				mm, cmd := mm.applyLifecycleResult(res)
				return mm, cmd
			}
			return mm, nil
		}
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
		ks.setInputValue("")
		ks.setSlashSuggestIndex(0)
		if !strings.HasPrefix(text, "/") {
			if res, handled, err := mm.lifecycleEngine().SubmitEnter(text, capture.SelectedIndex); handled && err == nil {
				mm, cmd := mm.applyLifecycleResult(res)
				return mm, cmd
			}
		}
		return mm.handleMainEnterCommand(text, capture.SelectedIndex)
	}

	cmd := ks.updateTextInput(msg)
	ks.syncSuggestAfterInputChange(ks.inputValue())
	return mm, cmd
}

func (m Model) handleMainEnterCommand(text string, slashSelectedIndex int) (Model, tea.Cmd) {
	if strings.HasPrefix(strings.TrimSpace(text), "/") {
		if res, handled, err := m.submitLifecycleSlash(text, text, slashSelectedIndex, inputlifecycletype.SourceMainEnter); handled {
			if err != nil {
				m = m.AppendTranscriptLines(errStyle.Render(m.delveMsg(err.Error())))
				return m.RefreshViewport(), nil
			}
			return m.applyLifecycleResult(res)
		}
	}
	return m.executeMainEnterCommandNoRelay(text, slashSelectedIndex)
}

// executeMainEnterCommandNoRelay runs the main Enter path without re-entering lifecycle routing.
func (m Model) executeMainEnterCommandNoRelay(text string, slashSelectedIndex int) (Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		if m2, cmd, handled := m.dispatchSlashExact(text); handled {
			return m2, cmd
		}
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
		return m2, cmd
	}

	if strings.HasPrefix(text, "/") {
		_, vis, viewOpts := m.slashSuggestionContext(text)
		sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
		delRemoteNoneMsg := i18n.T(m.getLang(), i18n.KeyDelRemoteNoHosts)
		plan := enterflow.PlanAfterSlashDispatches(text, slashSelectedIndex, viewOpts, vis, sessionNoneMsg, delRemoteNoneMsg)
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
	}

	if m.EmitChatSubmitIntent(text, inputlifecycletype.SourceMainEnter) {
		m.Interaction.WaitingForAI = true
	}
	return m, nil
}

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
	m.requestSlashDispatchAction(cmd)
	m, outCmd := entry.Handle(m)
	if entry.ClearInput {
		m = m.clearSlashInput()
	}
	m.traceSlashEnteredAction(cmd)
	return m, outCmd, true
}

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
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

// handleSlashEnterKey handles Enter when input starts with "/".
func (m Model) handleSlashEnterKey(inputVal string) (Model, tea.Cmd, bool) {
	if strings.TrimSpace(inputVal) == "" {
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
		return m, nil, true
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
	return m.execSlashEnterKeyLocal(inputVal)
}

// execSlashEnterKeyLocal runs slash-mode Enter locally after lifecycle submission routing.
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
	m.requestSlashDispatchAction(chosen)
	for _, p := range slashSelectedProviderChain.List() {
		if m2, cmd, handled := p(m, chosen); handled {
			m2.traceSlashEnteredAction(chosen)
			return m2, cmd, true
		}
	}
	return m, nil, false
}
