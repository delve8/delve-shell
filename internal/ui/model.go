package ui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/remotesvc"
	"delve-shell/internal/service/skillsvc"
	"delve-shell/internal/skills"
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
		key := msg.String()

		// Always allow ctrl+c to quit, even during pending approvals or sensitive prompts.
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		state := m.currentUIState()
		if state == uiStatePendingSensitive || state == uiStatePendingApproval {
			if handledModel, handled := m.handlePendingChoiceKey(key); handled {
				return handledModel, nil
			}
		}

		// Slash dropdown navigation should work even if other key paths evolve.
		// Handle it before overlay/key-to-input processing so Up/Down/Enter remain reliable.
		var inputVal string
		var inSlash bool
		inputVal = m.Input.Value()
		inSlash = strings.HasPrefix(inputVal, "/")
		if inSlash {
			if m2, cmd, handled := m.handleSlashNavigationKey(key, msg, inputVal); handled {
				return m2, cmd
			}
			if key == "enter" {
				if m2, cmd, handled := m.handleSlashEnterKey(inputVal); handled {
					return m2, cmd
				}
			}
		}

		// Overlay key handling: Esc closes, Enter submits, other keys go to overlay input.
		if m.currentUIState() == uiStateOverlay {
			switch key {
			case "esc":
				return m.closeOverlayCommon(true)
			default:
				// Add-skill overlay: URL, ref, path.
				if m2, cmd, handled := m.handleAddSkillOverlayKey(key, msg); handled {
					return m2, cmd
				}
				// Add-remote overlay: form with 5 fields (host, username, name, key path, save-as-remote checkbox).
				if m.AddRemoteActive {
					switch key {
					case "tab":
						// Tab only selects path candidate when list is visible; no longer used to move between fields.
						if m.AddRemoteFieldIndex == 3 {
							cands := m.PathCompletionCandidates
							if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
								chosen := cands[m.PathCompletionIndex]
								m.AddRemoteKeyInput.SetValue(chosen)
								m.AddRemoteKeyInput.CursorEnd()
								if strings.HasSuffix(chosen, "/") {
									m.PathCompletionCandidates = PathCandidates(chosen)
									m.PathCompletionIndex = 0
								} else {
									m.PathCompletionCandidates = nil
									m.PathCompletionIndex = -1
								}
								return m, nil
							}
						}
					case "up", "down":
						// In Key path with completion list: move within list. Else: Up/Down move focus between fields.
						if m.AddRemoteFieldIndex == 3 && len(m.PathCompletionCandidates) > 0 {
							cands := m.PathCompletionCandidates
							if key == "up" {
								m.PathCompletionIndex--
								if m.PathCompletionIndex < 0 {
									m.PathCompletionIndex = len(cands) - 1
								}
								return m, nil
							}
							if key == "down" {
								m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(cands)
								return m, nil
							}
						}
						dir := 1
						if key == "up" {
							dir = -1
						}
						// Field count: 4 for /config add-remote, 5 (with save checkbox) for /remote on.
						fieldCount := 4
						if m.AddRemoteConnect {
							fieldCount = 5
						}
						m.AddRemoteFieldIndex = (m.AddRemoteFieldIndex + dir + fieldCount) % fieldCount
						m.AddRemoteUserInput.Blur()
						m.AddRemoteHostInput.Blur()
						m.AddRemoteNameInput.Blur()
						m.AddRemoteKeyInput.Blur()
						switch m.AddRemoteFieldIndex {
						case 0:
							m.AddRemoteHostInput.Focus()
						case 1:
							m.AddRemoteUserInput.Focus()
						case 2:
							m.AddRemoteNameInput.Focus()
						case 3:
							m.AddRemoteKeyInput.Focus()
						case 4:
							// Save checkbox: no textinput to focus.
						}
						if m.AddRemoteFieldIndex != 3 {
							m.PathCompletionCandidates = nil
							m.PathCompletionIndex = -1
						} else {
							m.PathCompletionCandidates = PathCandidates(m.AddRemoteKeyInput.Value())
							if len(m.PathCompletionCandidates) > 0 {
								m.PathCompletionIndex = 0
							} else {
								m.PathCompletionIndex = -1
							}
						}
						return m, nil
					case "y", "Y":
						if m.AddRemoteOfferOverwrite {
							host := strings.TrimSpace(m.AddRemoteHostInput.Value())
							user := strings.TrimSpace(m.AddRemoteUserInput.Value())
							if user == "" {
								user = "root"
							}
							name := strings.TrimSpace(m.AddRemoteNameInput.Value())
							keyPath := strings.TrimSpace(m.AddRemoteKeyInput.Value())
							if host == "" {
								return m, nil
							}
							target := user + "@" + host
							if err := remotesvc.Update(target, name, keyPath); err != nil {
								m.AddRemoteError = err.Error()
								m.AddRemoteOfferOverwrite = false
								return m, nil
							}
							display := host
							if name != "" {
								display = name + " (" + host + ")"
							}
							lang := m.getLang()
							m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display))))
							m.Messages = append(m.Messages, "")
							m.Viewport.SetContent(m.buildContent())
							m.Viewport.GotoBottom()
							m.OverlayActive = false
							m.AddRemoteActive = false
							m.AddRemoteError = ""
							m.AddRemoteOfferOverwrite = false
							m.OverlayTitle = ""
							m.OverlayContent = ""
							// After closing Add Remote overlay (overwrite), refocus main input.
							m.Input.Focus()
							if m.ConfigUpdatedChan != nil {
								select {
								case m.ConfigUpdatedChan <- struct{}{}:
								default:
								}
							}
							return m, nil
						}
					case " ":
						// Space toggles save-as-remote only when focused on the checkbox field.
						if m.AddRemoteFieldIndex == 4 {
							m.AddRemoteSave = !m.AddRemoteSave
							return m, nil
						}
					case "enter":
						if m.AddRemoteFieldIndex == 3 {
							cands := m.PathCompletionCandidates
							if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
								chosen := cands[m.PathCompletionIndex]
								m.AddRemoteKeyInput.SetValue(chosen)
								m.AddRemoteKeyInput.CursorEnd()
								if strings.HasSuffix(chosen, "/") {
									m.PathCompletionCandidates = PathCandidates(chosen)
									m.PathCompletionIndex = 0
								} else {
									m.PathCompletionCandidates = nil
									m.PathCompletionIndex = -1
								}
								return m, nil
							}
						}
						host := strings.TrimSpace(m.AddRemoteHostInput.Value())
						user := strings.TrimSpace(m.AddRemoteUserInput.Value())
						if user == "" {
							user = "root"
						}
						name := strings.TrimSpace(m.AddRemoteNameInput.Value())
						keyPath := strings.TrimSpace(m.AddRemoteKeyInput.Value())
						if host == "" {
							m.AddRemoteError = "host is required"
							return m, nil
						}
						if strings.Contains(host, "@") {
							m.AddRemoteError = "host must not contain @"
							return m, nil
						}
						target := user + "@" + host
						// Optionally save/update remote config when requested.
						if m.AddRemoteSave {
							if err := remotesvc.Add(target, name, keyPath); err != nil {
								m.AddRemoteError = err.Error()
								m.AddRemoteOfferOverwrite = strings.Contains(err.Error(), "already exists")
								return m, nil
							}
							m.AddRemoteOfferOverwrite = false
							display := host
							if name != "" {
								display = name + " (" + host + ")"
							}
							lang := m.getLang()
							m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display))))
							m.Messages = append(m.Messages, "")
							if m.ConfigUpdatedChan != nil {
								select {
								case m.ConfigUpdatedChan <- struct{}{}:
								default:
								}
							}
						}
						m.Viewport.SetContent(m.buildContent())
						m.Viewport.GotoBottom()
						if m.AddRemoteConnect && m.RemoteOnChan != nil {
							// Show "Connecting..." and wait for RemoteConnectDoneMsg; close overlay only on success.
							m.AddRemoteConnecting = true
							m.AddRemoteError = ""
							select {
							case m.RemoteOnChan <- target:
							default:
								m.AddRemoteConnecting = false
							}
							return m, nil
						}
						m.OverlayActive = false
						m.AddRemoteActive = false
						m.AddRemoteError = ""
						m.AddRemoteOfferOverwrite = false
						m.OverlayTitle = ""
						m.OverlayContent = ""
						m.Input.Focus()
						return m, nil
					}
					var cmd tea.Cmd
					switch m.AddRemoteFieldIndex {
					case 0:
						m.AddRemoteHostInput, cmd = m.AddRemoteHostInput.Update(msg)
					case 1:
						m.AddRemoteUserInput, cmd = m.AddRemoteUserInput.Update(msg)
					case 2:
						m.AddRemoteNameInput, cmd = m.AddRemoteNameInput.Update(msg)
					case 3:
						m.AddRemoteKeyInput, cmd = m.AddRemoteKeyInput.Update(msg)
						m.PathCompletionCandidates = PathCandidates(m.AddRemoteKeyInput.Value())
						if len(m.PathCompletionCandidates) > 0 {
							m.PathCompletionIndex = 0
						} else {
							m.PathCompletionIndex = -1
						}
					case 4:
						// Save checkbox has no text input; ignore character keys here.
						cmd = nil
					}
					return m, cmd
				}
				if m.ConfigLLMActive {
					const configLLMFieldCount = 5
					switch key {
					case "up", "down":
						dir := 1
						if key == "up" {
							dir = -1
						}
						m.ConfigLLMFieldIndex = (m.ConfigLLMFieldIndex + dir + configLLMFieldCount) % configLLMFieldCount
						m.ConfigLLMBaseURLInput.Blur()
						m.ConfigLLMApiKeyInput.Blur()
						m.ConfigLLMModelInput.Blur()
						m.ConfigLLMMaxMessagesInput.Blur()
						m.ConfigLLMMaxCharsInput.Blur()
						switch m.ConfigLLMFieldIndex {
						case 0:
							m.ConfigLLMBaseURLInput.Focus()
						case 1:
							m.ConfigLLMApiKeyInput.Focus()
						case 2:
							m.ConfigLLMModelInput.Focus()
						case 3:
							m.ConfigLLMMaxMessagesInput.Focus()
						case 4:
							m.ConfigLLMMaxCharsInput.Focus()
						}
						return m, nil
					case "enter":
						if m.ConfigLLMChecking {
							return m, nil
						}
						baseURL := strings.TrimSpace(m.ConfigLLMBaseURLInput.Value())
						apiKey := strings.TrimSpace(m.ConfigLLMApiKeyInput.Value())
						model := strings.TrimSpace(m.ConfigLLMModelInput.Value())
						maxMessagesStr := strings.TrimSpace(m.ConfigLLMMaxMessagesInput.Value())
						maxCharsStr := strings.TrimSpace(m.ConfigLLMMaxCharsInput.Value())
						if model == "" {
							m.ConfigLLMError = i18n.T(m.getLang(), i18n.KeyConfigLLMModelRequired)
							return m, nil
						}
						m = m.applyConfigLLMFromOverlayStart(baseURL, apiKey, model, maxMessagesStr, maxCharsStr)
						if !m.ConfigLLMChecking {
							return m, nil
						}
						return m, RunConfigLLMCheckCmd()
					}
					var cmd tea.Cmd
					switch m.ConfigLLMFieldIndex {
					case 0:
						m.ConfigLLMBaseURLInput, cmd = m.ConfigLLMBaseURLInput.Update(msg)
					case 1:
						m.ConfigLLMApiKeyInput, cmd = m.ConfigLLMApiKeyInput.Update(msg)
					case 2:
						m.ConfigLLMModelInput, cmd = m.ConfigLLMModelInput.Update(msg)
					case 3:
						m.ConfigLLMMaxMessagesInput, cmd = m.ConfigLLMMaxMessagesInput.Update(msg)
					case 4:
						m.ConfigLLMMaxCharsInput, cmd = m.ConfigLLMMaxCharsInput.Update(msg)
					}
					return m, cmd
				}
				// Update-skill overlay: choose ref and confirm update.
				if m2, cmd, handled := m.handleUpdateSkillOverlayKey(key); handled {
					return m2, cmd
				}
				// Remote auth: step "username" → "choose" (1/2) → "password" or "identity".
				switch m.RemoteAuthStep {
				case "auto_identity":
					// Automatic connection with configured identity file: no interactive input; Esc handled above.
					return m, nil
				case "username":
					if key == "enter" {
						m.RemoteAuthUsername = strings.TrimSpace(m.RemoteAuthUsernameInput.Value())
						if m.RemoteAuthUsername == "" {
							m.RemoteAuthUsername = "root"
						}
						m.RemoteAuthStep = "choose"
						return m, nil
					}
					var cmd tea.Cmd
					m.RemoteAuthUsernameInput, cmd = m.RemoteAuthUsernameInput.Update(msg)
					return m, cmd
				case "choose":
					switch key {
					case "1":
						m.RemoteAuthStep = "password"
						m.RemoteAuthInput = textinput.New()
						m.RemoteAuthInput.Placeholder = "SSH password"
						m.RemoteAuthInput.EchoMode = textinput.EchoPassword
						m.RemoteAuthInput.Focus()
						var b strings.Builder
						if m.RemoteAuthError != "" {
							b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
						}
						b.WriteString("SSH password for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
						b.WriteString("Press Enter to submit, Esc to cancel.")
						m.OverlayContent = b.String()
						return m, nil
					case "2":
						m.RemoteAuthStep = "identity"
						m.RemoteAuthInput = textinput.New()
						m.RemoteAuthInput.Placeholder = "~/.ssh/id_rsa"
						m.RemoteAuthInput.EchoMode = textinput.EchoNormal
						m.RemoteAuthInput.Focus()
						m.PathCompletionCandidates = nil
						m.PathCompletionIndex = -1
						var b strings.Builder
						if m.RemoteAuthError != "" {
							b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
						}
						b.WriteString("SSH key file path for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
						b.WriteString("Press Enter to submit, Esc to cancel.")
						m.OverlayContent = b.String()
						return m, nil
					}
					return m, nil
				case "password":
					// When waiting for auth result, ignore further input except Esc (handled above).
					if m.RemoteAuthConnecting {
						return m, nil
					}
					if key == "enter" {
						input := m.RemoteAuthInput.Value()
						if input == "" {
							m.RemoteAuthStep = "choose"
							m.ChoiceIndex = 0
							var b strings.Builder
							if m.RemoteAuthError != "" {
								b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
							}
							b.WriteString("Choose authentication method:\n")
							b.WriteString("  1. Password\n")
							b.WriteString("  2. Key file (identity file)\n\n")
							b.WriteString("Press 1 or 2 to select, Esc to cancel.")
							m.OverlayContent = b.String()
							return m, nil
						}
						// Non-empty password: show connecting state and send credentials; overlay stays open until auth result.
						m.RemoteAuthConnecting = true
						var b strings.Builder
						if m.RemoteAuthError != "" {
							b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
						}
						b.WriteString("SSH password for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
						b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
						b.WriteString("Press Esc to cancel.")
						m.OverlayContent = b.String()
						if m.RemoteAuthRespChan != nil {
							select {
							case m.RemoteAuthRespChan <- RemoteAuthResponse{
								Target:   m.RemoteAuthTarget,
								Username: m.RemoteAuthUsername,
								Kind:     m.RemoteAuthStep,
								Password: input,
							}:
							default:
							}
						}
						return m, nil
					}
					var cmd tea.Cmd
					m.RemoteAuthInput, cmd = m.RemoteAuthInput.Update(msg)
					return m, cmd
				case "identity":
					// When waiting for auth result, ignore further input except Esc (handled above).
					if m.RemoteAuthConnecting {
						return m, nil
					}
					// Path completion: Up/Down to move, Enter or Tab to pick candidate (Tab matches bash habit), or submit with Enter.
					cands := m.PathCompletionCandidates
					if key == "up" && len(cands) > 0 {
						m.PathCompletionIndex--
						if m.PathCompletionIndex < 0 {
							m.PathCompletionIndex = len(cands) - 1
						}
						return m, nil
					}
					if key == "down" && len(cands) > 0 {
						m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(cands)
						return m, nil
					}
					pickIdentityCandidate := len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) && (key == "enter" || key == "tab")
					if pickIdentityCandidate {
						chosen := cands[m.PathCompletionIndex]
						m.RemoteAuthInput.SetValue(chosen)
						m.RemoteAuthInput.CursorEnd()
						if strings.HasSuffix(chosen, "/") {
							m.PathCompletionCandidates = PathCandidates(chosen)
							m.PathCompletionIndex = 0
						} else {
							m.PathCompletionCandidates = nil
							m.PathCompletionIndex = -1
						}
						return m, nil
					}
					if key == "enter" {
						input := m.RemoteAuthInput.Value()
						if input == "" {
							m.RemoteAuthStep = "choose"
							m.ChoiceIndex = 0
							m.PathCompletionCandidates = nil
							m.PathCompletionIndex = -1
							var b strings.Builder
							if m.RemoteAuthError != "" {
								b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
							}
							b.WriteString("Choose authentication method:\n")
							b.WriteString("  1. Password\n")
							b.WriteString("  2. Key file (identity file)\n\n")
							b.WriteString("Press 1 or 2 to select, Esc to cancel.")
							m.OverlayContent = b.String()
							return m, nil
						}
						// Non-empty key path: show connecting state and send credentials; overlay stays open until auth result.
						m.RemoteAuthConnecting = true
						var b strings.Builder
						if m.RemoteAuthError != "" {
							b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
						}
						b.WriteString("SSH key file path for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
						b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
						b.WriteString("Press Esc to cancel.")
						m.OverlayContent = b.String()
						if m.RemoteAuthRespChan != nil {
							select {
							case m.RemoteAuthRespChan <- RemoteAuthResponse{
								Target:   m.RemoteAuthTarget,
								Username: m.RemoteAuthUsername,
								Kind:     m.RemoteAuthStep,
								Password: input,
							}:
							default:
							}
						}
						return m, nil
					}
					if key == "tab" {
						// No candidate selected: refresh list (Tab already handled pick above when candidates exist).
						m.PathCompletionCandidates = PathCandidates(m.RemoteAuthInput.Value())
						if len(m.PathCompletionCandidates) > 0 {
							m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(m.PathCompletionCandidates)
						} else {
							m.PathCompletionIndex = -1
						}
						return m, nil
					}
					var cmd tea.Cmd
					m.RemoteAuthInput, cmd = m.RemoteAuthInput.Update(msg)
					// Refresh path candidates from new input (so dropdown updates as user types).
					m.PathCompletionCandidates = PathCandidates(m.RemoteAuthInput.Value())
					if len(m.PathCompletionCandidates) > 0 {
						m.PathCompletionIndex = 0
					} else {
						m.PathCompletionIndex = -1
					}
					return m, cmd
				}
				// Generic overlay: pass up/down/pgup/pgdown.
				var cmd tea.Cmd
				m.OverlayViewport, cmd = m.OverlayViewport.Update(msg)
				return m, cmd
			}
		}

		inputVal = m.Input.Value()
		inSlash = strings.HasPrefix(inputVal, "/")

		// scroll keys: Up/Down change selection in slash mode, else go to viewport with PgUp/PgDown
		if key == "up" || key == "down" || key == "pgup" || key == "pgdown" {
			if inSlash && (key == "up" || key == "down") {
				opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
				vis := visibleSlashOptions(inputVal, opts)
				if len(vis) > 0 {
					if m.SlashSuggestIndex >= len(vis) {
						m.SlashSuggestIndex = 0
					}
					if key == "down" {
						m.SlashSuggestIndex = (m.SlashSuggestIndex + 1) % len(vis)
					} else {
						m.SlashSuggestIndex = (m.SlashSuggestIndex - 1 + len(vis)) % len(vis)
					}
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.Viewport, cmd = m.Viewport.Update(msg)
			return m, cmd
		}

		if key == "enter" {
			text := strings.TrimSpace(inputVal)
			if text == "" {
				return m, nil
			}
			// WaitingForAI only blocks submitting new messages; slash commands starting with / always run
			if m.WaitingForAI && !strings.HasPrefix(text, "/") {
				return m, nil
			}
			// Save selected slash option before any state change; Enter handler resets SlashSuggestIndex below, so we must capture now.
			// Use inputVal (not text) for slash options so we match what the view shows and get correct opts/vis when user has trailing space.
			var slashSelectedPath string
			var slashSelectedIndex int = -1
			if strings.HasPrefix(inputVal, "/") {
				opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
				vis := visibleSlashOptions(inputVal, opts)
				if len(vis) > 0 && m.SlashSuggestIndex < len(vis) {
					chosen := opts[vis[m.SlashSuggestIndex]].Cmd
					// chosen != text => fill selection only, do not execute or add to View
					if (chosen == text || strings.HasPrefix(chosen, text)) && chosen != text {
						m.Input.SetValue(slashChosenToInputValue(chosen))
						m.Input.CursorEnd()
						m.SlashSuggestIndex = 0 // reset so next Enter (new opts set, e.g. skill list) uses index 0
						return m, nil
					}
					slashSelectedIndex = m.SlashSuggestIndex
					if opts[vis[m.SlashSuggestIndex]].Path != "" {
						slashSelectedPath = opts[vis[m.SlashSuggestIndex]].Path
					}
				}
			}
			// /new sends to run loop only; do not append to Messages
			if text == "/new" {
				if m.SubmitChan != nil {
					m.SubmitChan <- text
				}
				m.Input.SetValue("")
				m.Input.CursorEnd()
				m.SlashSuggestIndex = 0
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			}

			userLine := i18n.T(m.getLang(), i18n.KeyUserLabel) + text
			w := m.Width
			if w <= 0 {
				w = 80
			}
			sepW := w
			sepLine := separatorStyle.Render(strings.Repeat("─", sepW))
			if len(m.Messages) > 0 && m.Messages[len(m.Messages)-1] != sepLine {
				m.Messages = append(m.Messages, sepLine)
			}
			m.Messages = append(m.Messages, wrapString(userLine, w))
			m.Messages = append(m.Messages, "") // blank line before command or AI reply
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0
			switch text {
			case "/help", "/config llm", "/config add-skill", "/config add-remote", "/remote on", "/remote off", "/config update auto-run list", "/config reload", "/reload":
				if m2, cmd, handled := m.dispatchSlashExact(text); handled {
					return m2, cmd
				}
			}
			if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
				return m2, cmd
			}

			switch {
			case text == "/q":
				return m, tea.Quit
			case text == "/sh":
				if m.ShellRequestedChan != nil {
					msgs := make([]string, len(m.Messages))
					copy(msgs, m.Messages)
					select {
					case m.ShellRequestedChan <- msgs:
					default:
					}
				}
				return m, tea.Quit
			case text == "/cancel":
				if m.WaitingForAI && m.CancelRequestChan != nil {
					select {
					case m.CancelRequestChan <- struct{}{}:
					default:
					}
					m.WaitingForAI = false
				} else {
					lang := m.getLang()
					m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyNoRequestInProgress))))
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
				}
				return m, nil
			case strings.HasPrefix(text, "/config llm base_url "):
				m = m.applyConfigLLM("base_url", strings.TrimPrefix(text, "/config llm base_url "))
				return m, nil
			case strings.HasPrefix(text, "/config llm api_key "):
				m = m.applyConfigLLM("api_key", strings.TrimPrefix(text, "/config llm api_key "))
				return m, nil
			case strings.HasPrefix(text, "/config llm model "):
				m = m.applyConfigLLM("model", strings.TrimPrefix(text, "/config llm model "))
				return m, nil
			case text == "/config show", text == "/config":
				m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			case text == "/config auto-run list-only":
				m = m.applyConfigAllowlistAutoRun("list-only")
				return m, nil
			case text == "/config auto-run disable":
				m = m.applyConfigAllowlistAutoRun("disable")
				return m, nil
			case strings.HasPrefix(text, "/config add-remote "):
				m = m.applyConfigAddRemote(strings.TrimPrefix(text, "/config add-remote "))
				return m, nil
			case strings.HasPrefix(text, "/config del-remote "):
				m = m.applyConfigRemoveRemote(strings.TrimSpace(strings.TrimPrefix(text, "/config del-remote ")))
				return m, nil
			case strings.HasPrefix(text, "/config auto-run "):
				arg := strings.TrimSpace(strings.TrimPrefix(text, "/config auto-run "))
				m = m.applyConfigAllowlistAutoRun(arg)
				return m, nil
			case strings.HasPrefix(text, "/remote on "):
				target := strings.TrimSpace(strings.TrimPrefix(text, "/remote on "))
				if target != "" && m.RemoteOnChan != nil {
					select {
					case m.RemoteOnChan <- target:
					default:
					}
				}
				m.Input.SetValue("")
				m.Input.CursorEnd()
				return m, nil
			case strings.HasPrefix(text, "/config add-skill"):
				rest := strings.TrimSpace(text[len("/config add-skill"):])
				url, ref, path := "", "", ""
				if rest != "" {
					fields := strings.Fields(rest)
					if len(fields) >= 1 {
						url = fields[0]
					}
					if len(fields) >= 2 {
						if strings.Contains(fields[1], "/") {
							path = fields[1]
						} else {
							ref = fields[1]
						}
					}
					if len(fields) >= 3 {
						ref = fields[1]
						path = fields[2]
					}
				}
				m = m.openAddSkillOverlay(url, ref, path)
				m.Input.SetValue("")
				m.Input.CursorEnd()
				return m, nil
			case strings.HasPrefix(text, "/config del-skill "):
				rest := strings.TrimSpace(text[len("/config del-skill "):])
				fields := strings.Fields(rest)
				if len(fields) == 0 {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkillRemove))))
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
					return m, nil
				}
				skillNameToRemove := fields[0]
				if err := skillsvc.Remove(skillNameToRemove); err != nil {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoveFailed, err))))
				} else {
					m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillRemoved, skillNameToRemove))))
				}
				m.Input.SetValue("")
				m.Input.CursorEnd()
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			case strings.HasPrefix(text, "/config update-skill"):
				rest := strings.TrimSpace(text[len("/config update-skill"):])
				fields := strings.Fields(rest)
				if len(fields) == 0 {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyDescConfigUpdateSkill))))
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
					return m, nil
				}
				skillName := fields[0]
				m = m.openUpdateSkillOverlay(skillName)
				m.Input.SetValue("")
				m.Input.CursorEnd()
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			case strings.HasPrefix(text, "/skill "):
				rest := strings.TrimSpace(text[len("/skill "):])
				fields := strings.Fields(rest)
				if len(fields) < 1 {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkill))))
					return m, nil
				}
				skillName := fields[0]
				naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
				if naturalLanguage == "" {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUsageSkill))))
					return m, nil
				}
				skillDir := skills.SkillDir(skillName)
				if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeySkillNotFound))))
					return m, nil
				}
				skillContent, err := skills.ReadSKILLContent(skillDir)
				if err != nil {
					m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(m.getLang(), i18n.KeySkillInstallFailed, err))))
					return m, nil
				}
				payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
				if m.SubmitChan != nil {
					m.SubmitChan <- payload
					m.WaitingForAI = true
				}
				m.Input.SetValue("")
				m.Input.CursorEnd()
				return m, nil
			case strings.HasPrefix(text, "/config update-skill "):
				name := strings.TrimSpace(strings.TrimPrefix(text, "/config update-skill "))
				if fields := strings.Fields(name); len(fields) > 0 {
					name = fields[0]
				}
				if name == "" {
					return m, nil
				}
				m = m.openUpdateSkillOverlay(name)
				m.Input.SetValue("")
				m.Input.CursorEnd()
				m.SlashSuggestIndex = 0
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			case strings.HasPrefix(text, "/"):
				// Use path captured before SlashSuggestIndex was reset; otherwise we would always send opts[0].
				if slashSelectedPath != "" {
					if m.SessionSwitchChan != nil {
						select {
						case m.SessionSwitchChan <- slashSelectedPath:
						default:
						}
					}
					m.Input.SetValue("")
					m.Input.CursorEnd()
					m.SlashSuggestIndex = 0
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
					return m, nil
				}
				opts := getSlashOptionsForInput(text, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
				vis := visibleSlashOptions(text, opts)
				var selectedOpt slashOption
				if slashSelectedIndex >= 0 && slashSelectedIndex < len(vis) {
					selectedOpt = opts[vis[slashSelectedIndex]]
				}
				// Sessions list empty: show message only when the single option is the session-none placeholder (not for del-skill etc).
				sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
				if selectedOpt.Path == "" && len(vis) == 1 && selectedOpt.Cmd == sessionNoneMsg {
					m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
					m.Input.SetValue("")
					m.Input.CursorEnd()
					m.SlashSuggestIndex = 0
					return m, nil
				}
				chosen := selectedOpt.Cmd
				// input must match chosen command; skip when only "/". "Fill only" already returned above.
				if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && (chosen == text || strings.HasPrefix(chosen, text)) {
					// user input matches chosen (full input then Enter) => execute
					if m2, cmd, handled := m.dispatchSlashExact(chosen); handled {
						return m2, cmd
					}
					if m2, cmd, handled := m.dispatchSlashPrefix(chosen); handled {
						return m2, cmd
					}
					if chosen == "/run <cmd>" {
						m.Input.SetValue("/run ")
						m.Input.CursorEnd()
						return m, nil
					}
					if chosen == "/config add-skill" {
						m = m.openAddSkillOverlay("", "", "")
						m.Input.SetValue("")
						m.Input.CursorEnd()
						m.SlashSuggestIndex = 0
						return m, nil
					}
					if chosen == "/config auto-run list-only" {
						m = m.applyConfigAllowlistAutoRun("list-only")
						return m, nil
					}
					if chosen == "/config auto-run disable" {
						m = m.applyConfigAllowlistAutoRun("disable")
						return m, nil
					}
					if chosen == "/config llm" {
						m = m.openConfigLLMOverlay()
						return m, nil
					}
					if chosen == "/new" {
						if m.SubmitChan != nil {
							m.SubmitChan <- "/new"
						}
						m.Input.SetValue("")
						m.Input.CursorEnd()
						m.SlashSuggestIndex = 0
						m.Viewport.SetContent(m.buildContent())
						m.Viewport.GotoBottom()
						return m, nil
					}
					if m2, cmd, handled := m.handleSlashSelectedFallback(chosen); handled {
						return m2, cmd
					}
				}
				m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUnknownCmd))))
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			}
			if m.SubmitChan != nil {
				m.SubmitChan <- text
				m.WaitingForAI = true
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.Input, cmd = m.Input.Update(msg)
		if strings.HasPrefix(m.Input.Value(), "/") {
			inputVal := m.Input.Value()
			opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
			vis := visibleSlashOptions(inputVal, opts)
			// Session list (Path set): do not reset index on every keystroke so user can pick another session with Enter
			if len(opts) == 0 || opts[0].Path == "" {
				m.SlashSuggestIndex = 0
			}
			if len(vis) > 0 && m.SlashSuggestIndex >= len(vis) {
				m.SlashSuggestIndex = 0
			}
		}
		return m, cmd

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
