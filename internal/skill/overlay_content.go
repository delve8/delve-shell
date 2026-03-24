package skill

import (
	"fmt"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func buildSkillOverlayContent(m ui.Model) (string, bool) {
	if m.AddSkill.Active {
		lang := "en"
		var b strings.Builder
		if m.AddSkill.Error != "" {
			b.WriteString(ui.ErrStyleRender(m.AddSkill.Error) + "\n\n")
		}
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillURLLabel) + "\n")
		b.WriteString(m.AddSkill.URLInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillRefLabel) + "\n")
		b.WriteString(m.AddSkill.RefInput.View())
		if m.AddSkill.FieldIndex == 1 && len(m.AddSkill.RefCandidates) > 0 {
			b.WriteString("\n")
			b.WriteString("  (Up/Down select, Enter or Tab to pick)\n")
			for i, c := range m.AddSkill.RefCandidates {
				line := "  " + c
				if i == m.AddSkill.RefIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillPathLabel) + "\n")
		b.WriteString(m.AddSkill.PathInput.View())
		if m.AddSkill.FieldIndex == 2 && len(m.AddSkill.PathCandidates) > 0 {
			b.WriteString("\n")
			b.WriteString("  (Up/Down select, Enter or Tab to pick)\n")
			for i, c := range m.AddSkill.PathCandidates {
				line := "  " + c
				if i == m.AddSkill.PathIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillNameLabel) + "\n")
		b.WriteString(m.AddSkill.NameInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillHint))
		return b.String(), true
	}

	if m.UpdateSkill.Active {
		lang := "en"
		var b strings.Builder
		if m.UpdateSkill.Error != "" {
			b.WriteString(ui.ErrStyleRender(m.UpdateSkill.Error) + "\n\n")
		}
		b.WriteString("Update skill\n\n")
		b.WriteString("Skill: " + m.UpdateSkill.Name + "\n")
		b.WriteString("URL:   " + m.UpdateSkill.URL + "\n")
		path := m.UpdateSkill.Path
		if strings.TrimSpace(path) == "" {
			path = "."
		}
		b.WriteString("Path:  " + path + "\n\n")
		b.WriteString("Ref (Up/Down to change, Enter to update, Esc to cancel):\n")
		for i, r := range m.UpdateSkill.Refs {
			line := "  " + r
			if i == m.UpdateSkill.RefIndex {
				b.WriteString(ui.SuggestHiRender(line) + "\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(line) + "\n")
			}
		}
		b.WriteString("\n")
		current := strings.TrimSpace(m.UpdateSkill.CurrentCommit)
		if current == "" {
			current = "(unknown)"
		} else if len(current) > 7 {
			current = current[:7]
		}
		latest := strings.TrimSpace(m.UpdateSkill.LatestCommit)
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
