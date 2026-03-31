package uivm

// PendingOfflinePaste is the UI view-model for offline (manual) command relay: show command, user pastes output.
type PendingOfflinePaste struct {
	Command   string
	Reason    string
	RiskLevel string
	Respond   func(text string, cancelled bool)
}
