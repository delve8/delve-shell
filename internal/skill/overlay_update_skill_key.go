package skill

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/skill/git"
	"delve-shell/internal/skill/store"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui"
)

var suggestStyleUpdate = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

func handleUpdateSkillOverlayKey(m *ui.Model, key string) (*ui.Model, tea.Cmd, bool) {
	state := getSkillOverlayState()
	ret := func(model *ui.Model, cmd tea.Cmd, handled bool) (*ui.Model, tea.Cmd, bool) {
		setSkillOverlayState(state)
		return model, cmd, handled
	}
	if !state.UpdateSkill.Active {
		return m, nil, false
	}
	switch key {
	case teakey.Up, teakey.Down:
		if len(state.UpdateSkill.Refs) == 0 {
			return ret(m, nil, true)
		}
		dir := 1
		if key == teakey.Up {
			dir = -1
		}
		state.UpdateSkill.RefIndex = (state.UpdateSkill.RefIndex + dir + len(state.UpdateSkill.Refs)) % len(state.UpdateSkill.Refs)
		// Recompute latest commit for newly selected ref (best-effort; ignore errors).
		selectedRef := strings.TrimSpace(state.UpdateSkill.Refs[state.UpdateSkill.RefIndex])
		url := strings.TrimSpace(state.UpdateSkill.URL)
		if url != "" && selectedRef != "" {
			if commit, err := git.LatestCommit(context.Background(), url, selectedRef); err == nil {
				state.UpdateSkill.LatestCommit = commit
			}
		}
		return ret(m, nil, true)
	case teakey.Enter:
		if len(state.UpdateSkill.Refs) == 0 || state.UpdateSkill.Name == "" {
			return ret(m, nil, true)
		}
		selectedRef := strings.TrimSpace(state.UpdateSkill.Refs[state.UpdateSkill.RefIndex])
		if err := skillstore.Update(state.UpdateSkill.Name, selectedRef); err != nil {
			state.UpdateSkill.Error = err.Error()
			return ret(m, nil, true)
		}
		// On success, close overlay and show a short confirmation message.
		m.CloseOverlayVisual()
		state.UpdateSkill.Active = false
		state.UpdateSkill.Error = ""
		shortCommit := state.UpdateSkill.LatestCommit
		if len(shortCommit) > 7 {
			shortCommit = shortCommit[:7]
		}
		if shortCommit != "" {
			m.AppendTranscriptLines(suggestStyleUpdate.Render(delveMsg(fmt.Sprintf(
				"Skill %s updated to %s@%s.",
				state.UpdateSkill.Name,
				selectedRef,
				shortCommit,
			))))
		} else {
			m.AppendTranscriptLines(suggestStyleUpdate.Render(delveMsg(fmt.Sprintf(
				"Skill %s updated to %s.",
				state.UpdateSkill.Name,
				selectedRef,
			))))
		}
		m.AppendTranscriptLines("")
		m.Input.Focus()
		if m.CommandSender != nil {
			_ = m.CommandSender.Send(hostcmd.ConfigUpdated{})
		}
		return ret(m, nil, true)
	}

	return ret(m, nil, true)
}
