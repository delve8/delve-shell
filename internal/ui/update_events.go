package ui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

func (m Model) handleRemoteStatusMsg(msg RemoteStatusMsg) (Model, tea.Cmd) {
	m.RemoteActive = msg.Active
	m.RemoteLabel = msg.Label
	if msg.Active {
		// New remote active: clear any previous remote /run completion cache.
		m.RemoteRunLabel = msg.Label
		m.RemoteRunCommands = nil
	} else {
		// Switching back to local: drop any remote /run completion cache.
		m.RemoteRunLabel = ""
		m.RemoteRunCommands = nil
	}
	m.Viewport.SetContent(m.buildContent())
	return m, nil
}

func (m Model) handleRunCompletionCacheMsg(msg RunCompletionCacheMsg) (Model, tea.Cmd) {
	// Remote cache update (sent by CLI on successful /remote on).
	// Ignore stale results from previous remotes.
	if msg.RemoteLabel == "" || msg.RemoteLabel != m.RemoteLabel {
		return m, nil
	}
	m.RemoteRunLabel = msg.RemoteLabel
	m.RemoteRunCommands = msg.Commands
	return m, nil
}

func (m Model) handleSessionSwitchedMsg(msg SessionSwitchedMsg) (Model, tea.Cmd) {
	lang := m.getLang()
	m.CurrentSessionPath = msg.Path
	sessionID := ""
	if msg.Path != "" {
		sessionID = strings.TrimSuffix(filepath.Base(msg.Path), ".jsonl")
	}
	switchedLine := sessionSwitchedStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeySessionSwitchedTo, sessionID)))
	if msg.Path != "" {
		events, _ := history.ReadRecent(msg.Path, maxSessionHistoryEvents)
		msgs := sessionEventsToMessages(events, lang, m.Width)
		m.Messages = make([]string, 0, len(msgs)+2)
		m.Messages = append(m.Messages, msgs...)
		m.Messages = append(m.Messages, switchedLine)
	} else {
		m.Messages = []string{switchedLine}
	}
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, nil
}

func (m Model) handleConfigReloadedMsg() (Model, tea.Cmd) {
	lang := m.getLang()
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigReloaded))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, nil
}

func (m Model) handleAgentReplyMsg(msg AgentReplyMsg) (Model, tea.Cmd) {
	m.WaitingForAI = false
	lang := m.getLang()
	if msg.Err != nil {
		if errors.Is(msg.Err, context.Canceled) {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyCancelled))))
		} else if errors.Is(msg.Err, agent.ErrLLMNotConfigured) {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyErrLLMNotConfigured, config.ConfigPath()))))
		} else {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyErrorPrefix)+msg.Err.Error())))
		}
		m.Messages = append(m.Messages, "")
	} else if msg.Reply != "" {
		aiLine := i18n.T(lang, i18n.KeyAILabel) + msg.Reply
		w := m.Width
		if w <= 0 {
			w = 80
		}
		m.Messages = append(m.Messages, wrapString(aiLine, w))
		sepW := m.Width
		if sepW <= 0 {
			sepW = 80
		}
		m.Messages = append(m.Messages, separatorStyle.Render(strings.Repeat("─", sepW)))
	}
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, nil
}

func (m Model) handleSystemNotifyMsg(msg SystemNotifyMsg) (Model, tea.Cmd) {
	if msg.Text != "" {
		w := m.Width
		if w <= 0 {
			w = 80
		}
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(wrapString(msg.Text, w))))
		m.Messages = append(m.Messages, "")
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCommandExecutedMsg(msg CommandExecutedMsg) (Model, tea.Cmd) {
	lang := m.getLang()
	var tag string
	if msg.Direct {
		tag = i18n.T(lang, i18n.KeyRunTagDirect)
	} else if msg.Allowed {
		tag = i18n.T(lang, i18n.KeyRunTagAllowlist)
	} else {
		tag = i18n.T(lang, i18n.KeyRunTagApproved)
	}
	runLine := i18n.T(lang, i18n.KeyRunLabel) + msg.Command + " (" + tag + ")"
	w := m.Width
	if w <= 0 {
		w = 80
	}
	m.Messages = append(m.Messages, execStyle.Render(wrapString(runLine, w)))
	if msg.Sensitive {
		m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyResultSensitive)))
	}
	if msg.Result != "" {
		m.Messages = append(m.Messages, resultStyle.Render(wrapString(msg.Result, w)))
	}
	m.Messages = append(m.Messages, "") // blank line after command output
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, nil
}
