package skill

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/skills"
	"delve-shell/internal/ui"
)

// Register wires skill-related slash commands, overlays, and message providers into the UI. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	registerOverlayFeature()

	ui.RegisterSlashSelectedProvider(func(m ui.Model, chosen string) (ui.Model, tea.Cmd, bool) {
		if !strings.HasPrefix(chosen, "/skill ") {
			return m, nil, false
		}
		m.Input.SetValue(chosen + " ")
		m.Input.CursorEnd()
		m = m.ResetSlashSuggestIndex()
		return m, nil, true
	})

	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)

		if normalizedLower == "skill" || strings.HasPrefix(normalizedLower, "skill ") {
			filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "skill"))
			return getSkillSlashOptions(lang, filter), true
		}

		if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
			rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
			if rest == "del-skill" || strings.HasPrefix(rest, "del-skill ") {
				filter := strings.TrimSpace(strings.TrimPrefix(rest, "del-skill"))
				return getDelSkillSlashOptions(lang, filter), true
			}
			if rest == "update-skill" || strings.HasPrefix(rest, "update-skill ") {
				filter := strings.TrimSpace(strings.TrimPrefix(rest, "update-skill"))
				return getUpdateSkillSlashOptions(lang, filter), true
			}
		}

		return nil, false
	})

}

func getDelSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	list, err := skills.List()
	if err != nil || len(list) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeySkillNone)}}
	}
	filterLower := strings.ToLower(filter)
	var opts []ui.SlashOption
	for _, s := range list {
		if filterLower != "" && !strings.Contains(strings.ToLower(s.Name), filterLower) {
			continue
		}
		desc := strings.TrimSpace(s.Description)
		if desc == "" {
			desc = s.Name
		}
		cmdName := s.LocalName
		if cmdName == "" {
			cmdName = s.Name
		}
		opts = append(opts, ui.SlashOption{Cmd: "/config del-skill " + cmdName, Desc: desc})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeySkillNone)}}
	}
	return opts
}

func getUpdateSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	sources, err := skills.ListSources()
	if err != nil || len(sources) == 0 {
		return []ui.SlashOption{{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeySkillNone)}}
	}

	filterLower := strings.ToLower(filter)
	ctx := context.Background()
	var opts []ui.SlashOption
	for name, src := range sources {
		if filterLower != "" && !strings.Contains(strings.ToLower(name), filterLower) {
			continue
		}
		labelName := name
		descParts := make([]string, 0, 3)
		if src.Ref != "" {
			descParts = append(descParts, fmt.Sprintf("ref: %s", src.Ref))
		}
		if strings.TrimSpace(src.Path) != "" {
			descParts = append(descParts, fmt.Sprintf("path: %s", src.Path))
		}
		latest := ""
		if strings.TrimSpace(src.URL) != "" {
			if commit, e := git.LatestCommit(ctx, src.URL, src.Ref); e == nil && commit != "" {
				latest = commit
				if src.CommitID != "" && src.CommitID != commit {
					labelName = labelName + "*"
				}
			}
		}
		if src.CommitID != "" {
			short := src.CommitID
			if len(short) > 7 {
				short = short[:7]
			}
			descParts = append(descParts, fmt.Sprintf("current: %s", short))
		}
		if latest != "" {
			short := latest
			if len(short) > 7 {
				short = short[:7]
			}
			descParts = append(descParts, fmt.Sprintf("latest: %s", short))
		}
		desc := strings.Join(descParts, ", ")
		if desc == "" {
			desc = src.URL
		}
		// Note: cmd should not include "*" marker; it is only for display.
		_ = labelName
		opts = append(opts, ui.SlashOption{
			Cmd:  "/config update-skill " + name,
			Desc: desc,
		})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeySkillNone)}}
	}
	return opts
}

func getSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	list, _ := skills.List()
	parts := strings.Fields(filter)
	if len(parts) == 0 {
		if len(list) == 0 {
			return []ui.SlashOption{{Cmd: i18n.T(lang, i18n.KeySkillNone), Desc: ""}}
		}
		opts := make([]ui.SlashOption, 0, len(list))
		for _, s := range list {
			cmdName := s.LocalName
			if cmdName == "" {
				cmdName = s.Name
			}
			opts = append(opts, ui.SlashOption{Cmd: "/skill " + cmdName, Desc: s.Description})
		}
		return opts
	}

	skillName := parts[0]
	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		// No such skill: show skills whose name contains filter
		filterLower := strings.ToLower(skillName)
		opts := make([]ui.SlashOption, 0)
		for _, s := range list {
			if strings.Contains(strings.ToLower(s.Name), filterLower) {
				cmdName := s.LocalName
				if cmdName == "" {
					cmdName = s.Name
				}
				opts = append(opts, ui.SlashOption{Cmd: "/skill " + cmdName, Desc: s.Description})
			}
		}
		if len(opts) == 0 && len(list) > 0 {
			return opts
		}
		if len(opts) == 0 {
			return []ui.SlashOption{{Cmd: i18n.T(lang, i18n.KeySkillNone), Desc: ""}}
		}
		return opts
	}

	// Skill exists: no dropdown; user types natural language after "/skill <name> "
	return []ui.SlashOption{}
}
