package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/skills"
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

// openUpdateSkillOverlay initializes the update-skill overlay for the given skill name.
// It loads the skill's source from the manifest, fetches refs and latest commit info,
// and prepares UI state so the user can choose a ref and confirm the update.
func (m Model) openUpdateSkillOverlay(name string) Model {
	lang := m.getLang()
	url, ref, commitID, path, _, ok := skills.GetSkillSource(name)
	if !ok || strings.TrimSpace(url) == "" {
		// Keep this as an overlay (not a transient message) so "Enter" always produces visible feedback.
		m.OverlayActive = true
		m.OverlayTitle = "Update skill"
		m.UpdateSkillActive = true
		m.UpdateSkillName = strings.TrimSpace(name)
		m.UpdateSkillURL = strings.TrimSpace(url)
		m.UpdateSkillPath = strings.TrimSpace(path)
		m.UpdateSkillCurrentCommit = strings.TrimSpace(commitID)
		m.UpdateSkillRefs = nil
		m.UpdateSkillRefIndex = 0
		m.UpdateSkillLatestCommit = ""
		m.UpdateSkillError = i18n.T(lang, i18n.KeySkillNotFound)
		return m
	}
	ctx := context.Background()
	refs := git.ListRefs(ctx, url)
	if len(refs) == 0 {
		// Fallback to using the manifest ref or a sensible default.
		if strings.TrimSpace(ref) != "" {
			refs = []string{ref}
		} else {
			refs = []string{"main", "master"}
		}
	}
	// Determine selected ref and index.
	selectedRef := strings.TrimSpace(ref)
	if selectedRef == "" && len(refs) > 0 {
		selectedRef = refs[0]
	}
	idx := 0
	for i, r := range refs {
		if r == selectedRef {
			idx = i
			break
		}
	}
	latestCommit := ""
	if strings.TrimSpace(selectedRef) != "" {
		if commit, err := git.LatestCommit(ctx, url, selectedRef); err == nil {
			latestCommit = commit
		}
	}

	m.OverlayActive = true
	m.OverlayTitle = "Update skill"
	m.UpdateSkillActive = true
	m.UpdateSkillError = ""
	m.UpdateSkillName = name
	m.UpdateSkillURL = url
	m.UpdateSkillPath = path
	m.UpdateSkillCurrentCommit = commitID
	m.UpdateSkillRefs = refs
	m.UpdateSkillRefIndex = idx
	m.UpdateSkillLatestCommit = latestCommit
	return m
}
