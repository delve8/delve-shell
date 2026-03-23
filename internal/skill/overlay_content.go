package skill

import (
	"fmt"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func buildSkillOverlayContent(m ui.Model) (string, bool) {
	if m.AddSkillActive {
		lang := "en"
		var b strings.Builder
		if m.AddSkillError != "" {
			b.WriteString(ui.ErrStyleRender(m.AddSkillError) + "\n\n")
		}
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillURLLabel) + "\n")
		b.WriteString(m.AddSkillURLInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillRefLabel) + "\n")
		b.WriteString(m.AddSkillRefInput.View())
		if m.AddSkillFieldIndex == 1 && len(m.AddSkillRefCandidates) > 0 {
			b.WriteString("\n")
			b.WriteString("  (Up/Down select, Enter or Tab to pick)\n")
			for i, c := range m.AddSkillRefCandidates {
				line := "  " + c
				if i == m.AddSkillRefIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillPathLabel) + "\n")
		b.WriteString(m.AddSkillPathInput.View())
		if m.AddSkillFieldIndex == 2 && len(m.AddSkillPathCandidates) > 0 {
			b.WriteString("\n")
			b.WriteString("  (Up/Down select, Enter or Tab to pick)\n")
			for i, c := range m.AddSkillPathCandidates {
				line := "  " + c
				if i == m.AddSkillPathIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillNameLabel) + "\n")
		b.WriteString(m.AddSkillNameInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyAddSkillHint))
		return b.String(), true
	}

	if m.UpdateSkillActive {
		lang := "en"
		var b strings.Builder
		if m.UpdateSkillError != "" {
			b.WriteString(ui.ErrStyleRender(m.UpdateSkillError) + "\n\n")
		}
		b.WriteString("Update skill\n\n")
		b.WriteString("Skill: " + m.UpdateSkillName + "\n")
		b.WriteString("URL:   " + m.UpdateSkillURL + "\n")
		path := m.UpdateSkillPath
		if strings.TrimSpace(path) == "" {
			path = "."
		}
		b.WriteString("Path:  " + path + "\n\n")
		b.WriteString("Ref (Up/Down to change, Enter to update, Esc to cancel):\n")
		for i, r := range m.UpdateSkillRefs {
			line := "  " + r
			if i == m.UpdateSkillRefIndex {
				b.WriteString(ui.SuggestHiRender(line) + "\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(line) + "\n")
			}
		}
		b.WriteString("\n")
		current := strings.TrimSpace(m.UpdateSkillCurrentCommit)
		if current == "" {
			current = "(unknown)"
		} else if len(current) > 7 {
			current = current[:7]
		}
		latest := strings.TrimSpace(m.UpdateSkillLatestCommit)
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
