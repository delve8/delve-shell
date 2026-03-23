package skill

import (
	"context"
	"strings"

	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/skills"
	"delve-shell/internal/ui"
)

// openUpdateSkillOverlay initializes update-skill overlay state.
func openUpdateSkillOverlay(m ui.Model, name string) ui.Model {
	lang := "en"
	url, ref, commitID, path, _, ok := skills.GetSkillSource(name)
	if !ok || strings.TrimSpace(url) == "" {
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
		if strings.TrimSpace(ref) != "" {
			refs = []string{ref}
		} else {
			refs = []string{"main", "master"}
		}
	}

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
