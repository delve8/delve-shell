package ui

import (
	"errors"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/i18n"
)

func (m Model) handleAddSkillOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	// Add-skill overlay: URL, ref, path, name.
	if !m.AddSkillActive {
		return m, nil, false
	}
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
				return m, RunListRefsCmd(urlForRefs), true
			}
		case 2:
			m.AddSkillPathInput.Focus()
			m = m.updateAddSkillPathCandidates()
			urlForPaths := strings.TrimSpace(m.AddSkillURLInput.Value())
			if urlForPaths != "" {
				refForPaths := strings.TrimSpace(m.AddSkillRefInput.Value())
				return m, RunListPathsCmd(urlForPaths, refForPaths), true
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
			m.AddSkillError = i18n.T(m.getLang(), i18n.KeyAddSkillURLRequired)
			return m, nil, true
		}
		m.AddSkillError = ""
		name, err := skillsvc.InstallFromGit(url, ref, nameInput, path)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				m.AddSkillError = i18n.T(m.getLang(), i18n.KeySkillAlreadyExists)
			} else {
				m.AddSkillError = i18n.Tf(m.getLang(), i18n.KeySkillInstallFailed, err)
			}
			return m, nil, true
		}
		m.OverlayActive = false
		m.AddSkillActive = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		m.Input.Focus()
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillInstalled, name))))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
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
		m = m.updateAddSkillPathCandidates()
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

