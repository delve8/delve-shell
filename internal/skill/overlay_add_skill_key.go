package skill

import (
	"context"
	"errors"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/ui"
)

var suggestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

const addSkillFieldCount = 4

var addSkillPathOptions = []string{
	".",
	"skills",
	"skills/.curated",
	"skills/.experimental",
	".agents/skills",
	".agent/skills",
	".claude/skills",
}

func delveMsg(lang, msg string) string {
	return i18n.T(lang, i18n.KeyDelveLabel) + " " + msg
}

func filterByPrefix(s []string, prefix string) []string {
	if prefix == "" {
		return s
	}
	var out []string
	for _, v := range s {
		if strings.HasPrefix(v, prefix) {
			out = append(out, v)
		}
	}
	return out
}

func runListRefsCmd(url string) tea.Cmd {
	return func() tea.Msg {
		refs := git.ListRefs(context.Background(), url)
		return ui.AddSkillRefsLoadedMsg{Refs: refs}
	}
}

func runListPathsCmd(url, ref string) tea.Cmd {
	return func() tea.Msg {
		paths, _ := git.ListPaths(context.Background(), url, ref)
		return ui.AddSkillPathsLoadedMsg{Paths: paths}
	}
}

func updateAddSkillPathCandidates(m ui.Model) ui.Model {
	source := addSkillPathOptions
	if len(m.AddSkillPathsFullList) > 0 {
		source = m.AddSkillPathsFullList
	}
	m.AddSkillPathCandidates = filterByPrefix(source, m.AddSkillPathInput.Value())
	m.AddSkillPathIndex = 0
	return m
}

