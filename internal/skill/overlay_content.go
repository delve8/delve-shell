package skill

import (
	"fmt"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func buildSkillOverlayContent(m ui.Model) (string, bool) {
	state := getSkillOverlayState()
	if state.AddSkill.Active {
		lang := m.GetLang()
		var b strings.Builder
		if state.AddSkill.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.AddSkill.Error) + "\n\n")
		}
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillURLLabel) + "\n")
		b.WriteString(state.AddSkill.URLInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillRefLabel) + "\n")
		b.WriteString(state.AddSkill.RefInput.View())
		if state.AddSkill.FieldIndex == 1 && len(state.AddSkill.RefCandidates) > 0 {
			b.WriteString("\n")
			b.WriteString(ui.RenderOverlayPicklistHintLine(lang))
			for i, c := range state.AddSkill.RefCandidates {
				line := "  " + c
				if i == state.AddSkill.RefIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillPathLabel) + "\n")
		b.WriteString(state.AddSkill.PathInput.View())
		if state.AddSkill.FieldIndex == 2 && len(state.AddSkill.PathCandidates) > 0 {
			b.WriteString("\n")
			b.WriteString(ui.RenderOverlayPicklistHintLine(lang))
			for i, c := range state.AddSkill.PathCandidates {
				line := "  " + c
				if i == state.AddSkill.PathIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillNameLabel) + "\n")
		b.WriteString(state.AddSkill.NameInput.View())
		b.WriteString("\n\n")
		b.WriteString(ui.RenderOverlayFormFooterHint(lang))
		return b.String(), true
	}

	if state.UpdateSkill.Active {
		lang := m.GetLang()
		var b strings.Builder
		if state.UpdateSkill.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.UpdateSkill.Error) + "\n\n")
		}
		b.WriteString("Update skill\n\n")
		b.WriteString("Skill: " + state.UpdateSkill.Name + "\n")
		b.WriteString("URL:   " + state.UpdateSkill.URL + "\n")
		path := state.UpdateSkill.Path
		if strings.TrimSpace(path) == "" {
			path = "."
		}
		b.WriteString("Path:  " + path + "\n\n")
		b.WriteString(ui.RenderOverlayUpdateSkillRefTitleLine(lang))
		for i, r := range state.UpdateSkill.Refs {
			line := "  " + r
			if i == state.UpdateSkill.RefIndex {
				b.WriteString(ui.SuggestHiRender(line) + "\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(line) + "\n")
			}
		}
		b.WriteString("\n")
		current := strings.TrimSpace(state.UpdateSkill.CurrentCommit)
		if current == "" {
			current = "(unknown)"
		} else if len(current) > 7 {
			current = current[:7]
		}
		latest := strings.TrimSpace(state.UpdateSkill.LatestCommit)
		if latest == "" {
			latest = "(unknown)"
		} else if len(latest) > 7 {
			latest = latest[:7]
		}
		b.WriteString(fmt.Sprintf("Current commit: %s\n", current))
		b.WriteString(fmt.Sprintf("Latest commit:  %s\n\n", latest))
		b.WriteString(i18n.T(lang, i18n.KeyDescConfigUpdateSkill))
		return b.String(), true
	}

	return "", false
}
