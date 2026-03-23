package ui

import tea "github.com/charmbracelet/bubbletea"

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
