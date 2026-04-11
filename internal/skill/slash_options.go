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
			switch {
			case filter == slashskill.ReservedRemove || strings.HasPrefix(filter, slashskill.ReservedRemove+" "):
				filter = strings.TrimSpace(strings.TrimPrefix(filter, slashskill.ReservedRemove))
				return getDelSkillSlashOptions(lang, filter), true
			case filter == slashskill.ReservedUpdate || strings.HasPrefix(filter, slashskill.ReservedUpdate+" "):
				filter = strings.TrimSpace(strings.TrimPrefix(filter, slashskill.ReservedUpdate))
				return getUpdateSkillSlashOptions(lang, filter), true
			default:
				return getSkillSlashOptions(lang, filter), true
			}
		}

		return nil, false
	})
}

func getDelSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	list, err := skillstore.List()
	if err != nil || len(list) == 0 {
		return []ui.SlashOption{{Cmd: slashskill.Command(slashskill.ReservedRemove), Desc: i18n.T(i18n.KeySkillNone)}}
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
		opts = append(opts, ui.SlashOption{Cmd: slashskill.Command(slashskill.ReservedRemove) + " " + cmdName, Desc: desc})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: slashskill.Command(slashskill.ReservedRemove), Desc: i18n.T(i18n.KeySkillNone)}}
	}
	return opts
}

func getUpdateSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	sources, err := skillstore.ListSources()
	if err != nil || len(sources) == 0 {
		return []ui.SlashOption{{Cmd: slashskill.Command(slashskill.ReservedUpdate), Desc: i18n.T(i18n.KeySkillNone)}}
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
			Cmd:  slashskill.Command(slashskill.ReservedUpdate) + " " + name,
			Desc: desc,
		})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: slashskill.Command(slashskill.ReservedUpdate), Desc: i18n.T(i18n.KeySkillNone)}}
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
	if len(parts) == 1 && (skillName == slashskill.ReservedNew || skillName == slashskill.ReservedRemove || skillName == slashskill.ReservedUpdate) {
		return reservedSkillOptionsFiltered(skillName)
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
			opts = append(opts, reservedSkillOptionsFiltered(skillName)...)
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
	return reservedSkillTokenCollision(skillName)
}

func buildSkillSlashOptions(list []skillstore.SkillMeta, includeReserved bool) []ui.SlashOption {
	if len(list) == 0 {
		if includeReserved {
			return reservedSkillOptionsFiltered("")
		}
		return []ui.SlashOption{{Cmd: i18n.T(i18n.KeySkillNone), Desc: ""}}
	}
	opts := make([]ui.SlashOption, 0, len(list)+3)
	for _, s := range list {
		opts = append(opts, skillSlashOption(s))
	}
	if includeReserved {
		opts = append(opts, reservedSkillOptionsFiltered("")...)
	}
	return opts
}

func reservedSkillOptionsFiltered(filter string) []ui.SlashOption {
	all := []struct {
		title  string
		lower  string
		descID string
	}{
		{title: slashskill.ReservedNew, lower: slashskill.FilterNew, descID: i18n.KeyDescSkillInstall},
		{title: slashskill.ReservedRemove, lower: slashskill.FilterRemove, descID: i18n.KeyDescSkillRemove},
		{title: slashskill.ReservedUpdate, lower: slashskill.FilterUpdate, descID: i18n.KeyDescSkillUpdate},
	}
	opts := make([]ui.SlashOption, 0, len(all))
	for _, reserved := range all {
		if !filterMatchesReservedSkill(filter, reserved.title, reserved.lower) {
			continue
		}
		opts = append(opts, ui.SlashOption{
			Cmd:  slashskill.Command(reserved.title),
			Desc: i18n.T(reserved.descID),
		})
	}
	return opts
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
	for _, reserved := range []struct {
		title  string
		filter string
	}{
		{title: slashskill.ReservedNew, filter: slashskill.FilterNew},
		{title: slashskill.ReservedRemove, filter: slashskill.FilterRemove},
		{title: slashskill.ReservedUpdate, filter: slashskill.FilterUpdate},
	} {
		if filterMatchesReservedSkill(filter, reserved.title, reserved.filter) {
			return true
		}
	}
	return false
}

func skillFilterMatch(s skillstore.SkillMeta, filterLower string) bool {
	if filterLower == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s.Name), filterLower) || strings.Contains(skillOptionName(s), filterLower)
}

func reservedSkillTokenCollision(skillName string) bool {
	for _, reserved := range []struct {
		title  string
		filter string
	}{
		{title: slashskill.ReservedNew, filter: slashskill.FilterNew},
		{title: slashskill.ReservedRemove, filter: slashskill.FilterRemove},
		{title: slashskill.ReservedUpdate, filter: slashskill.FilterUpdate},
	} {
		if strings.EqualFold(skillName, reserved.filter) && skillName != reserved.title {
			return true
		}
	}
	return false
}

func filterMatchesReservedSkill(filter, title, lower string) bool {
	if filter == "" {
		return true
	}
	if filter == title {
		return true
	}
	filterLower := strings.ToLower(filter)
	return strings.HasPrefix(lower, filterLower)
}
