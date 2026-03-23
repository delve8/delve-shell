package ui

import (
	"strings"

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
	CurrentSessionPath         string                    // path of current session (excluded from /sessions list so switch loads another)
	GetAllowlistAutoRun        func() bool               // for header and Pending card 2 vs 3 options
	RemoteActive               bool                      // whether commands run on a remote executor
	RemoteLabel                string                    // label for remote in header, e.g. "dev (root@1.2.3.4)" or "user@host"
	// /run completion caches (best-effort).
	LocalRunCommands  []string
	RemoteRunCommands []string
	RemoteRunLabel    string // which remote the RemoteRunCommands came from
	Width             int
	Height            int
	SlashSuggestIndex int  // 0..len(visible)-1 when input starts with /
	ChoiceIndex       int  // 0-based selection when in Pending/PendingSensitive/PendingSuggested; Up/Down to move, Enter to confirm
	WaitingForAI      bool // when true only blocks submitting new messages (Enter); /xxx slash commands always allowed

	// Overlay state: when OverlayActive is true, a modal is rendered on top of the main UI.
	OverlayActive   bool
	OverlayTitle    string
	OverlayContent  string
	OverlayViewport viewport.Model

	// Add-remote overlay state (username + host separate).
	// Fields: 0=host, 1=user, 2=name, 3=key path, 4=save-as-remote checkbox.
	AddRemoteActive         bool
	AddRemoteUserInput      textinput.Model
	AddRemoteHostInput      textinput.Model
	AddRemoteNameInput      textinput.Model
	AddRemoteKeyInput       textinput.Model
	AddRemoteFieldIndex     int
	AddRemoteError          string
	AddRemoteOfferOverwrite bool // when true, error was "already exists"; show overwrite hint and accept O to overwrite
	AddRemoteSave           bool // true = save/update remote config; false = only connect (for /remote on)
	AddRemoteConnect        bool // true when opened via /remote on; false for /config add-remote
	AddRemoteConnecting     bool // true while waiting for connection result (show "Connecting...")

	// Remote auth overlay state.
	// RemoteAuthStep: "" = inactive, "choose" = selecting auth method, "password" = entering password, "identity" = entering key path.
	RemoteAuthStep          string
	RemoteAuthTarget        string
	RemoteAuthError         string
	RemoteAuthUsername      string          // username to use when submitting (default root)
	RemoteAuthUsernameInput textinput.Model // username input in choose step
	RemoteAuthInput         textinput.Model // for password or identity path
	RemoteAuthConnecting    bool            // true while waiting for remote auth result ("Connecting..." state)
	// Path completion (shared): used for any path input with dropdown (auth identity key path, add-remote key path).
	PathCompletionCandidates []string
	PathCompletionIndex      int

	// InitialShowConfigLLM: when true, open Config LLM overlay on first WindowSizeMsg (e.g. no config / model empty at startup).
	InitialShowConfigLLM bool
	// Config LLM overlay: base_url, api_key, model, max_context_messages, max_context_chars.
	ConfigLLMActive           bool
	ConfigLLMChecking         bool // true while async "hello" check is in progress after save
	ConfigLLMBaseURLInput     textinput.Model
	ConfigLLMApiKeyInput      textinput.Model
	ConfigLLMModelInput       textinput.Model
	ConfigLLMMaxMessagesInput textinput.Model
	ConfigLLMMaxCharsInput    textinput.Model
	ConfigLLMFieldIndex       int // 0=base_url, 1=api_key, 2=model, 3=max_messages, 4=max_chars
	ConfigLLMError            string

	// Add-skill overlay: URL (required), ref, path and local name (optional).
	AddSkillActive         bool
	AddSkillURLInput       textinput.Model
	AddSkillRefInput       textinput.Model
	AddSkillPathInput      textinput.Model
	AddSkillNameInput      textinput.Model
	AddSkillFieldIndex     int // 0=url, 1=ref, 2=path, 3=name
	AddSkillError          string
	AddSkillRefsFullList   []string // all refs from remote (for filtering)
	AddSkillRefCandidates  []string // refs filtered by Ref input prefix
	AddSkillRefIndex       int      // selection in ref dropdown
	AddSkillPathsFullList  []string // paths from git repo (when non-nil, Path dropdown uses this instead of static list)
	AddSkillPathCandidates []string // path options filtered by Path input prefix
	AddSkillPathIndex      int      // selection in path dropdown

	// Update-skill overlay: choose ref and confirm update for an installed skill.
	UpdateSkillActive        bool
	UpdateSkillName          string
	UpdateSkillURL           string
	UpdateSkillPath          string
	UpdateSkillCurrentCommit string
	UpdateSkillRefs          []string
	UpdateSkillRefIndex      int
	UpdateSkillLatestCommit  string
	UpdateSkillError         string
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tea.BlurMsg:
		return m.handleBlurMsg()
	case tea.FocusMsg:
		return m.handleFocusMsg()
	case RemoteStatusMsg:
		return m.handleRemoteStatusMsg(msg)
	case RunCompletionCacheMsg:
		return m.handleRunCompletionCacheMsg(msg)
	case ConfigLLMCheckDoneMsg:
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
	case AddSkillRefsLoadedMsg:
		if m.AddSkillActive {
			m.AddSkillRefsFullList = msg.Refs
			m.AddSkillRefCandidates = filterByPrefix(msg.Refs, m.AddSkillRefInput.Value())
			m.AddSkillRefIndex = 0
		}
		return m, nil
	case AddSkillPathsLoadedMsg:
		if m.AddSkillActive {
			m.AddSkillPathsFullList = msg.Paths
			m = m.updateAddSkillPathCandidates()
		}
		return m, nil
	case RemoteConnectDoneMsg:
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
	case RemoteAuthPromptMsg:
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
	case OverlayShowMsg:
		m.OverlayActive = true
		m.OverlayTitle = msg.Title
		m.OverlayContent = msg.Content
		m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
		m.OverlayViewport.SetContent(m.OverlayContent)
		return m, nil
	case OverlayCloseMsg:
		return m.closeOverlayCommon(false)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case ApprovalRequestMsg:
		// When an approval is requested, immediately refresh the viewport so the
		// approval card becomes visible, and scroll to bottom.
		m.Pending = msg
		m.ChoiceIndex = 0
		m.syncInputPlaceholder()
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case SensitiveConfirmationRequestMsg:
		// Same as approval: ensure the sensitive confirmation card is visible.
		m.PendingSensitive = msg
		m.ChoiceIndex = 0
		m.syncInputPlaceholder()
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case SessionSwitchedMsg:
		return m.handleSessionSwitchedMsg(msg)

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
		CurrentSessionPath:         initialSessionPath,
		GetAllowlistAutoRun:        getAllowlistAutoRun,
		InitialShowConfigLLM:       initialShowConfigLLM,
		Width:                      defaultWidth,
		Height:                     defaultHeight,
	}
}
