package ui

import tea "github.com/charmbracelet/bubbletea"

// HandleAddSkillRefsLoadedMsg delegates to the internal add-skill refs handler.
func (m Model) HandleAddSkillRefsLoadedMsg(msg AddSkillRefsLoadedMsg) (Model, tea.Cmd) {
	return m.handleAddSkillRefsLoadedMsg(msg)
}

// HandleAddSkillPathsLoadedMsg delegates to the internal add-skill paths handler.
func (m Model) HandleAddSkillPathsLoadedMsg(msg AddSkillPathsLoadedMsg) (Model, tea.Cmd) {
	return m.handleAddSkillPathsLoadedMsg(msg)
}

// HandleRemoteConnectDoneMsg delegates to the internal remote connect done handler.
func (m Model) HandleRemoteConnectDoneMsg(msg RemoteConnectDoneMsg) (Model, tea.Cmd) {
	return m.handleRemoteConnectDoneMsg(msg)
}

// HandleRemoteAuthPromptMsg delegates to the internal remote auth prompt handler.
func (m Model) HandleRemoteAuthPromptMsg(msg RemoteAuthPromptMsg) (Model, tea.Cmd) {
	return m.handleRemoteAuthPromptMsg(msg)
}

// HandleSessionSwitchedMsg delegates to the internal session switched handler.
func (m Model) HandleSessionSwitchedMsg(msg SessionSwitchedMsg) (Model, tea.Cmd) {
	return m.handleSessionSwitchedMsg(msg)
}
