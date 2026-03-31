package skill

import (
	"context"
	"errors"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/i18n"
	"delve-shell/internal/skill/git"
	"delve-shell/internal/skill/store"
	"delve-shell/internal/ui"
)

var suggestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

const addSkillFieldCount = 4

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
		return AddRefsLoadedMsg{Refs: refs}
	}
}

func runListPathsCmd(url, ref string) tea.Cmd {
	return func() tea.Msg {
		paths, _ := git.ListPaths(context.Background(), url, ref)
		return AddPathsLoadedMsg{Paths: paths}
	}
}

func updateAddSkillPathCandidates(state skillOverlayState) skillOverlayState {
	var source []string
	if len(state.AddSkill.PathsFullList) > 0 {
		source = state.AddSkill.PathsFullList
	}
	state.AddSkill.PathCandidates = filterByPrefix(source, state.AddSkill.PathInput.Value())
	state.AddSkill.PathIndex = 0
	return state
}

// handleAddSkillOverlayKey implements keyboard interactions for the Add-skill overlay.
func handleAddSkillOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	state := getSkillOverlayState()
	ret := func(model ui.Model, cmd tea.Cmd, handled bool) (ui.Model, tea.Cmd, bool) {
		setSkillOverlayState(state)
		return model, cmd, handled
	}
	if !state.AddSkill.Active {
		return m, nil, false
	}
	lang := "en"
	switch key {
	case "tab":
		if state.AddSkill.FieldIndex == 1 && len(state.AddSkill.RefCandidates) > 0 && state.AddSkill.RefIndex >= 0 && state.AddSkill.RefIndex < len(state.AddSkill.RefCandidates) {
			state.AddSkill.RefInput.SetValue(state.AddSkill.RefCandidates[state.AddSkill.RefIndex])
			state.AddSkill.RefInput.CursorEnd()
			state.AddSkill.RefCandidates = nil
			state.AddSkill.RefIndex = 0
			return ret(m, nil, true)
		}
		if state.AddSkill.FieldIndex == 2 && len(state.AddSkill.PathCandidates) > 0 && state.AddSkill.PathIndex >= 0 && state.AddSkill.PathIndex < len(state.AddSkill.PathCandidates) {
			state.AddSkill.PathInput.SetValue(state.AddSkill.PathCandidates[state.AddSkill.PathIndex])
			state.AddSkill.PathInput.CursorEnd()
			state.AddSkill.PathCandidates = nil
			state.AddSkill.PathIndex = 0
			return ret(m, nil, true)
		}
	case "up", "down":
		dir := 1
		if key == "up" {
			dir = -1
		}
		if state.AddSkill.FieldIndex == 1 && len(state.AddSkill.RefCandidates) > 0 {
			state.AddSkill.RefIndex = (state.AddSkill.RefIndex + dir + len(state.AddSkill.RefCandidates)) % len(state.AddSkill.RefCandidates)
			return ret(m, nil, true)
		}
		if state.AddSkill.FieldIndex == 2 && len(state.AddSkill.PathCandidates) > 0 {
			state.AddSkill.PathIndex = (state.AddSkill.PathIndex + dir + len(state.AddSkill.PathCandidates)) % len(state.AddSkill.PathCandidates)
			return ret(m, nil, true)
		}
		state.AddSkill.FieldIndex = (state.AddSkill.FieldIndex + dir + addSkillFieldCount) % addSkillFieldCount
		state.AddSkill.URLInput.Blur()
		state.AddSkill.RefInput.Blur()
		state.AddSkill.PathInput.Blur()
		state.AddSkill.NameInput.Blur()
		switch state.AddSkill.FieldIndex {
		case 0:
			state.AddSkill.URLInput.Focus()
		case 1:
			state.AddSkill.RefInput.Focus()
			state.AddSkill.RefCandidates = nil
			state.AddSkill.RefIndex = 0
			urlForRefs := strings.TrimSpace(state.AddSkill.URLInput.Value())
			if urlForRefs != "" {
				return ret(m, runListRefsCmd(urlForRefs), true)
			}
		case 2:
			state.AddSkill.PathInput.Focus()
			state = updateAddSkillPathCandidates(state)
			urlForPaths := strings.TrimSpace(state.AddSkill.URLInput.Value())
			if urlForPaths != "" {
				refForPaths := strings.TrimSpace(state.AddSkill.RefInput.Value())
				return ret(m, runListPathsCmd(urlForPaths, refForPaths), true)
			}
		case 3:
			state.AddSkill.NameInput.Focus()
		}
		return ret(m, nil, true)
	case "enter":
		// In Ref field with ref candidates: pick selected and fill
		if state.AddSkill.FieldIndex == 1 && len(state.AddSkill.RefCandidates) > 0 {
			if state.AddSkill.RefIndex >= 0 && state.AddSkill.RefIndex < len(state.AddSkill.RefCandidates) {
				state.AddSkill.RefInput.SetValue(state.AddSkill.RefCandidates[state.AddSkill.RefIndex])
				state.AddSkill.RefInput.CursorEnd()
				state.AddSkill.RefCandidates = nil
				state.AddSkill.RefIndex = 0
			}
			return ret(m, nil, true)
		}
		// In Path field with path candidates: pick selected and fill
		if state.AddSkill.FieldIndex == 2 && len(state.AddSkill.PathCandidates) > 0 {
			if state.AddSkill.PathIndex >= 0 && state.AddSkill.PathIndex < len(state.AddSkill.PathCandidates) {
				chosenPath := state.AddSkill.PathCandidates[state.AddSkill.PathIndex]
				state.AddSkill.PathInput.SetValue(chosenPath)
				state.AddSkill.PathInput.CursorEnd()
				state.AddSkill.PathCandidates = nil
				state.AddSkill.PathIndex = 0
				// Auto-fill local name from chosen path last segment when name is empty.
				if strings.TrimSpace(state.AddSkill.NameInput.Value()) == "" {
					p := strings.TrimSpace(chosenPath)
					if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
						p = p[idx+1:]
					}
					state.AddSkill.NameInput.SetValue(p)
					state.AddSkill.NameInput.CursorEnd()
				}
			}
			return ret(m, nil, true)
		}
		// Submit form
		url := strings.TrimSpace(state.AddSkill.URLInput.Value())
		ref := strings.TrimSpace(state.AddSkill.RefInput.Value())
		path := strings.TrimSpace(state.AddSkill.PathInput.Value())
		nameInput := strings.TrimSpace(state.AddSkill.NameInput.Value())
		if path == "." {
			path = ""
		}
		if url == "" {
			state.AddSkill.Error = i18n.T(lang, i18n.KeyAddSkillURLRequired)
			return ret(m, nil, true)
		}
		state.AddSkill.Error = ""
		name, err := skillstore.InstallFromGit(url, ref, nameInput, path)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				state.AddSkill.Error = i18n.T(lang, i18n.KeySkillAlreadyExists)
			} else {
				state.AddSkill.Error = i18n.Tf(lang, i18n.KeySkillInstallFailed, err)
			}
			return ret(m, nil, true)
		}
		m = m.CloseOverlayVisual()
		state.AddSkill.Active = false
		m.Input.Focus()
		m = m.AppendTranscriptLines(suggestStyle.Render(delveMsg(lang, i18n.Tf(lang, i18n.KeySkillInstalled, name))))
		return ret(m, nil, true)
	}

	// Default: forward to active field input.
	var cmd tea.Cmd
	switch state.AddSkill.FieldIndex {
	case 0:
		state.AddSkill.URLInput, cmd = state.AddSkill.URLInput.Update(msg)
	case 1:
		state.AddSkill.RefInput, cmd = state.AddSkill.RefInput.Update(msg)
		state.AddSkill.RefCandidates = filterByPrefix(state.AddSkill.RefsFullList, state.AddSkill.RefInput.Value())
		state.AddSkill.RefIndex = 0
	case 2:
		state.AddSkill.PathInput, cmd = state.AddSkill.PathInput.Update(msg)
		state = updateAddSkillPathCandidates(state)
		// Auto-fill local name from path last segment when name is empty.
		if strings.TrimSpace(state.AddSkill.NameInput.Value()) == "" {
			if p := strings.TrimSpace(state.AddSkill.PathInput.Value()); p != "" {
				if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
					p = p[idx+1:]
				}
				state.AddSkill.NameInput.SetValue(p)
				state.AddSkill.NameInput.CursorEnd()
			}
		}
	case 3:
		state.AddSkill.NameInput, cmd = state.AddSkill.NameInput.Update(msg)
	}
	return ret(m, cmd, true)
}
