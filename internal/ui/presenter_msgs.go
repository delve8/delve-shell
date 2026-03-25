package ui

import tea "github.com/charmbracelet/bubbletea"

// Presenter message factories (used by uipresenter; keeps struct literals out of the host→TUI boundary).

func NewConfigReloadedMsg() tea.Msg { return ConfigReloadedMsg{} }

func NewSessionSwitchedMsg() tea.Msg { return SessionSwitchedMsg{} }

func NewAgentReplyMsg(reply string, err error) tea.Msg {
	return AgentReplyMsg{Reply: reply, Err: err}
}

func NewSystemNotifyMsg(text string) tea.Msg {
	return SystemNotifyMsg{Text: text}
}

func NewCommandExecutedDirectMsg(cmd, result string) tea.Msg {
	return CommandExecutedMsg{Command: cmd, Direct: true, Result: result}
}

func NewCommandExecutedFromToolMsg(cmd string, allowed bool, result string, sensitive, suggested bool) tea.Msg {
	return CommandExecutedMsg{
		Command:   cmd,
		Allowed:   allowed,
		Direct:    false,
		Result:    result,
		Sensitive: sensitive,
		Suggested: suggested,
	}
}

func NewRemoteStatusMsg(active bool, label string) tea.Msg {
	return RemoteStatusMsg{Active: active, Label: label}
}

func NewRemoteConnectDoneMsg(success bool, label, errText string) tea.Msg {
	return RemoteConnectDoneMsg{Success: success, Label: label, Err: errText}
}

func NewOverlayCloseMsg() tea.Msg { return OverlayCloseMsg{} }

func NewOverlayShowMsg(title, content string) tea.Msg {
	return OverlayShowMsg{Title: title, Content: content}
}

func NewConfigLLMCheckDoneMsg(err error, correctedBaseURL string) tea.Msg {
	return ConfigLLMCheckDoneMsg{Err: err, CorrectedBaseURL: correctedBaseURL}
}

func NewAddSkillRefsLoadedMsg(refs []string) tea.Msg {
	return AddSkillRefsLoadedMsg{Refs: refs}
}

func NewAddSkillPathsLoadedMsg(paths []string) tea.Msg {
	return AddSkillPathsLoadedMsg{Paths: paths}
}

func NewRunCompletionCacheMsg(remoteLabel string, commands []string) tea.Msg {
	return RunCompletionCacheMsg{RemoteLabel: remoteLabel, Commands: commands}
}
