package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/app/service/skillsvc"
	"delve-shell/internal/git"
)

func (m Model) handleUpdateSkillOverlayKey(key string) (Model, tea.Cmd, bool) {
	if !m.UpdateSkillActive {
		return m, nil, false
	}
	switch key {
	case "up", "down":
		if len(m.UpdateSkillRefs) == 0 {
			return m, nil, true
		}
		dir := 1
		if key == "up" {
			dir = -1
		}
		m.UpdateSkillRefIndex = (m.UpdateSkillRefIndex + dir + len(m.UpdateSkillRefs)) % len(m.UpdateSkillRefs)
		// Recompute latest commit for newly selected ref (best-effort; ignore errors).
		selectedRef := strings.TrimSpace(m.UpdateSkillRefs[m.UpdateSkillRefIndex])
		url := strings.TrimSpace(m.UpdateSkillURL)
		if url != "" && selectedRef != "" {
			if commit, err := git.LatestCommit(context.Background(), url, selectedRef); err == nil {
				m.UpdateSkillLatestCommit = commit
			}
		}
		return m, nil, true
	case "enter":
		if len(m.UpdateSkillRefs) == 0 || m.UpdateSkillName == "" {
			return m, nil, true
		}
		selectedRef := strings.TrimSpace(m.UpdateSkillRefs[m.UpdateSkillRefIndex])
		if err := skillsvc.Update(m.UpdateSkillName, selectedRef); err != nil {
			m.UpdateSkillError = err.Error()
			return m, nil, true
		}
		// On success, close overlay and show a short confirmation message.
		m.OverlayActive = false
		m.UpdateSkillActive = false
		m.UpdateSkillError = ""
		shortCommit := m.UpdateSkillLatestCommit
		if len(shortCommit) > 7 {
			shortCommit = shortCommit[:7]
		}
		if shortCommit != "" {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(
				fmt.Sprintf("Skill %s updated to %s@%s.", m.UpdateSkillName, selectedRef, shortCommit),
			)))
		} else {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(
				fmt.Sprintf("Skill %s updated to %s.", m.UpdateSkillName, selectedRef),
			)))
		}
		m.Messages = append(m.Messages, "")
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		m.Input.Focus()
		if m.ConfigUpdatedChan != nil {
			select {
			case m.ConfigUpdatedChan <- struct{}{}:
			default:
			}
		}
		return m, nil, true
	}
	return m, nil, true
}

