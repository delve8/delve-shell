package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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
	Ports    UIPorts
	Context  RuntimeContextState
	// /run completion cache (best-effort).
	RunCompletion RunCompletionState
	Layout        LayoutState
	Interaction   InteractionState

	// Overlay state: when Overlay.Active is true, a modal is rendered on top of the main UI.
	Overlay OverlayState

	// Startup stores one-time startup toggles consumed in lifecycle handlers.
	Startup StartupState
	// Config LLM overlay state.
	ConfigLLM ConfigLLMOverlayState
}

// ConfigLLMOverlayState stores overlay-only state for `/config llm`.
type ConfigLLMOverlayState struct {
	Active           bool
	Checking         bool // true while async "hello" check is in progress after save
	BaseURLInput     textinput.Model
	ApiKeyInput      textinput.Model
	ModelInput       textinput.Model
	MaxMessagesInput textinput.Model
	MaxCharsInput    textinput.Model
	FieldIndex       int // 0=base_url, 1=api_key, 2=model, 3=max_messages, 4=max_chars
	Error            string
}

// RuntimeContextState stores session and remote execution context reflected in UI.
type RuntimeContextState struct {
	RemoteActive bool   // whether commands run on a remote executor
	RemoteLabel  string // label for remote in header, e.g. "dev (root@1.2.3.4)" or "user@host"
	ConfigPath   string // config path for user-facing hints (injected by host)
}

// RunCompletionState stores local/remote completion caches for `/run`.
type RunCompletionState struct {
	LocalCommands  []string
	RemoteCommands []string
	RemoteLabel    string // which remote the RemoteCommands came from
}

// InteractionState stores transient keyboard/interaction state.
type InteractionState struct {
	SlashSuggestIndex int  // 0..len(visible)-1 when input starts with /
	ChoiceIndex       int  // 0-based selection when in Pending/PendingSensitive/PendingSuggested; Up/Down to move, Enter to confirm
	WaitingForAI      bool // when true only blocks submitting new messages (Enter); /xxx slash commands always allowed
}

// ApprovalState stores current pending approvals.
type ApprovalState struct {
	Pending          *agent.ApprovalRequest
	PendingSensitive *agent.SensitiveConfirmationRequest
}

// LayoutState stores terminal layout dimensions for rendering.
type LayoutState struct {
	Width  int
	Height int
}

// StartupState stores one-shot startup flags.
type StartupState struct {
	// InitialShowConfigLLM: when true, open Config LLM overlay on first WindowSizeMsg
	// (e.g. no config / model empty at startup).
	InitialShowConfigLLM bool
}

// OverlayState stores generic modal overlay state shared across features.
type OverlayState struct {
	Active   bool
	Title    string
	Content  string
	Viewport viewport.Model
}

// UIPorts are side-effect channels/getters injected by CLI host loop.
type UIPorts struct {
	SubmitChan          chan<- string
	GetAllowlistAutoRun func() bool // for header and Pending card 2 vs 3 options
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
	case ConfigLLMCheckDoneMsg:
		return m.handleConfigLLMCheckDoneMsg(msg)
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
	}

	return m, nil
}

// NewModel creates a Model with default input (slash commands and viewport scrolling).
// initialMessages if non-nil is used as existing conversation (e.g. after /sh return).
// initialShowConfigLLM: when true, Config LLM overlay is opened on first WindowSizeMsg (used when no config or model empty at startup).
func NewModel(
	submitChan chan<- string,
	getAllowlistAutoRun func() bool,
	initialMessages []string,
	initialShowConfigLLM bool,
) Model {
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
		Ports: UIPorts{
			SubmitChan:          submitChan,
			GetAllowlistAutoRun: getAllowlistAutoRun,
		},
		Context: RuntimeContextState{},
		Startup: StartupState{
			InitialShowConfigLLM: initialShowConfigLLM,
		},
		Layout: LayoutState{
			Width:  defaultWidth,
			Height: defaultHeight,
		},
	}
}
