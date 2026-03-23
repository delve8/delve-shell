package ui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/skills"
)

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry {
		if strings.HasPrefix(text, e.prefix) {
			rest := strings.TrimPrefix(text, e.prefix)
			return e.handle(m, rest)
		}
	}
	return m, nil, false
}

func init() {
	// NOTE: order matters for prefix overlaps. Keep it explicit and deterministic.
	registerSlashPrefix("/run ", slashPrefixDispatchEntry{
		prefix: "/run ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			cmd := strings.TrimSpace(rest)
			if mm.ExecDirectChan != nil && cmd != "" {
				mm.ExecDirectChan <- cmd
			} else if cmd == "" {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageRun))))
			}
			return mm, nil, true
		},
	})

	registerSlashPrefix("/skill ", slashPrefixDispatchEntry{
		prefix: "/skill ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) < 1 {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageSkill))))
				return mm, nil, true
			}
			skillName := fields[0]
			naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
			if naturalLanguage == "" {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageSkill))))
				return mm, nil, true
			}
			skillDir := skills.SkillDir(skillName)
			if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeySkillNotFound))))
				return mm, nil, true
			}
			skillContent, err := skills.ReadSKILLContent(skillDir)
			if err != nil {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.Tf(mm.getLang(), i18n.KeySkillInstallFailed, err))))
				return mm, nil, true
			}
			payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
			if mm.SubmitChan != nil {
				mm.SubmitChan <- payload
				mm.WaitingForAI = true
			}
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			return mm, nil, true
		},
	})

	registerSlashPrefix("/config llm base_url ", slashPrefixDispatchEntry{
		prefix: "/config llm base_url ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigLLM("base_url", strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config llm api_key ", slashPrefixDispatchEntry{
		prefix: "/config llm api_key ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigLLM("api_key", strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config llm model ", slashPrefixDispatchEntry{
		prefix: "/config llm model ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigLLM("model", strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config add-remote ", slashPrefixDispatchEntry{
		prefix: "/config add-remote ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigAddRemote(strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	registerSlashPrefix("/config auto-run ", slashPrefixDispatchEntry{
		prefix: "/config auto-run ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigAllowlistAutoRun(strings.TrimSpace(rest))
			return mm, nil, true
		},
	})
	// Keep prefix without trailing space so "/config add-skill" matches too.
	// Exact "/config add-skill" is handled by dispatchSlashExact.
	registerSlashPrefix("/config add-skill", slashPrefixDispatchEntry{
		prefix: "/config add-skill",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
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
					// Preserve existing parsing behavior.
					ref = fields[1]
					path = fields[2]
				}
			}
			mm = mm.openAddSkillOverlay(url, ref, path)
			return mm, nil, true
		},
	})
	// Parse even without trailing space; exact "/config update-skill" is expected
	// to show usage/error from the existing logic.
	registerSlashPrefix("/config update-skill", slashPrefixDispatchEntry{
		prefix: "/config update-skill",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyDescConfigUpdateSkill))))
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil, true
			}
			skillName := fields[0]
			mm = mm.openUpdateSkillOverlay(skillName)
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.SlashSuggestIndex = 0
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})

	registerSlashPrefix("/sessions ", slashPrefixDispatchEntry{
		prefix: "/sessions ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			id := strings.TrimSpace(rest)
			if id == "" {
				return mm, nil, true
			}
			if mm.SessionSwitchChan != nil {
				sessionPath := filepath.Join(config.HistoryDir(), id+".jsonl")
				select {
				case mm.SessionSwitchChan <- sessionPath:
				default:
				}
			}
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})

	registerSlashPrefix("/config del-remote ", slashPrefixDispatchEntry{
		prefix: "/config del-remote ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			nameOrTarget := strings.TrimSpace(rest)
			if nameOrTarget == "" {
				return mm, nil, true
			}
			return mm.applyConfigRemoveRemote(nameOrTarget), nil, true
		},
	})

	registerSlashPrefix("/config del-skill ", slashPrefixDispatchEntry{
		prefix: "/config del-skill ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			name := strings.TrimSpace(rest)
			if name == "" {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageSkillRemove))))
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil, true
			}
			if err := skillsvc.Remove(name); err != nil {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.Tf(mm.getLang(), i18n.KeySkillRemoveFailed, err))))
			} else {
				mm.Messages = append(mm.Messages, suggestStyle.Render(mm.delveMsg(i18n.Tf(mm.getLang(), i18n.KeySkillRemoved, name))))
			}
			mm.Input.SetValue("")
			mm.Input.CursorEnd()
			mm.Viewport.SetContent(mm.buildContent())
			mm.Viewport.GotoBottom()
			return mm, nil, true
		},
	})

	registerSlashPrefix("/remote on ", slashPrefixDispatchEntry{
		prefix: "/remote on ",
		handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			target := strings.TrimSpace(rest)
			if target == "" {
				return mm, nil, true
			}
			if mm.RemoteOnChan != nil {
				select {
				case mm.RemoteOnChan <- target:
				default:
				}
			}
			return mm, nil, true
		},
	})
}
