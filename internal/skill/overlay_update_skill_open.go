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
		m = m.OpenOverlay("Update skill", "")
		m.UpdateSkill.Active = true
		m.UpdateSkill.Name = strings.TrimSpace(name)
		m.UpdateSkill.URL = strings.TrimSpace(url)
		m.UpdateSkill.Path = strings.TrimSpace(path)
		m.UpdateSkill.CurrentCommit = strings.TrimSpace(commitID)
		m.UpdateSkill.Refs = nil
		m.UpdateSkill.RefIndex = 0
		m.UpdateSkill.LatestCommit = ""
		m.UpdateSkill.Error = i18n.T(lang, i18n.KeySkillNotFound)
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

	m = m.OpenOverlay("Update skill", "")
	m.UpdateSkill.Active = true
	m.UpdateSkill.Error = ""
	m.UpdateSkill.Name = name
	m.UpdateSkill.URL = url
	m.UpdateSkill.Path = path
	m.UpdateSkill.CurrentCommit = commitID
	m.UpdateSkill.Refs = refs
	m.UpdateSkill.RefIndex = idx
	m.UpdateSkill.LatestCommit = latestCommit
	return m
}
