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
var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

const addSkillFieldCount = 4

var addSkillPathOptions = []string{
	".",
	"skills",
	"skills/.curated",
	"skills/.experimental",
	"skills/.system",
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
	if len(m.AddSkill.PathsFullList) > 0 {
		source = m.AddSkill.PathsFullList
	}
	m.AddSkill.PathCandidates = filterByPrefix(source, m.AddSkill.PathInput.Value())
	m.AddSkill.PathIndex = 0
	return m
}

// handleAddSkillOverlayKey implements keyboard interactions for the Add-skill overlay.
func handleAddSkillOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	if !m.AddSkill.Active {
		return m, nil, false
	}
	lang := "en"
	switch key {
	case "tab":
		if m.AddSkill.FieldIndex == 1 && len(m.AddSkill.RefCandidates) > 0 && m.AddSkill.RefIndex >= 0 && m.AddSkill.RefIndex < len(m.AddSkill.RefCandidates) {
			m.AddSkill.RefInput.SetValue(m.AddSkill.RefCandidates[m.AddSkill.RefIndex])
			m.AddSkill.RefInput.CursorEnd()
			m.AddSkill.RefCandidates = nil
			m.AddSkill.RefIndex = 0
			return m, nil, true
		}
		if m.AddSkill.FieldIndex == 2 && len(m.AddSkill.PathCandidates) > 0 && m.AddSkill.PathIndex >= 0 && m.AddSkill.PathIndex < len(m.AddSkill.PathCandidates) {
			m.AddSkill.PathInput.SetValue(m.AddSkill.PathCandidates[m.AddSkill.PathIndex])
			m.AddSkill.PathInput.CursorEnd()
			m.AddSkill.PathCandidates = nil
			m.AddSkill.PathIndex = 0
			return m, nil, true
		}
	case "up", "down":
		dir := 1
		if key == "up" {
			dir = -1
		}
		if m.AddSkill.FieldIndex == 1 && len(m.AddSkill.RefCandidates) > 0 {
			m.AddSkill.RefIndex = (m.AddSkill.RefIndex + dir + len(m.AddSkill.RefCandidates)) % len(m.AddSkill.RefCandidates)
			return m, nil, true
		}
		if m.AddSkill.FieldIndex == 2 && len(m.AddSkill.PathCandidates) > 0 {
			m.AddSkill.PathIndex = (m.AddSkill.PathIndex + dir + len(m.AddSkill.PathCandidates)) % len(m.AddSkill.PathCandidates)
			return m, nil, true
		}
		m.AddSkill.FieldIndex = (m.AddSkill.FieldIndex + dir + addSkillFieldCount) % addSkillFieldCount
		m.AddSkill.URLInput.Blur()
		m.AddSkill.RefInput.Blur()
		m.AddSkill.PathInput.Blur()
		m.AddSkill.NameInput.Blur()
		switch m.AddSkill.FieldIndex {
		case 0:
			m.AddSkill.URLInput.Focus()
		case 1:
			m.AddSkill.RefInput.Focus()
			m.AddSkill.RefCandidates = nil
			m.AddSkill.RefIndex = 0
			urlForRefs := strings.TrimSpace(m.AddSkill.URLInput.Value())
			if urlForRefs != "" {
				return m, runListRefsCmd(urlForRefs), true
			}
		case 2:
			m.AddSkill.PathInput.Focus()
			m = updateAddSkillPathCandidates(m)
			urlForPaths := strings.TrimSpace(m.AddSkill.URLInput.Value())
			if urlForPaths != "" {
				refForPaths := strings.TrimSpace(m.AddSkill.RefInput.Value())
				return m, runListPathsCmd(urlForPaths, refForPaths), true
			}
		case 3:
			m.AddSkill.NameInput.Focus()
		}
		return m, nil, true
	case "enter":
		// In Ref field with ref candidates: pick selected and fill
		if m.AddSkill.FieldIndex == 1 && len(m.AddSkill.RefCandidates) > 0 {
			if m.AddSkill.RefIndex >= 0 && m.AddSkill.RefIndex < len(m.AddSkill.RefCandidates) {
				m.AddSkill.RefInput.SetValue(m.AddSkill.RefCandidates[m.AddSkill.RefIndex])
				m.AddSkill.RefInput.CursorEnd()
				m.AddSkill.RefCandidates = nil
				m.AddSkill.RefIndex = 0
			}
			return m, nil, true
		}
		// In Path field with path candidates: pick selected and fill
		if m.AddSkill.FieldIndex == 2 && len(m.AddSkill.PathCandidates) > 0 {
			if m.AddSkill.PathIndex >= 0 && m.AddSkill.PathIndex < len(m.AddSkill.PathCandidates) {
				chosenPath := m.AddSkill.PathCandidates[m.AddSkill.PathIndex]
				m.AddSkill.PathInput.SetValue(chosenPath)
				m.AddSkill.PathInput.CursorEnd()
				m.AddSkill.PathCandidates = nil
				m.AddSkill.PathIndex = 0
				// Auto-fill local name from chosen path last segment when name is empty.
				if strings.TrimSpace(m.AddSkill.NameInput.Value()) == "" {
					p := strings.TrimSpace(chosenPath)
					if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
						p = p[idx+1:]
					}
					m.AddSkill.NameInput.SetValue(p)
					m.AddSkill.NameInput.CursorEnd()
				}
			}
			return m, nil, true
		}
		// Submit form
		url := strings.TrimSpace(m.AddSkill.URLInput.Value())
		ref := strings.TrimSpace(m.AddSkill.RefInput.Value())
		path := strings.TrimSpace(m.AddSkill.PathInput.Value())
		nameInput := strings.TrimSpace(m.AddSkill.NameInput.Value())
		if path == "." {
			path = ""
		}
		if url == "" {
			m.AddSkill.Error = i18n.T(lang, i18n.KeyAddSkillURLRequired)
			return m, nil, true
		}
		m.AddSkill.Error = ""
		name, err := skillsvc.InstallFromGit(url, ref, nameInput, path)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				m.AddSkill.Error = i18n.T(lang, i18n.KeySkillAlreadyExists)
			} else {
				m.AddSkill.Error = i18n.Tf(lang, i18n.KeySkillInstallFailed, err)
			}
			return m, nil, true
		}
		m.Overlay.Active = false
		m.AddSkill.Active = false
		m.Overlay.Title = ""
		m.Overlay.Content = ""
		m.Input.Focus()
		m.Messages = append(m.Messages, suggestStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillInstalled, name))))
		m = m.RefreshViewport()
		return m, nil, true
	}

	// Default: forward to active field input.
	var cmd tea.Cmd
	switch m.AddSkill.FieldIndex {
	case 0:
		m.AddSkill.URLInput, cmd = m.AddSkill.URLInput.Update(msg)
	case 1:
		m.AddSkill.RefInput, cmd = m.AddSkill.RefInput.Update(msg)
		m.AddSkill.RefCandidates = filterByPrefix(m.AddSkill.RefsFullList, m.AddSkill.RefInput.Value())
		m.AddSkill.RefIndex = 0
	case 2:
		m.AddSkill.PathInput, cmd = m.AddSkill.PathInput.Update(msg)
		m = updateAddSkillPathCandidates(m)
		// Auto-fill local name from path last segment when name is empty.
		if strings.TrimSpace(m.AddSkill.NameInput.Value()) == "" {
			if p := strings.TrimSpace(m.AddSkill.PathInput.Value()); p != "" {
				if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
					p = p[idx+1:]
				}
				m.AddSkill.NameInput.SetValue(p)
				m.AddSkill.NameInput.CursorEnd()
			}
		}
	case 3:
		m.AddSkill.NameInput, cmd = m.AddSkill.NameInput.Update(msg)
	}
	return m, cmd, true
}
