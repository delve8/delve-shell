package skill

import (
	"context"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/skill/git"
	"delve-shell/internal/skill/store"
	"delve-shell/internal/ui"
)

// openUpdateSkillOverlay initializes update-skill overlay state.
func openUpdateSkillOverlay(m ui.Model, name string) ui.Model {
	url, ref, commitID, path, _, ok := skillstore.GetSkillSource(name)
	state := getSkillOverlayState()
	if !ok || strings.TrimSpace(url) == "" {
		m = m.OpenOverlayFeature(OverlayFeatureKey, "Update skill", "")
		state.UpdateSkill.Active = true
		state.AddSkill = AddSkillOverlayState{}
		state.UpdateSkill.Name = strings.TrimSpace(name)
		state.UpdateSkill.URL = strings.TrimSpace(url)
		state.UpdateSkill.Path = strings.TrimSpace(path)
		state.UpdateSkill.CurrentCommit = strings.TrimSpace(commitID)
		state.UpdateSkill.Refs = nil
		state.UpdateSkill.RefIndex = 0
		state.UpdateSkill.LatestCommit = ""
		state.UpdateSkill.Error = i18n.T(i18n.KeySkillNotFound)
		setSkillOverlayState(state)
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

	m = m.OpenOverlayFeature(OverlayFeatureKey, "Update skill", "")
	state.UpdateSkill.Active = true
	state.AddSkill = AddSkillOverlayState{}
	state.UpdateSkill.Error = ""
	state.UpdateSkill.Name = name
	state.UpdateSkill.URL = url
	state.UpdateSkill.Path = path
	state.UpdateSkill.CurrentCommit = commitID
	state.UpdateSkill.Refs = refs
	state.UpdateSkill.RefIndex = idx
	state.UpdateSkill.LatestCommit = latestCommit
	setSkillOverlayState(state)
	return m
}
