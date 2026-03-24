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
	Input                      textinput.Model
	Viewport                   viewport.Model
	Messages                   []string
	Pending                    *agent.ApprovalRequest
	PendingSensitive           *agent.SensitiveConfirmationRequest
	Ports                      UIPorts
	Context                    RuntimeContextState
	// /run completion cache (best-effort).
	RunCompletion RunCompletionState
	Width       int
	Height      int
	Interaction InteractionState

	// Overlay state: when Overlay.Active is true, a modal is rendered on top of the main UI.
	Overlay OverlayState

	// Add-remote overlay state (username + host separate).
	// Fields: 0=host, 1=user, 2=name, 3=key path, 4=save-as-remote checkbox.
	AddRemote AddRemoteOverlayState

	// Remote auth overlay state.
	RemoteAuth RemoteAuthOverlayState
	// Path completion (shared): used for any path input with dropdown (auth identity key path, add-remote key path).
	PathCompletion PathCompletionState

	// InitialShowConfigLLM: when true, open Config LLM overlay on first WindowSizeMsg (e.g. no config / model empty at startup).
	InitialShowConfigLLM bool
	// Config LLM overlay state.
	ConfigLLM ConfigLLMOverlayState

	// Add-skill overlay: URL (required), ref, path and local name (optional).
	AddSkill AddSkillOverlayState

	// Update-skill overlay: choose ref and confirm update for an installed skill.
	UpdateSkill UpdateSkillOverlayState
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

// RemoteAuthOverlayState stores overlay-only state for remote authentication prompts.
// Step: "" = inactive, "choose" = selecting auth method, "password" = entering password, "identity" = entering key path.
type RemoteAuthOverlayState struct {
	Step          string
	Target        string
	Error         string
	Username      string          // username to use when submitting (default root)
	UsernameInput textinput.Model // username input in choose step
	Input         textinput.Model // for password or identity path
	Connecting    bool            // true while waiting for remote auth result ("Connecting..." state)
}

// AddRemoteOverlayState stores overlay-only state for add/connect remote dialogs.
type AddRemoteOverlayState struct {
	Active         bool
	UserInput      textinput.Model
	HostInput      textinput.Model
	NameInput      textinput.Model
	KeyInput       textinput.Model
	FieldIndex     int
	Error          string
	OfferOverwrite bool // when true, error was "already exists"; show overwrite hint and accept O to overwrite
	Save           bool // true = save/update remote config; false = only connect (for /remote on)
	Connect        bool // true when opened via /remote on; false for /config add-remote
	Connecting     bool // true while waiting for connection result (show "Connecting...")
}

// AddSkillOverlayState stores overlay-only state for add-skill flow.
type AddSkillOverlayState struct {
	Active         bool
	URLInput       textinput.Model
	RefInput       textinput.Model
	PathInput      textinput.Model
	NameInput      textinput.Model
	FieldIndex     int // 0=url, 1=ref, 2=path, 3=name
	Error          string
	RefsFullList   []string // all refs from remote (for filtering)
	RefCandidates  []string // refs filtered by Ref input prefix
	RefIndex       int      // selection in ref dropdown
	PathsFullList  []string // paths from git repo (when non-nil, Path dropdown uses this instead of static list)
	PathCandidates []string // path options filtered by Path input prefix
	PathIndex      int      // selection in path dropdown
}

// UpdateSkillOverlayState stores overlay-only state for update-skill flow.
type UpdateSkillOverlayState struct {
	Active        bool
	Name          string
	URL           string
	Path          string
	CurrentCommit string
	Refs          []string
	RefIndex      int
	LatestCommit  string
	Error         string
}

// PathCompletionState stores shared dropdown state for filesystem path completion.
type PathCompletionState struct {
	Candidates []string
	Index      int
}

// RuntimeContextState stores session and remote execution context reflected in UI.
type RuntimeContextState struct {
	CurrentSessionPath string // path of current session (excluded from /sessions list so switch loads another)
	RemoteActive       bool   // whether commands run on a remote executor
	RemoteLabel        string // label for remote in header, e.g. "dev (root@1.2.3.4)" or "user@host"
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

// OverlayState stores generic modal overlay state shared across features.
type OverlayState struct {
	Active   bool
	Title    string
	Content  string
	Viewport viewport.Model
}

// UIPorts are side-effect channels/getters injected by CLI host loop.
type UIPorts struct {
	SubmitChan                 chan<- string
	ExecDirectChan             chan<- string
	ShellRequestedChan         chan<- []string           // on /sh send current Messages to preserve after return
	CancelRequestChan          chan<- struct{}           // on /cancel request cancel of in-flight AI
	ConfigUpdatedChan          chan<- struct{}           // on /config save or /config reload, invalidate runner so next message reloads config/allowlist
	AllowlistAutoRunChangeChan chan<- bool               // runtime toggle for allowlist auto-run (true = list only, false = none)
	SessionSwitchChan          chan<- string             // on /sessions choice, send selected session path to continue
	RemoteOnChan               chan<- string             // on /remote on <target>, send resolved target/name to CLI
	RemoteOffChan              chan<- struct{}           // on /remote off, switch back to local
	RemoteAuthRespChan         chan<- RemoteAuthResponse // on remote password entry, send credentials back to CLI
	GetAllowlistAutoRun        func() bool               // for header and Pending card 2 vs 3 options
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

	for _, p := range messageProviders {
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
// initialSessionPath is the current session file path (excluded from /sessions list so first option is another session).
// initialShowConfigLLM: when true, Config LLM overlay is opened on first WindowSizeMsg (used when no config or model empty at startup).
func NewModel(
	submitChan chan<- string,
	execDirectChan chan<- string,
	shellRequestedChan chan<- []string,
	cancelRequestChan chan<- struct{},
	configUpdatedChan chan<- struct{},
	allowlistAutoRunChangeChan chan<- bool,
	sessionSwitchChan chan<- string,
	remoteOnChan chan<- string,
	remoteOffChan chan<- struct{},
	remoteAuthRespChan chan<- RemoteAuthResponse,
	getAllowlistAutoRun func() bool,
	initialMessages []string,
	initialSessionPath string,
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
		Input:                      ti,
		Viewport:                   vp,
		Messages:                   msgs,
		Ports: UIPorts{
			SubmitChan:                 submitChan,
			ExecDirectChan:             execDirectChan,
			ShellRequestedChan:         shellRequestedChan,
			CancelRequestChan:          cancelRequestChan,
			ConfigUpdatedChan:          configUpdatedChan,
			AllowlistAutoRunChangeChan: allowlistAutoRunChangeChan,
			SessionSwitchChan:          sessionSwitchChan,
			RemoteOnChan:               remoteOnChan,
			RemoteOffChan:              remoteOffChan,
			RemoteAuthRespChan:         remoteAuthRespChan,
			GetAllowlistAutoRun:        getAllowlistAutoRun,
		},
		Context: RuntimeContextState{
			CurrentSessionPath: initialSessionPath,
		},
		InitialShowConfigLLM:       initialShowConfigLLM,
		Width:                      defaultWidth,
		Height:                     defaultHeight,
	}
}
