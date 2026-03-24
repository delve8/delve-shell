package skill

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/git"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/ui"
)

var suggestStyleUpdate = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

func handleUpdateSkillOverlayKey(m ui.Model, key string) (ui.Model, tea.Cmd, bool) {
	if !m.UpdateSkill.Active {
		return m, nil, false
	}
	lang := "en"

	switch key {
	case "up", "down":
		if len(m.UpdateSkill.Refs) == 0 {
			return m, nil, true
		}
		dir := 1
		if key == "up" {
			dir = -1
		}
		m.UpdateSkill.RefIndex = (m.UpdateSkill.RefIndex + dir + len(m.UpdateSkill.Refs)) % len(m.UpdateSkill.Refs)
		// Recompute latest commit for newly selected ref (best-effort; ignore errors).
		selectedRef := strings.TrimSpace(m.UpdateSkill.Refs[m.UpdateSkill.RefIndex])
		url := strings.TrimSpace(m.UpdateSkill.URL)
		if url != "" && selectedRef != "" {
			if commit, err := git.LatestCommit(context.Background(), url, selectedRef); err == nil {
				m.UpdateSkill.LatestCommit = commit
			}
		}
		return m, nil, true
	case "enter":
		if len(m.UpdateSkill.Refs) == 0 || m.UpdateSkill.Name == "" {
			return m, nil, true
		}
		selectedRef := strings.TrimSpace(m.UpdateSkill.Refs[m.UpdateSkill.RefIndex])
		if err := skillsvc.Update(m.UpdateSkill.Name, selectedRef); err != nil {
			m.UpdateSkill.Error = err.Error()
			return m, nil, true
		}
		// On success, close overlay and show a short confirmation message.
		m.Overlay.Active = false
		m.UpdateSkill.Active = false
		m.UpdateSkill.Error = ""
		shortCommit := m.UpdateSkill.LatestCommit
		if len(shortCommit) > 7 {
			shortCommit = shortCommit[:7]
		}
		if shortCommit != "" {
			m.Messages = append(m.Messages, suggestStyleUpdate.Render(delveMsg(lang, fmt.Sprintf(
				"Skill %s updated to %s@%s.",
				m.UpdateSkill.Name,
				selectedRef,
				shortCommit,
			))))
		} else {
			m.Messages = append(m.Messages, suggestStyleUpdate.Render(delveMsg(lang, fmt.Sprintf(
				"Skill %s updated to %s.",
				m.UpdateSkill.Name,
				selectedRef,
			))))
		}
		m.Messages = append(m.Messages, "")
		m = m.RefreshViewport()
		m.Input.Focus()
		if m.Ports.ConfigUpdatedChan != nil {
			select {
			case m.Ports.ConfigUpdatedChan <- struct{}{}:
			default:
			}
		}
		return m, nil, true
	}

	return m, nil, true
}
