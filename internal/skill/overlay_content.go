package skill

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const skillPicklistFixedRows = 4

func appendSkillPicklistBlock(b *strings.Builder, showTitle bool, items []string, selectedIndex int) {
	if showTitle {
		b.WriteString(ui.RenderOverlayPicklistHintLine())
	} else {
		b.WriteString("\n")
	}
	if len(items) == 0 {
		for i := 0; i < skillPicklistFixedRows; i++ {
			b.WriteString(ui.SuggestStyleRender("  ") + "\n")
		}
		return
	}
	start := 0
	if len(items) > skillPicklistFixedRows {
		if selectedIndex < 0 {
			selectedIndex = 0
		}
		if selectedIndex >= len(items) {
			selectedIndex = len(items) - 1
		}
		start = selectedIndex - skillPicklistFixedRows/2
		if start < 0 {
			start = 0
		}
		if start+skillPicklistFixedRows > len(items) {
			start = len(items) - skillPicklistFixedRows
		}
	}
	win := items[start:]
	if len(win) > skillPicklistFixedRows {
		win = win[:skillPicklistFixedRows]
	}
	for len(win) < skillPicklistFixedRows {
		win = append(win, "")
	}
	for i := 0; i < skillPicklistFixedRows; i++ {
		abs := start + i
		line := "  " + strings.TrimRight(win[i], "\n")
		if win[i] == "" {
			line = "  "
		}
		if abs == selectedIndex && abs < len(items) {
			b.WriteString(ui.SuggestHiRender(line) + "\n")
		} else {
			b.WriteString(ui.SuggestStyleRender(line) + "\n")
		}
	}
}

func buildSkillOverlayContent(m *ui.Model) (string, bool) {
	state := getSkillOverlayState()
	if state.AddSkill.Active {
		var b strings.Builder
		if state.AddSkill.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.AddSkill.Error) + "\n\n")
		}
		b.WriteString(i18n.T(i18n.KeyAddSkillURLLabel) + "\n")
		b.WriteString(state.AddSkill.URLInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(i18n.KeyAddSkillRefLabel) + "\n")
		b.WriteString(state.AddSkill.RefInput.View())
		b.WriteString("\n\n")
		refFocused := state.AddSkill.FieldIndex == 1
		refCandidates := state.AddSkill.RefCandidates
		refIndex := state.AddSkill.RefIndex
		if !refFocused {
			refCandidates = nil
			refIndex = 0
		}
		appendSkillPicklistBlock(&b, refFocused, refCandidates, refIndex)
		b.WriteString("\n")
		b.WriteString(i18n.T(i18n.KeyAddSkillPathLabel) + "\n")
		b.WriteString(state.AddSkill.PathInput.View())
		b.WriteString("\n\n")
		pathFocused := state.AddSkill.FieldIndex == 2
		pathCandidates := state.AddSkill.PathCandidates
		pathIndex := state.AddSkill.PathIndex
		if !pathFocused {
			pathCandidates = nil
			pathIndex = 0
		}
		appendSkillPicklistBlock(&b, pathFocused, pathCandidates, pathIndex)
		b.WriteString("\n")
		b.WriteString(i18n.T(i18n.KeyAddSkillNameLabel) + "\n")
		b.WriteString(state.AddSkill.NameInput.View())
		b.WriteString("\n\n")
		b.WriteString(ui.RenderOverlayFormFooterHint())
		return b.String(), true
	}

	if state.UpdateSkill.Active {
		var b strings.Builder
		if state.UpdateSkill.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.UpdateSkill.Error) + "\n\n")
		}
		b.WriteString(i18n.T(i18n.KeyUpdateSkillTitle) + "\n\n")
		b.WriteString(i18n.Tf(i18n.KeyUpdateSkillSkillLabel, state.UpdateSkill.Name) + "\n")
		b.WriteString(i18n.Tf(i18n.KeyUpdateSkillURLLabel, state.UpdateSkill.URL) + "\n")
		path := state.UpdateSkill.Path
		if strings.TrimSpace(path) == "" {
			path = "."
		}
		b.WriteString(i18n.Tf(i18n.KeyUpdateSkillPathLabel, path) + "\n\n")
		b.WriteString(i18n.T(i18n.KeyOverlayUpdateSkillRefTitle) + "\n")
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
		b.WriteString(i18n.Tf(i18n.KeyUpdateSkillCurrentCommitLabel, current) + "\n")
		b.WriteString(i18n.Tf(i18n.KeyUpdateSkillLatestCommitLabel, latest) + "\n\n")
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEnterUpdateEsc))
		return b.String(), true
	}

	return "", false
}
