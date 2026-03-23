package ui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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

func (m Model) handleRemoteConnectDoneMsg(msg RemoteConnectDoneMsg) (Model, tea.Cmd) {
	// Connection attempt finished: clear any "connecting" states for add-remote or remote auth.
	m.AddRemoteConnecting = false
	m.AddRemoteError = ""
	m.AddRemoteOfferOverwrite = false
	m.RemoteAuthConnecting = false

	// When Remote Auth overlay is active, close it on successful connection.
	if m.RemoteAuthStep != "" {
		if msg.Success {
			m.OverlayActive = false
			m.OverlayTitle = ""
			m.OverlayContent = ""
			m.RemoteAuthStep = ""
			m.RemoteAuthTarget = ""
			m.RemoteAuthError = ""
			m.RemoteAuthUsername = ""
			m.PathCompletionCandidates = nil
			m.PathCompletionIndex = -1
			m.Input.Focus()
		}
		return m, nil
	}

	// Fallback: add-remote overlay (opened via /remote on or /config add-remote).
	m.AddRemoteActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	if msg.Success {
		m.OverlayActive = false
		m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleRemoteAuthPromptMsg(msg RemoteAuthPromptMsg) (Model, tea.Cmd) {
	m.AddRemoteConnecting = false
	m.AddRemoteActive = false
	m.OverlayActive = true
	m.OverlayTitle = "Remote Auth"
	m.RemoteAuthTarget = msg.Target
	m.RemoteAuthError = msg.Err
	m.ChoiceIndex = 0
	// When UseConfiguredIdentity is true, show a non-interactive "connecting with configured key" state.
	if msg.UseConfiguredIdentity {
		m.RemoteAuthStep = "auto_identity"
		m.RemoteAuthConnecting = true
		return m, nil
	}
	// Default: interactive auth flow starting from username.
	m.RemoteAuthConnecting = false
	m.RemoteAuthStep = "username" // first step: username only; Enter then shows "choose" (1/2) so username can contain 1 or 2
	m.RemoteAuthUsernameInput = textinput.New()
	m.RemoteAuthUsernameInput.Placeholder = "root"
	if i := strings.Index(msg.Target, "@"); i > 0 && i < len(msg.Target)-1 {
		m.RemoteAuthUsernameInput.SetValue(msg.Target[:i])
	} else {
		m.RemoteAuthUsernameInput.SetValue("root")
	}
	m.RemoteAuthUsernameInput.Focus()
	return m, nil
}

func (m Model) handleConfigLLMCheckDoneMsg(msg ConfigLLMCheckDoneMsg) (Model, tea.Cmd) {
	m.ConfigLLMChecking = false
	lang := m.getLang()
	if msg.Err != nil {
		m.ConfigLLMError = i18n.Tf(lang, i18n.KeyConfigLLMCheckFailed, msg.Err)
		m.Viewport.SetContent(m.buildContent())
		return m, nil
	}
	m.ConfigLLMError = ""
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigSavedLLM))))
	if msg.CorrectedBaseURL != "" {
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigLLMBaseURLAutoCorrected, msg.CorrectedBaseURL))))
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigLLMCheckOK))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	m.OverlayActive = false
	m.ConfigLLMActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m, nil
}

func (m Model) handleAddSkillRefsLoadedMsg(msg AddSkillRefsLoadedMsg) (Model, tea.Cmd) {
	if m.AddSkillActive {
		m.AddSkillRefsFullList = msg.Refs
		m.AddSkillRefCandidates = filterByPrefix(msg.Refs, m.AddSkillRefInput.Value())
		m.AddSkillRefIndex = 0
	}
	return m, nil
}

func (m Model) handleAddSkillPathsLoadedMsg(msg AddSkillPathsLoadedMsg) (Model, tea.Cmd) {
	if m.AddSkillActive {
		m.AddSkillPathsFullList = msg.Paths
		m = m.updateAddSkillPathCandidates()
	}
	return m, nil
}

func (m Model) handleApprovalRequestMsg(msg ApprovalRequestMsg) (Model, tea.Cmd) {
	// When an approval is requested, immediately refresh the viewport so the
	// approval card becomes visible, and scroll to bottom.
	m.Pending = msg
	m.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, nil
}

func (m Model) handleSensitiveConfirmationRequestMsg(msg SensitiveConfirmationRequestMsg) (Model, tea.Cmd) {
	// Same as approval: ensure the sensitive confirmation card is visible.
	m.PendingSensitive = msg
	m.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, nil
}
