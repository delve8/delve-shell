package ui

import "delve-shell/internal/approvalview"

// ApprovalRequestMsg asks the UI to show an approval card.
// Pending is a UI view-model; Respond is invoked by UI when user decides.
type ApprovalRequestMsg struct {
	Pending *approvalview.PendingApproval
}

// SensitiveConfirmationRequestMsg asks the UI to show a sensitive confirmation card.
type SensitiveConfirmationRequestMsg struct {
	Pending *approvalview.PendingSensitive
}

// AgentReplyMsg is the agent's reply to the user message.
type AgentReplyMsg struct {
	Reply     string
	ErrText   string // already human-readable
	Cancelled bool
}

// SystemNotifyMsg is a system/tool notification (e.g. connected to remote, switched to local), not from the AI.
type SystemNotifyMsg struct {
	Text string
}

// CommandExecutedMsg carries command execution process and result (allowlist/approved/direct/suggested) for display in the conversation.
type CommandExecutedMsg struct {
	Command   string
	Allowed   bool
	Direct    bool // true = /run direct execution, no AI
	Result    string
	Sensitive bool // true = result contains private data, not stored and LLM sees "done"
	Suggested bool // true = suggest mode, command was not executed (user can copy)
}

// ConfigReloadedMsg notifies the UI that config/allowlist was reloaded; next message will use new config.
type ConfigReloadedMsg struct{}

// SessionSwitchedMsg notifies the UI that the session was switched (/new or /sessions).
type SessionSwitchedMsg struct{}

// RemoteStatusMsg notifies the UI that the executor is local or remote, for header display.
type RemoteStatusMsg struct {
	Active bool   // true = remote, false = local
	Label  string // e.g. "dev (root@1.2.3.4)" or "user@host"
}

// RemoteConnectDoneMsg notifies the UI that a /remote on connection attempt finished (from the add-remote overlay).
// When Success is true, the UI closes the overlay and refocuses; when false, the UI clears the "Connecting..." state (auth overlay may follow).
type RemoteConnectDoneMsg struct {
	Success bool
	Label   string
	Err     string
}

// RemoteAuthPromptMsg asks the user to provide additional credentials for a remote target.
type RemoteAuthPromptMsg struct {
	Target                string
	Err                   string
	UseConfiguredIdentity bool
}

// OverlayCloseMsg closes any active overlay.
type OverlayCloseMsg struct{}

// OverlayShowMsg shows an overlay with the given title and content.
type OverlayShowMsg struct {
	Title   string
	Content string
}

// ConfigLLMCheckDoneMsg is sent when the async LLM check (after save) finishes.
// Err non-nil means check failed; CorrectedBaseURL non-empty means /v1 was added and config was updated.
type ConfigLLMCheckDoneMsg struct {
	Err              error
	CorrectedBaseURL string
}

// AddSkillRefsLoadedMsg is sent when branch/tag list for add-skill URL has been loaded (for Ref dropdown).
type AddSkillRefsLoadedMsg struct {
	Refs []string
}

// AddSkillPathsLoadedMsg is sent when directory paths in repo have been loaded (for Path dropdown).
type AddSkillPathsLoadedMsg struct {
	Paths []string
}

// RunCompletionCacheMsg provides a cached list of candidate strings for /run completion.
// RemoteLabel identifies which remote the list belongs to (empty for local).
type RunCompletionCacheMsg struct {
	RemoteLabel string
	Commands    []string
}

// SlashSubmitRelayMsg carries structured slash intent from host controller back into Update (§10.8.1).
// Handlers must call executeMainEnterCommandNoRelay, not handleMainEnterCommand, to avoid relay recursion.
type SlashSubmitRelayMsg struct {
	RawLine            string
	SlashSelectedIndex int
	// InputLine is set when relaying slash-mode Enter (preserve raw input buffer).
	InputLine string
}
