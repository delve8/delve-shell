package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"delve-shell/internal/host/app"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

// Model is the Bubble Tea session and approval UI.
type Model struct {
	Input    textinput.Model
	Viewport viewport.Model
	Messages []string
	Approval ApprovalState
	Layout        LayoutState
	Interaction   InteractionState

	// Overlay state: when Overlay.Active is true, a modal is rendered on top of the main UI.
	Overlay OverlayState

	// Host is the injectable host façade (bus send endpoints + UI mirrors). Non-nil after NewModel.
	Host app.Host
}

// InteractionState stores transient keyboard/interaction state.
type InteractionState struct {
	slashSuggestIndex int  // 0..len(visible)-1 when input starts with /
	ChoiceIndex       int  // 0-based selection when in Pending/PendingSensitive/PendingSuggested; Up/Down to move, Enter to confirm
	WaitingForAI      bool // when true only blocks submitting new messages (Enter); /xxx slash commands always allowed
}

// ApprovalState stores current pending approvals.
type ApprovalState struct {
	pending          *agent.ApprovalRequest
	pendingSensitive *agent.SensitiveConfirmationRequest
}

// LayoutState stores terminal layout dimensions for rendering.
type LayoutState struct {
	Width  int
	Height int
}

// OverlayState stores generic modal overlay state shared across features.
type OverlayState struct {
	Active   bool
	Title    string
	Content  string
	Viewport viewport.Model
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Input.Cursor.BlinkCmd(), tea.WindowSize())
}

// getLang returns the UI language for i18n. Currently UI is English-only.
func (m Model) getLang() string {
	return "en"
}

// delveMsg prefixes msg with "Delve: " for tool/system messages (config, session, notify, etc.).
func (m Model) delveMsg(msg string) string {
	return i18n.T(m.getLang(), i18n.KeyDelveLabel) + " " + msg
}

// Update implements tea.Model.
//
// Routing (first match wins):
//   - messageProviderChain — feature-registered handlers (session, config-LLM, skills); see RegisterMessageProvider.
//   - update_lifecycle.go — WindowSize, Blur, Focus, overlay open/close, mouse / viewport.
//   - update_overlay_key.go then update_keymsg.go, update_slash.go, update_approval.go — keyboard when overlay vs main input.
//   - update_approval.go, update_events.go — agent approval, transcript, SlashSubmitRelayMsg.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.syncInputPlaceholder()

	for _, p := range messageProviderChain.List() {
		if m2, cmd, handled := p(m, msg); handled {
			return m2, cmd
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tea.BlurMsg:
		return m.handleBlurMsg()
	case tea.FocusMsg:
		return m.handleFocusMsg()
	case OverlayShowMsg:
		return m.handleOverlayShowMsg(msg)
	case OverlayCloseMsg:
		return m.handleOverlayCloseMsg()
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case ApprovalRequestMsg:
		return m.handleApprovalRequestMsg(msg)

	case SensitiveConfirmationRequestMsg:
		return m.handleSensitiveConfirmationRequestMsg(msg)

	case ConfigReloadedMsg:
		return m.handleConfigReloadedMsg()
	case AgentReplyMsg:
		return m.handleAgentReplyMsg(msg)
	case SystemNotifyMsg:
		return m.handleSystemNotifyMsg(msg)

	case CommandExecutedMsg:
		return m.handleCommandExecutedMsg(msg)

	case SlashSubmitRelayMsg:
		return m.handleSlashSubmitRelayMsg(msg)
	}

	return m, nil
}

// NewModel creates a Model with default input (slash commands and viewport scrolling).
// initialMessages if non-nil is used as existing conversation (e.g. after /sh return).
// host must be non-nil in production (typically *app.Runtime); nil is treated as app.Nop().
func NewModel(initialMessages []string, host app.Host) Model {
	if host == nil {
		host = app.Nop()
	}
	ti := textinput.New()
	ti.Placeholder = i18n.T("en", i18n.KeyPlaceholderInput)
	ti.Prompt = "> "
	ti.PromptStyle = inputPromptStyle
	ti.TextStyle = inputTextStyle
	ti.Cursor.Style = inputCursorStyle
	ti.CharLimit = 0
	ti.Width = defaultWidth - 4 // will be updated on first WindowSizeMsg to match terminal
	ti.Focus()
	vp := viewport.New(defaultWidth, defaultHeight-4)
	vp.MouseWheelEnabled = true
	msgs := []string(nil)
	if len(initialMessages) > 0 {
		msgs = make([]string, len(initialMessages))
		copy(msgs, initialMessages)
	}
	return Model{
		Input:    ti,
		Viewport: vp,
		Messages: msgs,
		Host:     host,
		Layout: LayoutState{
			Width:  defaultWidth,
			Height: defaultHeight,
		},
	}
}