// handleAddSkillOverlayKey implements keyboard interactions for the Add-skill overlay.
func handleAddSkillOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	if !m.AddSkillActive {
		return m, nil, false
	}
	lang := "en"
	switch key {
	case "tab":
		if m.AddSkillFieldIndex == 1 && len(m.AddSkillRefCandidates) > 0 && m.AddSkillRefIndex >= 0 && m.AddSkillRefIndex < len(m.AddSkillRefCandidates) {
			m.AddSkillRefInput.SetValue(m.AddSkillRefCandidates[m.AddSkillRefIndex])
			m.AddSkillRefInput.CursorEnd()
			m.AddSkillRefCandidates = nil
			m.AddSkillRefIndex = 0
			return m, nil, true
		}
		if m.AddSkillFieldIndex == 2 && len(m.AddSkillPathCandidates) > 0 && m.AddSkillPathIndex >= 0 && m.AddSkillPathIndex < len(m.AddSkillPathCandidates) {
			m.AddSkillPathInput.SetValue(m.AddSkillPathCandidates[m.AddSkillPathIndex])
			m.AddSkillPathInput.CursorEnd()
			m.AddSkillPathCandidates = nil
			m.AddSkillPathIndex = 0
			return m, nil, true
		}
	case "up", "down":
		dir := 1
		if key == "up" {
			dir = -1
		}
		if m.AddSkillFieldIndex == 1 && len(m.AddSkillRefCandidates) > 0 {
			m.AddSkillRefIndex = (m.AddSkillRefIndex + dir + len(m.AddSkillRefCandidates)) % len(m.AddSkillRefCandidates)
			return m, nil, true
		}
		if m.AddSkillFieldIndex == 2 && len(m.AddSkillPathCandidates) > 0 {
			m.AddSkillPathIndex = (m.AddSkillPathIndex + dir + len(m.AddSkillPathCandidates)) % len(m.AddSkillPathCandidates)
			return m, nil, true
		}
		m.AddSkillFieldIndex = (m.AddSkillFieldIndex + dir + addSkillFieldCount) % addSkillFieldCount
		m.AddSkillURLInput.Blur()
		m.AddSkillRefInput.Blur()
		m.AddSkillPathInput.Blur()
		m.AddSkillNameInput.Blur()
		switch m.AddSkillFieldIndex {
		case 0:
			m.AddSkillURLInput.Focus()
		case 1:
			m.AddSkillRefInput.Focus()
			m.AddSkillRefCandidates = nil
			m.AddSkillRefIndex = 0
			urlForRefs := strings.TrimSpace(m.AddSkillURLInput.Value())
			if urlForRefs != "" {
				return m, runListRefsCmd(urlForRefs), true
			}
		case 2:
			m.AddSkillPathInput.Focus()
			m = updateAddSkillPathCandidates(m)
			urlForPaths := strings.TrimSpace(m.AddSkillURLInput.Value())
			if urlForPaths != "" {
				refForPaths := strings.TrimSpace(m.AddSkillRefInput.Value())
				return m, runListPathsCmd(urlForPaths, refForPaths), true
			}
		case 3:
			m.AddSkillNameInput.Focus()
		}
		return m, nil, true
	case "enter":
		// In Ref field with ref candidates: pick selected and fill
		if m.AddSkillFieldIndex == 1 && len(m.AddSkillRefCandidates) > 0 {
			if m.AddSkillRefIndex >= 0 && m.AddSkillRefIndex < len(m.AddSkillRefCandidates) {
				m.AddSkillRefInput.SetValue(m.AddSkillRefCandidates[m.AddSkillRefIndex])
				m.AddSkillRefInput.CursorEnd()
				m.AddSkillRefCandidates = nil
				m.AddSkillRefIndex = 0
			}
			return m, nil, true
		}
		// In Path field with path candidates: pick selected and fill
		if m.AddSkillFieldIndex == 2 && len(m.AddSkillPathCandidates) > 0 {
			if m.AddSkillPathIndex >= 0 && m.AddSkillPathIndex < len(m.AddSkillPathCandidates) {
				chosenPath := m.AddSkillPathCandidates[m.AddSkillPathIndex]
				m.AddSkillPathInput.SetValue(chosenPath)
				m.AddSkillPathInput.CursorEnd()
				m.AddSkillPathCandidates = nil
				m.AddSkillPathIndex = 0
				// Auto-fill local name from chosen path last segment when name is empty.
				if strings.TrimSpace(m.AddSkillNameInput.Value()) == "" {
					p := strings.TrimSpace(chosenPath)
					if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
						p = p[idx+1:]
					}
					m.AddSkillNameInput.SetValue(p)
					m.AddSkillNameInput.CursorEnd()
				}
			}
			return m, nil, true
		}
		// Submit form
		url := strings.TrimSpace(m.AddSkillURLInput.Value())
		ref := strings.TrimSpace(m.AddSkillRefInput.Value())
		path := strings.TrimSpace(m.AddSkillPathInput.Value())
		nameInput := strings.TrimSpace(m.AddSkillNameInput.Value())
		if path == "." {
			path = ""
		}
		if url == "" {
			m.AddSkillError = i18n.T(lang, i18n.KeyAddSkillURLRequired)
			return m, nil, true
		}
		m.AddSkillError = ""
		name, err := skillsvc.InstallFromGit(url, ref, nameInput, path)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				m.AddSkillError = i18n.T(lang, i18n.KeySkillAlreadyExists)
			} else {
				m.AddSkillError = i18n.Tf(lang, i18n.KeySkillInstallFailed, err)
			}
			return m, nil, true
		}
		m.OverlayActive = false
		m.AddSkillActive = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		m.Input.Focus()
		m.Messages = append(m.Messages, suggestStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillInstalled, name))))
		m = m.RefreshViewport()
		return m, nil, true
	}

	// Default: forward to active field input.
	var cmd tea.Cmd
	switch m.AddSkillFieldIndex {
	case 0:
		m.AddSkillURLInput, cmd = m.AddSkillURLInput.Update(msg)
	case 1:
		m.AddSkillRefInput, cmd = m.AddSkillRefInput.Update(msg)
		m.AddSkillRefCandidates = filterByPrefix(m.AddSkillRefsFullList, m.AddSkillRefInput.Value())
		m.AddSkillRefIndex = 0
	case 2:
		m.AddSkillPathInput, cmd = m.AddSkillPathInput.Update(msg)
		m = updateAddSkillPathCandidates(m)
		// Auto-fill local name from path last segment when name is empty.
		if strings.TrimSpace(m.AddSkillNameInput.Value()) == "" {
			if p := strings.TrimSpace(m.AddSkillPathInput.Value()); p != "" {
				if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
					p = p[idx+1:]
				}
				m.AddSkillNameInput.SetValue(p)
				m.AddSkillNameInput.CursorEnd()
			}
		}
	case 3:
		m.AddSkillNameInput, cmd = m.AddSkillNameInput.Update(msg)
	}
	return m, cmd, true
}
