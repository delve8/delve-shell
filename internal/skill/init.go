package skill

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/skills"
	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashExact("/config add-skill", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return openAddSkillOverlay(m, "", "", ""), nil
		},
		ClearInput: true,
	})

	ui.RegisterSlashPrefix("/skill ", ui.SlashPrefixDispatchEntry{
		Prefix: "/skill ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			return handleSlashSkillPrefix(m, rest), nil, true
		},
	})

	ui.RegisterSlashPrefix("/config add-skill", ui.SlashPrefixDispatchEntry{
		Prefix: "/config add-skill",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			rest = strings.TrimSpace(rest)
			url, ref, path := "", "", ""
			if rest != "" {
				fields := strings.Fields(rest)
				if len(fields) >= 1 {
					url = fields[0]
				}
				if len(fields) >= 2 {
					if strings.Contains(fields[1], "/") {
						path = fields[1]
					} else {
						ref = fields[1]
					}
				}
				if len(fields) >= 3 {
					ref = fields[1]
					path = fields[2]
				}
			}
			return openAddSkillOverlay(m, url, ref, path), nil, true
		},
	})

	ui.RegisterSlashPrefix("/config update-skill", ui.SlashPrefixDispatchEntry{
		Prefix: "/config update-skill",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			return handleSlashConfigUpdateSkillPrefix(m, rest), nil, true
		},
	})

	ui.RegisterSlashPrefix("/config del-skill ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config del-skill ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			return handleSlashConfigDelSkillPrefix(m, rest), nil, true
		},
	})

	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		_ string,
		_ []string,
		_ []string,
		_ bool,
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

	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		if m.AddSkillActive {
			return handleAddSkillOverlayKey(m, key, msg)
		}
		if m.UpdateSkillActive {
			return handleUpdateSkillOverlayKey(m, key)
		}
		return m, nil, false
	})

	// Delegate add-skill async messages (refs/paths list) to ui handler.
	ui.RegisterMessageProvider(func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
		switch t := msg.(type) {
		case ui.AddSkillRefsLoadedMsg:
			m2, cmd := m.HandleAddSkillRefsLoadedMsg(t)
			return m2, cmd, true
		case ui.AddSkillPathsLoadedMsg:
			m2, cmd := m.HandleAddSkillPathsLoadedMsg(t)
			return m2, cmd, true
		default:
			return m, nil, false
		}
	})
}

func openAddSkillOverlay(m ui.Model, url, ref, path string) ui.Model {
	lang := "en" // ui.getLang() currently always returns "en"
	m.OverlayActive = true
	m.OverlayTitle = i18n.T(lang, i18n.KeyAddSkillTitle)
	m.AddSkillActive = true
	m.AddSkillError = ""
	m.AddSkillFieldIndex = 0

	m.AddSkillURLInput = textinput.New()
	m.AddSkillURLInput.Placeholder = "https://github.com/owner/repo or owner/repo"
	m.AddSkillURLInput.SetValue(url)
	m.AddSkillURLInput.Focus()

	m.AddSkillRefInput = textinput.New()
	m.AddSkillRefInput.Placeholder = "main"
	m.AddSkillRefInput.SetValue(ref)
	m.AddSkillRefInput.Blur()

	m.AddSkillPathInput = textinput.New()
	m.AddSkillPathInput.Placeholder = "skills/foo"
	m.AddSkillPathInput.SetValue(path)
	m.AddSkillPathInput.Blur()

	m.AddSkillNameInput = textinput.New()
	m.AddSkillNameInput.Placeholder = "local skill name"
	// Derive local name from path last segment when provided.
	if p := strings.TrimSpace(path); p != "" {
		if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
			p = p[idx+1:]
		}
		m.AddSkillNameInput.SetValue(p)
		m.AddSkillNameInput.CursorEnd()
	} else {
		m.AddSkillNameInput.SetValue("")
	}
	m.AddSkillNameInput.Blur()

	m.AddSkillRefsFullList = nil
	m.AddSkillRefCandidates = nil
	m.AddSkillRefIndex = 0
	m.AddSkillPathsFullList = nil
	m.AddSkillPathCandidates = nil
	m.AddSkillPathIndex = 0
	return m
}

func getDelSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	list, err := skills.List()
	if err != nil || len(list) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeySkillNone), Path: ""}}
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
		opts = append(opts, ui.SlashOption{Cmd: "/config del-skill " + cmdName, Desc: desc, Path: ""})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeySkillNone), Path: ""}}
	}
	return opts
}

func getUpdateSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	sources, err := skills.ListSources()
	if err != nil || len(sources) == 0 {
		return []ui.SlashOption{{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeySkillNone), Path: ""}}
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
			Path: "",
		})
	}
	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeySkillNone), Path: ""}}
	}
	return opts
}

func getSkillSlashOptions(lang string, filter string) []ui.SlashOption {
	list, _ := skills.List()
	parts := strings.Fields(filter)
	if len(parts) == 0 {
		if len(list) == 0 {
			return []ui.SlashOption{{Cmd: i18n.T(lang, i18n.KeySkillNone), Desc: "", Path: ""}}
		}
		opts := make([]ui.SlashOption, 0, len(list))
		for _, s := range list {
			cmdName := s.LocalName
			if cmdName == "" {
				cmdName = s.Name
			}
			opts = append(opts, ui.SlashOption{Cmd: "/skill " + cmdName, Desc: s.Description, Path: ""})
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
				opts = append(opts, ui.SlashOption{Cmd: "/skill " + cmdName, Desc: s.Description, Path: ""})
			}
		}
		if len(opts) == 0 && len(list) > 0 {
			return opts
		}
		if len(opts) == 0 {
			return []ui.SlashOption{{Cmd: i18n.T(lang, i18n.KeySkillNone), Desc: "", Path: ""}}
		}
		return opts
	}

	// Skill exists: no dropdown; user types natural language after "/skill <name> "
	return []ui.SlashOption{}
}
