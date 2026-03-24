package ui

// hasPendingApproval reports whether the UI is in approval choice mode.
func (m Model) hasPendingApproval() bool {
	return m.Approval.Pending != nil || m.Approval.PendingSensitive != nil
}

// contentWidth returns a safe rendering width with fallback.
func (m Model) contentWidth() int {
	w := m.Layout.Width
	if w <= 0 {
		return 80
	}
	return w
}
