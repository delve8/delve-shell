package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/skill/git"
	"delve-shell/internal/skill/store"
	slashskill "delve-shell/internal/slash/skill"
	"delve-shell/internal/ui"
)

func registerSlashOptionsProvider() {
	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)

		if normalizedLower == SlashSubcommand || strings.HasPrefix(normalizedLower, SlashSubcommand+" ") {
			filter := strings.TrimSpace(strings.TrimPrefix(normalized, SlashSubcommand))
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
	list, err := skillstore.List()
	if err != nil || len(list) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-skill", Desc: i18n.T(i18n.KeySkillNone)}}
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
		return []ui.SlashOption{{Cmd: "/config del-skill", Desc: i18n.T(i18n.KeySkillNone)}}
	}
	return opts
}

func getUpdateSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	sources, err := skillstore.ListSources()
	if err != nil || len(sources) == 0 {
		return []ui.SlashOption{{Cmd: "/config update-skill", Desc: i18n.T(i18n.KeySkillNone)}}
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
		_ = labelName
		opts = append(opts, ui.SlashOption{
			Cmd:  "/config update-skill " + name,
			Desc: desc,
		})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config update-skill", Desc: i18n.T(i18n.KeySkillNone)}}
	}
	return opts
}

func getSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	list, _ := skillstore.List()
	parts := strings.Fields(filter)
	if len(parts) == 0 {
		return buildSkillSlashOptions(list, true)
	}

	skillName := parts[0]
	if len(parts) == 1 && skillName == slashskill.ReservedNew {
		return []ui.SlashOption{newSkillInstallOption()}
	}
	skillDir := skillstore.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil || skillNameCollisionKeepsDropdown(skillName) {
		filterLower := strings.ToLower(skillName)
		opts := make([]ui.SlashOption, 0, len(list)+1)
		includeReserved := skillReservedMatch(skillName)
		for _, s := range list {
			if skillName == slashskill.ReservedNew && skillOptionName(s) == slashskill.FilterNew {
				continue
			}
			if skillFilterMatch(s, filterLower) {
				opts = append(opts, skillSlashOption(s))
			}
		}
		if includeReserved {
			opts = append(opts, newSkillInstallOption())
		}
		if len(opts) == 0 && len(list) > 0 {
			return opts
		}
		if len(opts) == 0 {
			return []ui.SlashOption{{Cmd: i18n.T(i18n.KeySkillNone), Desc: ""}}
		}
		return opts
	}

	return []ui.SlashOption{}
}

func skillNameCollisionKeepsDropdown(skillName string) bool {
	return strings.EqualFold(skillName, slashskill.FilterNew) && skillName != slashskill.ReservedNew
}

func buildSkillSlashOptions(list []skillstore.SkillMeta, includeReserved bool) []ui.SlashOption {
	if len(list) == 0 {
		if includeReserved {
			return []ui.SlashOption{newSkillInstallOption()}
		}
		return []ui.SlashOption{{Cmd: i18n.T(i18n.KeySkillNone), Desc: ""}}
	}
	opts := make([]ui.SlashOption, 0, len(list)+1)
	for _, s := range list {
		opts = append(opts, skillSlashOption(s))
	}
	if includeReserved {
		opts = append(opts, newSkillInstallOption())
	}
	return opts
}

func newSkillInstallOption() ui.SlashOption {
	return ui.SlashOption{
		Cmd:  slashskill.Command(slashskill.ReservedNew),
		Desc: i18n.T(i18n.KeyDescSkillInstall),
	}
}

func skillSlashOption(s skillstore.SkillMeta) ui.SlashOption {
	cmdName := skillOptionNameMeta(s)
	return ui.SlashOption{
		Cmd:       slashskill.Prefix + cmdName,
		Desc:      s.Description,
		FillValue: slashskill.Prefix + cmdName,
	}
}

func skillOptionNameMeta(s skillstore.SkillMeta) string {
	if s.LocalName != "" {
		return s.LocalName
	}
	return s.Name
}

func skillOptionName(s skillstore.SkillMeta) string {
	return strings.ToLower(skillOptionNameMeta(s))
}

func skillReservedMatch(filter string) bool {
	if filter == "" {
		return true
	}
	if filter == slashskill.ReservedNew {
		return true
	}
	filterLower := strings.ToLower(filter)
	return strings.HasPrefix(slashskill.FilterNew, filterLower)
}

func skillFilterMatch(s skillstore.SkillMeta, filterLower string) bool {
	if filterLower == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s.Name), filterLower) || strings.Contains(skillOptionName(s), filterLower)
}
