package ui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/git"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/remotesvc"
)

const addSkillFieldCount = 4

// openAddSkillOverlay opens the Add skill dialog. url, ref, path can be pre-filled (e.g. from slash args).
func (m Model) openAddSkillOverlay(url, ref, path string) Model {
	m.OverlayActive = true
	m.OverlayTitle = i18n.T(m.getLang(), i18n.KeyAddSkillTitle)
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
	// Default local name from path last segment when provided.
	if p := strings.TrimSpace(path); p != "" {
		if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
			p = p[idx+1:]
		}
		m.AddSkillNameInput.SetValue(p)
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

// RunListRefsCmd runs git.ListRefs in the background and sends AddSkillRefsLoadedMsg. Call when Ref field is focused and URL is set.
func RunListRefsCmd(url string) tea.Cmd {
	return func() tea.Msg {
		refs := git.ListRefs(context.Background(), url)
		return AddSkillRefsLoadedMsg{Refs: refs}
	}
}

// RunListPathsCmd runs git.ListPaths in the background and sends AddSkillPathsLoadedMsg. Call when Path field is focused and URL is set.
func RunListPathsCmd(url, ref string) tea.Cmd {
	return func() tea.Msg {
		paths, _ := git.ListPaths(context.Background(), url, ref)
		return AddSkillPathsLoadedMsg{Paths: paths}
	}
}

// addSkillPathOptions are conventional paths for the Path dropdown (same as skill discovery dirs).
var addSkillPathOptions = []string{
	".",
	"skills",
	"skills/.curated",
	"skills/.experimental",
	"skills/.system",
	".agents/skills",
	".agent/skills",
	".claude/skills",
}

// filterByPrefix returns elements of s that have the given prefix (case-sensitive).
func filterByPrefix(s []string, prefix string) []string {
	if prefix == "" {
		return s
	}
	var out []string
	for _, v := range s {
		if strings.HasPrefix(v, prefix) {
			out = append(out, v)
		}
	}
	return out
}

func (m Model) updateAddSkillPathCandidates() Model {
	source := addSkillPathOptions
	if len(m.AddSkillPathsFullList) > 0 {
		source = m.AddSkillPathsFullList
	}
	m.AddSkillPathCandidates = filterByPrefix(source, m.AddSkillPathInput.Value())
	m.AddSkillPathIndex = 0
	return m
}

// skillInvocationPrompt returns the user-message payload for /skill <name> <natural language>.
// It instructs the model to use only this skill (run_skill) and includes the full SKILL.md and the user request.
func skillInvocationPrompt(skillName, skillContent, naturalLanguage string) string {
	const header = `[Skill invocation] Fulfill the user's request using ONLY the skill below. Use the run_skill tool with this skill's scripts and parameters; do not suggest arbitrary shell commands unless the skill documentation explicitly allows it.`

	return header + "\n\n## Skill: " + skillName + "\n\n" + skillContent + "\n\n## User request\n\n" + naturalLanguage
}

// applyAllowlistAutoRunSwitch sets runtime allowlist auto-run (on -> true, off -> false) and sends to AllowlistAutoRunChangeChan; does not write config.
func (m Model) applyAllowlistAutoRunSwitch(value string) Model {
	value = strings.TrimSpace(strings.ToLower(value))
	lang := m.getLang()
	var on bool
	switch value {
	case "on":
		on = true
	case "off":
		on = false
	default:
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigAutoRunRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if m.AllowlistAutoRunChangeChan != nil {
		select {
		case m.AllowlistAutoRunChangeChan <- on:
		default:
		}
	}
	display := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyAllowlistAutoRunSetTo, display))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// applyConfigAllowlistAutoRun sets allowlist_auto_run in config and writes; next startup will use it.
// value: "list-only" -> on, "disable" -> off.
func (m Model) applyConfigAllowlistAutoRun(value string) Model {
	value = strings.TrimSpace(strings.ToLower(value))
	var on bool
	switch value {
	case "list-only":
		on = true
	case "disable":
		on = false
	default:
		lang := m.getLang()
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigAutoRunRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	cfg.AllowlistAutoRun = &on
	if on {
		cfg.Mode = "run"
	} else {
		cfg.Mode = "suggest"
	}
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	display := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigSavedAllowlistAutoRun, display))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigAllowlistUpdate merges built-in default allowlist into current allowlist.yaml, appending only missing patterns.
func (m Model) applyConfigAllowlistUpdate() Model {
	lang := m.getLang()
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyAllowlistUpdateDone, added))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigAddRemote adds a remote via /config add-remote <user@host> [name] [identity_file]. Name is optional.
func (m Model) applyConfigAddRemote(args string) Model {
	lang := m.getLang()
	parts := strings.Fields(args)
	if len(parts) < 1 {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+"Usage: /config add-remote <user@host> [name] [identity_file]"))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	target := parts[0]
	name := ""
	identityFile := ""
	if len(parts) >= 2 {
		name = parts[1]
	}
	if len(parts) >= 3 {
		identityFile = parts[2]
	}
	if err := remotesvc.Add(target, name, identityFile); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	display := target
	if name != "" {
		display = name + " (" + target + ")"
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigRemoveRemote removes a remote via /config del-remote <name-or-target> (name or target from list).
func (m Model) applyConfigRemoveRemote(nameOrTarget string) Model {
	lang := m.getLang()
	nameOrTarget = strings.TrimSpace(nameOrTarget)
	if nameOrTarget == "" {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+"Usage: select a remote from /config del-remote list"))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if err := remotesvc.Remove(nameOrTarget); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigRemoteRemoved, nameOrTarget))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}
