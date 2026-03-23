package ui

import (
	"context"
	"strings"

	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/skills"
)

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
