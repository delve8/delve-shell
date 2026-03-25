package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"
)

// keySession narrows keyboard handling for [Model.handleKeyMsg] so update_keymsg does not
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
