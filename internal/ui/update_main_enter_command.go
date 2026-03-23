package ui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/skills"
)

func (m Model) handleMainEnterCommand(text string, slashSelectedPath string, slashSelectedIndex int) (Model, tea.Cmd) {
	switch text {
	case "/help", "/config llm", "/config add-skill", "/config add-remote", "/remote on", "/remote off", "/config update auto-run list", "/config reload", "/reload":
		if m2, cmd, handled := m.dispatchSlashExact(text); handled {
			return m2, cmd
		}
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
		return m2, cmd
	}

	switch {
	case text == "/q":
		return m, tea.Quit
	case text == "/sh":
		if m.ShellRequestedChan != nil {
			msgs := make([]string, len(m.Messages))
			copy(msgs, m.Messages)
			select {
			case m.ShellRequestedChan <- msgs:
			default:
			}
		}
		return m, tea.Quit
	case text == "/cancel":
		if m.WaitingForAI && m.CancelRequestChan != nil {
			select {
			case m.CancelRequestChan <- struct{}{}:
			default:
			}
			m.WaitingForAI = false
		} else {
			lang := m.getLang()
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyNoRequestInProgress))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
		}
		return m, nil

	case strings.HasPrefix(text, "/config llm base_url "):
		m = m.applyConfigLLM("base_url", strings.TrimPrefix(text, "/config llm base_url "))
		return m, nil
	case strings.HasPrefix(text, "/config llm api_key "):
		m = m.applyConfigLLM("api_key", strings.TrimPrefix(text, "/config llm api_key "))
		return m, nil
	case strings.HasPrefix(text, "/config llm model "):
		m = m.applyConfigLLM("model", strings.TrimPrefix(text, "/config llm model "))
		return m, nil

	case text == "/config show", text == "/config":
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil
	case text == "/config auto-run list-only":
		m = m.applyConfigAllowlistAutoRun("list-only")
		return m, nil
	case text == "/config auto-run disable":
		m = m.applyConfigAllowlistAutoRun("disable")
		return m, nil
	case strings.HasPrefix(text, "/config add-remote "):
		m = m.applyConfigAddRemote(strings.TrimPrefix(text, "/config add-remote "))
		return m, nil
	case strings.HasPrefix(text, "/config del-remote "):
		m = m.applyConfigRemoveRemote(strings.TrimSpace(strings.TrimPrefix(text, "/config del-remote ")))
		return m, nil
	case strings.HasPrefix(text, "/config auto-run "):
		arg := strings.TrimSpace(strings.TrimPrefix(text, "/config auto-run "))
		m = m.applyConfigAllowlistAutoRun(arg)
		return m, nil

	case strings.HasPrefix(text, "/config add-skill"):
		rest := strings.TrimSpace(text[len("/config add-skill"):])
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
		m = m.openAddSkillOverlay(url, ref, path)
		return m, nil

	case strings.HasPrefix(text, "/config del-skill "):
		rest := strings.TrimSpace(text[len("/config del-skill "):])
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkillRemove))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		}
		skillNameToRemove := fields[0]
		if err := skillsvc.Remove(skillNameToRemove); err != nil {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoveFailed, err))))
		} else {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoved, skillNameToRemove))))
		}
		m.Input.SetValue("")
		m.Input.CursorEnd()
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case strings.HasPrefix(text, "/config update-skill"):
		rest := strings.TrimSpace(text[len("/config update-skill"):])
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyDescConfigUpdateSkill))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		}
		skillName := fields[0]
		m = m.openUpdateSkillOverlay(skillName)
		m.Input.SetValue("")
		m.Input.CursorEnd()
		m.SlashSuggestIndex = 0
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case strings.HasPrefix(text, "/skill "):
		rest := strings.TrimSpace(text[len("/skill "):])
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkill))))
			return m, nil
		}
		skillName := fields[0]
		naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
		if naturalLanguage == "" {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkill))))
			return m, nil
		}
		skillDir := skills.SkillDir(skillName)
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeySkillNotFound))))
			return m, nil
		}
		skillContent, err := skills.ReadSKILLContent(skillDir)
		if err != nil {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillInstallFailed, err))))
			return m, nil
		}
		payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
		if m.SubmitChan != nil {
			m.SubmitChan <- payload
			m.WaitingForAI = true
		}
		m.Input.SetValue("")
		m.Input.CursorEnd()
		return m, nil

	case strings.HasPrefix(text, "/config update-skill "):
		// Kept for backward compatibility with older spacing in suggestions.
		return m, nil

	case strings.HasPrefix(text, "/"):
		// Use path captured before SlashSuggestIndex was reset; otherwise we would always send opts[0].
		if slashSelectedPath != "" {
			if m.SessionSwitchChan != nil {
				select {
				case m.SessionSwitchChan <- slashSelectedPath:
				default:
				}
			}
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		}
		opts := getSlashOptionsForInput(text, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
		vis := visibleSlashOptions(text, opts)
		var selectedOpt slashOption
		if slashSelectedIndex >= 0 && slashSelectedIndex < len(vis) {
			selectedOpt = opts[vis[slashSelectedIndex]]
		}
		// Sessions list empty: show message only when the single option is the session-none placeholder (not for del-skill etc).
		sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
		if selectedOpt.Path == "" && len(vis) == 1 && selectedOpt.Cmd == sessionNoneMsg {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0
			return m, nil
		}
		chosen := selectedOpt.Cmd
		// input must match chosen command; skip when only "/". "Fill only" already returned above.
		if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && (chosen == text || strings.HasPrefix(chosen, text)) {
			// user input matches chosen (full input then Enter) => execute
			if m2, cmd, handled := m.dispatchSlashExact(chosen); handled {
				return m2, cmd
			}
			if m2, cmd, handled := m.dispatchSlashPrefix(chosen); handled {
				return m2, cmd
			}
			if chosen == "/run <cmd>" {
				m.Input.SetValue("/run ")
				m.Input.CursorEnd()
				return m, nil
			}
			if chosen == "/config add-skill" {
				m = m.openAddSkillOverlay("", "", "")
				m.Input.SetValue("")
				m.Input.CursorEnd()
				m.SlashSuggestIndex = 0
				return m, nil
			}
			if chosen == "/config auto-run list-only" {
				m = m.applyConfigAllowlistAutoRun("list-only")
				return m, nil
			}
			if chosen == "/config auto-run disable" {
				m = m.applyConfigAllowlistAutoRun("disable")
				return m, nil
			}
			if chosen == "/config llm" {
				m = m.openConfigLLMOverlay()
				return m, nil
			}
			if chosen == "/new" {
				if m.SubmitChan != nil {
					m.SubmitChan <- "/new"
				}
				m.Input.SetValue("")
				m.Input.CursorEnd()
				m.SlashSuggestIndex = 0
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			}
			if m2, cmd, handled := m.handleSlashSelectedFallback(chosen); handled {
				return m2, cmd
			}
		}
		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUnknownCmd))))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil
	}

	if m.SubmitChan != nil {
		m.SubmitChan <- text
		m.WaitingForAI = true
	}
	return m, nil
}
