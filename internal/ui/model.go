package ui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/history"
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
	Input               textinput.Model
	Viewport            viewport.Model
	Messages            []string
	Pending             *agent.ApprovalRequest
	PendingSensitive    *agent.SensitiveConfirmationRequest
	SubmitChan          chan<- string
	ExecDirectChan      chan<- string
	ShellRequestedChan  chan<- []string // on /sh send current Messages to preserve after return
	CancelRequestChan   chan<- struct{}  // on /cancel request cancel of in-flight AI
	ConfigUpdatedChan   chan<- struct{}  // on /config save or /reload, invalidate runner so next message reloads config/allowlist
	AllowlistAutoRunChangeChan chan<- bool // runtime toggle for allowlist auto-run (true = list only, false = none)
	SessionSwitchChan          chan<- string // on /sessions choice, send selected session path to continue
	RemoteOnChan               chan<- string  // on /remote on <target>, send resolved target/name to CLI
	RemoteOffChan              chan<- struct{} // on /remote off, switch back to local
	RemoteAuthRespChan         chan<- RemoteAuthResponse // on remote password entry, send credentials back to CLI
	CurrentSessionPath         string       // path of current session (excluded from /sessions list so switch loads another)
	GetAllowlistAutoRun        func() bool  // for header and Pending card 2 vs 3 options
	RemoteActive        bool   // whether commands run on a remote executor
	RemoteLabel         string // label for remote in header, e.g. "dev (root@1.2.3.4)" or "user@host"
	Width               int
	Height              int
	SlashSuggestIndex   int  // 0..len(visible)-1 when input starts with /
	ChoiceIndex         int  // 0-based selection when in Pending/PendingSensitive/PendingSuggested; Up/Down to move, Enter to confirm
	WaitingForAI        bool // when true only blocks submitting new messages (Enter); /xxx slash commands always allowed

	// Overlay state: when OverlayActive is true, a modal is rendered on top of the main UI.
	OverlayActive   bool
	OverlayTitle    string
	OverlayContent  string
	OverlayViewport viewport.Model

	// Add-remote overlay state (username + host separate).
	AddRemoteActive      bool
	AddRemoteUserInput   textinput.Model
	AddRemoteHostInput   textinput.Model
	AddRemoteNameInput   textinput.Model
	AddRemoteKeyInput    textinput.Model
	AddRemoteFieldIndex  int
	AddRemoteError       string
	AddRemoteOfferOverwrite bool // when true, error was "already exists"; show overwrite hint and accept O to overwrite

	// Remote auth overlay state.
	// RemoteAuthStep: "" = inactive, "choose" = selecting auth method, "password" = entering password, "identity" = entering key path.
	RemoteAuthStep        string
	RemoteAuthTarget      string
	RemoteAuthError       string
	RemoteAuthUsername    string       // username to use when submitting (default root)
	RemoteAuthUsernameInput textinput.Model // username input in choose step
	RemoteAuthInput         textinput.Model // for password or identity path
	// Path completion (shared): used for any path input with dropdown (auth identity key path, add-remote key path).
	PathCompletionCandidates []string
	PathCompletionIndex       int
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Input.Cursor.BlinkCmd(), tea.WindowSize())
}

// getLang returns the UI language for i18n. Currently UI is English-only.
func (m Model) getLang() string {
	return "en"
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.syncInputPlaceholder()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Height > 4 {
			vh := m.Height - 8 // 2 header + 1 blank + 1 sep + 1 input + 3 choice/hint max
			if vh < 1 {
				vh = 1
			}
			m.Viewport.Width = m.Width
			m.Viewport.Height = vh
		}
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case RemoteStatusMsg:
		m.RemoteActive = msg.Active
		m.RemoteLabel = msg.Label
		m.Viewport.SetContent(m.buildContent())
		return m, nil
	case RemoteAuthPromptMsg:
		m.OverlayActive = true
		m.OverlayTitle = "Remote Auth"
		m.RemoteAuthStep = "username" // first step: username only; Enter then shows "choose" (1/2) so username can contain 1 or 2
		m.RemoteAuthTarget = msg.Target
		m.RemoteAuthError = msg.Err
		m.ChoiceIndex = 0
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
		m.OverlayActive = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		m.AddRemoteActive = false
		m.AddRemoteError = ""
		m.AddRemoteOfferOverwrite = false
		m.RemoteAuthStep = ""
		m.RemoteAuthTarget = ""
		m.RemoteAuthError = ""
		m.RemoteAuthUsername = ""
		return m, nil
	case tea.KeyMsg:
		key := msg.String()

		// Always allow ctrl+c to quit, even during pending approvals or sensitive prompts.
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		// Overlay key handling: Esc closes, Enter submits, other keys go to overlay input.
		if m.OverlayActive {
			switch key {
			case "esc":
				m.OverlayActive = false
				m.AddRemoteActive = false
				m.AddRemoteError = ""
				m.AddRemoteOfferOverwrite = false
				m.OverlayTitle = ""
				m.OverlayContent = ""
				m.RemoteAuthStep = ""
				m.RemoteAuthTarget = ""
				m.RemoteAuthError = ""
				m.RemoteAuthUsername = ""
				return m, nil
			default:
				// Add-remote overlay: form with 4 fields (host first, then username default root, name, key path).
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
						m.AddRemoteFieldIndex = (m.AddRemoteFieldIndex + dir + 4) % 4
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
							if err := config.UpdateRemote(target, name, keyPath); err != nil {
								m.AddRemoteError = err.Error()
								m.AddRemoteOfferOverwrite = false
								return m, nil
							}
							display := host
							if name != "" {
								display = name + " (" + host + ")"
							}
							lang := m.getLang()
							m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)))
							m.Viewport.SetContent(m.buildContent())
							m.Viewport.GotoBottom()
							m.OverlayActive = false
							m.AddRemoteActive = false
							m.AddRemoteError = ""
							m.AddRemoteOfferOverwrite = false
							m.OverlayTitle = ""
							m.OverlayContent = ""
							if m.ConfigUpdatedChan != nil {
								select {
								case m.ConfigUpdatedChan <- struct{}{}:
								default:
								}
							}
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
						if err := config.AddRemote(target, name, keyPath); err != nil {
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
						m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)))
						m.Viewport.SetContent(m.buildContent())
						m.Viewport.GotoBottom()
						m.OverlayActive = false
						m.AddRemoteActive = false
						m.AddRemoteError = ""
						m.AddRemoteOfferOverwrite = false
						m.OverlayTitle = ""
						m.OverlayContent = ""
						if m.ConfigUpdatedChan != nil {
							select {
							case m.ConfigUpdatedChan <- struct{}{}:
							default:
							}
						}
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
					}
					return m, cmd
				}
				// Remote auth: step "username" → "choose" (1/2) → "password" or "identity".
				switch m.RemoteAuthStep {
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
						m.OverlayActive = false
						m.RemoteAuthStep = ""
						m.RemoteAuthTarget = ""
						m.RemoteAuthError = ""
						return m, nil
					}
					var cmd tea.Cmd
					m.RemoteAuthInput, cmd = m.RemoteAuthInput.Update(msg)
					return m, cmd
				case "identity":
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
						m.OverlayActive = false
						m.RemoteAuthStep = ""
						m.RemoteAuthTarget = ""
						m.RemoteAuthError = ""
						m.PathCompletionCandidates = nil
						m.PathCompletionIndex = -1
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

		inChoice := m.Pending != nil || m.PendingSensitive != nil
		if inChoice {
			n := choiceCount(m)
			if n > 0 {
				if key == "enter" {
					// Treat Enter as selecting current option (1-based)
					key = string(rune('1' + m.ChoiceIndex))
				} else if key == "up" || key == "down" {
					if key == "down" {
						m.ChoiceIndex = (m.ChoiceIndex + 1) % n
					} else {
						m.ChoiceIndex = (m.ChoiceIndex - 1 + n) % n
					}
					return m, nil
				}
			}
		}

		if m.PendingSensitive != nil {
			lang := m.getLang()
			switch key {
			case "1":
				// Persist a static summary of the sensitive confirmation card and user's choice.
				m.Messages = append(m.Messages,
					approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
					execStyle.Render(m.PendingSensitive.Command),
					suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice1)),
				)
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				m.PendingSensitive.ResponseCh <- agent.SensitiveRefuse
				m.PendingSensitive = nil
				return m, nil
			case "2":
				m.Messages = append(m.Messages,
					approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
					execStyle.Render(m.PendingSensitive.Command),
					suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice2)),
				)
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				m.PendingSensitive.ResponseCh <- agent.SensitiveRunAndStore
				m.PendingSensitive = nil
				return m, nil
			case "3":
				m.Messages = append(m.Messages,
					approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
					execStyle.Render(m.PendingSensitive.Command),
					suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice3)),
				)
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				m.PendingSensitive.ResponseCh <- agent.SensitiveRunNoStore
				m.PendingSensitive = nil
				return m, nil
			}
			return m, nil
		}
		if m.Pending != nil {
			lang := m.getLang()
			switch key {
			case "1":
				// Persist a static summary of the approval card and user's decision.
				riskLabel := ""
				switch m.Pending.RiskLevel {
				case "read_only":
					riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
				case "low":
					riskLabel = i18n.T(lang, i18n.KeyRiskLow)
				case "high":
					riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
				}
				commandLine := m.Pending.Command
				if riskLabel != "" {
					commandLine = "[" + riskLabel + "] " + commandLine
				}
				m.Messages = append(m.Messages,
					approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)),
					execStyle.Render(commandLine),
					approvalDecisionApprovedStyle.Render(i18n.T(lang, i18n.KeyApprovalDecisionApproved)),
				)
				if m.Pending.Reason != "" {
					m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason))
				}
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()

				m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: true, CopyRequested: false}
				m.Pending = nil
				return m, nil
			case "2":
				riskLabel := ""
				switch m.Pending.RiskLevel {
				case "read_only":
					riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
				case "low":
					riskLabel = i18n.T(lang, i18n.KeyRiskLow)
				case "high":
					riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
				}
				commandLine := m.Pending.Command
				if riskLabel != "" {
					commandLine = "[" + riskLabel + "] " + commandLine
				}
				m.Messages = append(m.Messages,
					approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)),
					execStyle.Render(commandLine),
					approvalDecisionRejectedStyle.Render(i18n.T(lang, i18n.KeyApprovalDecisionRejected)),
				)
				if m.Pending.Reason != "" {
					m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason))
				}
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				threeOptions := m.GetAllowlistAutoRun != nil && !m.GetAllowlistAutoRun()
				if threeOptions {
					// 2 = Copy
					_ = clipboard.WriteAll(m.Pending.Command)
					m.appendSuggestedLine(m.Pending.Command, lang)
					m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopied)))
					m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: true}
				} else {
					m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
					m.WaitingForAI = false
				}
				m.Pending = nil
				return m, nil
			case "3":
				// Only when 3 options: Dismiss
				riskLabel := ""
				switch m.Pending.RiskLevel {
				case "read_only":
					riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
				case "low":
					riskLabel = i18n.T(lang, i18n.KeyRiskLow)
				case "high":
					riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
				}
				commandLine := m.Pending.Command
				if riskLabel != "" {
					commandLine = "[" + riskLabel + "] " + commandLine
				}
				m.Messages = append(m.Messages,
					approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)),
					execStyle.Render(commandLine),
					suggestStyle.Render(i18n.T(lang, i18n.KeyChoiceDismiss)),
				)
				if m.Pending.Reason != "" {
					m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason))
				}
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
				m.Pending = nil
				m.WaitingForAI = false
				return m, nil
			}
			return m, nil
		}

		inputVal := m.Input.Value()
		inSlash := strings.HasPrefix(inputVal, "/")

		// scroll keys: Up/Down change selection in slash mode, else go to viewport with PgUp/PgDown
		if key == "up" || key == "down" || key == "pgup" || key == "pgdown" {
			if inSlash && (key == "up" || key == "down") {
				opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath)
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
			var slashSelectedPath string
			var slashSelectedIndex int = -1
			if strings.HasPrefix(text, "/") {
				opts := getSlashOptionsForInput(text, m.getLang(), m.CurrentSessionPath)
				vis := visibleSlashOptions(text, opts)
				if len(vis) > 0 && m.SlashSuggestIndex < len(vis) {
					chosen := opts[vis[m.SlashSuggestIndex]].Cmd
					// Execute-on-select: chosen is a specific item (remove-remote <name> or remote on <target>); submit it directly.
					if chosen != text && strings.HasPrefix(chosen, text) {
						if strings.HasPrefix(chosen, "/config remove-remote ") && strings.TrimSpace(strings.TrimPrefix(chosen, "/config remove-remote ")) != "" {
							text = chosen
						} else if strings.HasPrefix(chosen, "/remote on ") {
							t := strings.TrimSpace(strings.TrimPrefix(chosen, "/remote on "))
							if t != "" && !strings.Contains(t, "<") {
								text = chosen
							}
						}
					}
					// chosen != text => fill selection only, do not execute or add to View
					if (chosen == text || strings.HasPrefix(chosen, text)) && chosen != text {
						m.Input.SetValue(slashChosenToInputValue(chosen))
						m.Input.CursorEnd()
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

			m.Messages = append(m.Messages, i18n.T(m.getLang(), i18n.KeyUserLabel)+text)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0

			switch {
			case text == "/exit", text == "/q":
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
					m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyNoRequestInProgress)))
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
				}
				return m, nil
			case text == "/help":
				m.OverlayActive = true
				m.OverlayTitle = "Help"
				m.OverlayContent = i18n.T(m.getLang(), i18n.KeyHelpText)
				m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
				m.OverlayViewport.SetContent(m.OverlayContent)
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
				m = m.showConfig()
				return m, nil
			case text == "/config update auto-run list":
				m = m.applyConfigAllowlistUpdate()
				return m, nil
			case text == "/config add-remote":
				m.OverlayActive = true
				m.OverlayTitle = "Add Remote"
				m.AddRemoteActive = true
				m.AddRemoteError = ""
				m.AddRemoteOfferOverwrite = false
				m.PathCompletionCandidates = nil
				m.PathCompletionIndex = -1
				m.AddRemoteFieldIndex = 0
				m.AddRemoteHostInput = textinput.New()
				m.AddRemoteHostInput.Placeholder = "host or host:22"
				m.AddRemoteHostInput.Focus()
				m.AddRemoteUserInput = textinput.New()
				m.AddRemoteUserInput.Placeholder = "e.g. root"
				m.AddRemoteUserInput.SetValue("root")
				m.AddRemoteNameInput = textinput.New()
				m.AddRemoteNameInput.Placeholder = "name (optional)"
				m.AddRemoteKeyInput = textinput.New()
				m.AddRemoteKeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
				return m, nil
			case strings.HasPrefix(text, "/config add-remote "):
				m = m.applyConfigAddRemote(strings.TrimPrefix(text, "/config add-remote "))
				return m, nil
			case strings.HasPrefix(text, "/config remove-remote "):
				m = m.applyConfigRemoveRemote(strings.TrimSpace(strings.TrimPrefix(text, "/config remove-remote ")))
				return m, nil
			case strings.HasPrefix(text, "/config auto-run "):
				arg := strings.TrimSpace(strings.TrimPrefix(text, "/config auto-run "))
				m = m.applyConfigAllowlistAutoRun(arg)
				return m, nil
			case text == "/reload":
				if m.ConfigUpdatedChan != nil {
					select {
					case m.ConfigUpdatedChan <- struct{}{}:
					default:
					}
				}
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
			case text == "/remote off":
				if m.RemoteOffChan != nil {
					select {
					case m.RemoteOffChan <- struct{}{}:
					default:
					}
				}
				m.Input.SetValue("")
				m.Input.CursorEnd()
				return m, nil
			case strings.HasPrefix(text, "/run "):
				cmd := strings.TrimSpace(text[len("/run "):])
				if m.ExecDirectChan != nil && cmd != "" {
					m.ExecDirectChan <- cmd
				} else if cmd == "" {
					m.Messages = append(m.Messages, errStyle.Render(i18n.T(m.getLang(), i18n.KeyUsageRun)))
				}
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
				opts := getSlashOptionsForInput(text, m.getLang(), m.CurrentSessionPath)
				vis := visibleSlashOptions(text, opts)
				var selectedOpt slashOption
				if slashSelectedIndex >= 0 && slashSelectedIndex < len(vis) {
					selectedOpt = opts[vis[slashSelectedIndex]]
				}
				// "No previous sessions" single option: show message and clear input.
				if selectedOpt.Path == "" && len(vis) == 1 && selectedOpt.Desc == "" {
					m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(m.getLang(), i18n.KeySessionNone)))
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
						if chosen == "/exit" || chosen == "/q" {
							return m, tea.Quit
						}
						if chosen == "/sh" {
							if m.ShellRequestedChan != nil {
								msgs := make([]string, len(m.Messages))
								copy(msgs, m.Messages)
								select {
								case m.ShellRequestedChan <- msgs:
								default:
								}
							}
							return m, tea.Quit
						}
						if chosen == "/run <cmd>" {
							m.Input.SetValue("/run ")
							m.Input.CursorEnd()
							return m, nil
						}
						if chosen == "/cancel" {
							if m.WaitingForAI && m.CancelRequestChan != nil {
								select {
								case m.CancelRequestChan <- struct{}{}:
								default:
								}
								m.WaitingForAI = false
							}
							return m, nil
						}
						if chosen == "/help" {
							m.OverlayActive = true
							m.OverlayTitle = "Help"
							m.OverlayContent = i18n.T(m.getLang(), i18n.KeyHelpText)
							m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
							m.OverlayViewport.SetContent(m.OverlayContent)
							return m, nil
						}
						if chosen == "/config show" {
							m = m.showConfig()
							return m, nil
						}
						if chosen == "/config update auto-run list" {
							m = m.applyConfigAllowlistUpdate()
							return m, nil
						}
						if chosen == "/config add-remote" {
							m.OverlayActive = true
							m.OverlayTitle = "Add Remote"
							m.AddRemoteActive = true
							m.AddRemoteError = ""
							m.AddRemoteOfferOverwrite = false
							m.PathCompletionCandidates = nil
							m.PathCompletionIndex = -1
							m.AddRemoteFieldIndex = 0
							m.AddRemoteHostInput = textinput.New()
							m.AddRemoteHostInput.Placeholder = "host or host:22"
							m.AddRemoteHostInput.Focus()
							m.AddRemoteUserInput = textinput.New()
							m.AddRemoteUserInput.Placeholder = "e.g. root"
							m.AddRemoteUserInput.SetValue("root")
							m.AddRemoteNameInput = textinput.New()
							m.AddRemoteNameInput.Placeholder = "name (optional)"
							m.AddRemoteKeyInput = textinput.New()
							m.AddRemoteKeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
							return m, nil
						}
						if strings.HasPrefix(chosen, "/config add-remote ") {
							m.Input.SetValue("/config add-remote ")
							m.Input.CursorEnd()
							return m, nil
						}
						if strings.HasPrefix(chosen, "/config remove-remote ") {
							nameOrTarget := strings.TrimSpace(strings.TrimPrefix(chosen, "/config remove-remote "))
							if nameOrTarget != "" {
								m = m.applyConfigRemoveRemote(nameOrTarget)
								return m, nil
							}
							m.Input.SetValue("/config remove-remote ")
							m.Input.CursorEnd()
							return m, nil
						}
						if chosen == "/config remove-remote" {
							m.Input.SetValue("/config remove-remote ")
							m.Input.CursorEnd()
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
						if strings.HasPrefix(chosen, "/config llm base_url") {
							m.Input.SetValue("/config llm base_url ")
							m.Input.CursorEnd()
							return m, nil
						}
						if strings.HasPrefix(chosen, "/config llm api_key") {
							m.Input.SetValue("/config llm api_key ")
							m.Input.CursorEnd()
							return m, nil
						}
						if strings.HasPrefix(chosen, "/config llm model") {
							m.Input.SetValue("/config llm model ")
							m.Input.CursorEnd()
							return m, nil
						}
						if chosen == "/config" || strings.HasPrefix(chosen, "/config") {
							m = m.showConfig()
							return m, nil
						}
						if chosen == "/reload" {
							if m.ConfigUpdatedChan != nil {
								select {
								case m.ConfigUpdatedChan <- struct{}{}:
								default:
								}
							}
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
						if strings.HasPrefix(chosen, "/remote on ") {
							target := strings.TrimSpace(strings.TrimPrefix(chosen, "/remote on "))
							if target != "" && m.RemoteOnChan != nil {
								select {
								case m.RemoteOnChan <- target:
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
						if chosen == "/remote off" {
							if m.RemoteOffChan != nil {
								select {
								case m.RemoteOffChan <- struct{}{}:
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
					}
				m.Messages = append(m.Messages, errStyle.Render(i18n.T(m.getLang(), i18n.KeyUnknownCmd)))
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
		opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath)
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
		var cmd tea.Cmd
		m.Viewport, cmd = m.Viewport.Update(msg)
		return m, cmd

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
		lang := m.getLang()
		m.CurrentSessionPath = msg.Path
		sessionID := ""
		if msg.Path != "" {
			sessionID = strings.TrimSuffix(filepath.Base(msg.Path), ".jsonl")
		}
		switchedLine := sessionSwitchedStyle.Render(i18n.Tf(lang, i18n.KeySessionSwitchedTo, sessionID))
		if msg.Path != "" {
			events, _ := history.ReadRecent(msg.Path, maxSessionHistoryEvents)
			msgs := sessionEventsToMessages(events, lang)
			m.Messages = make([]string, 0, len(msgs)+1)
			m.Messages = append(m.Messages, msgs...)
			m.Messages = append(m.Messages, switchedLine)
		} else {
			m.Messages = []string{switchedLine}
		}
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case ConfigReloadedMsg:
		lang := m.getLang()
		m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyConfigReloaded)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil
	case AgentReplyMsg:
		m.WaitingForAI = false
		lang := m.getLang()
		if msg.Err != nil {
			if errors.Is(msg.Err, context.Canceled) {
				m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyCancelled)))
			} else if errors.Is(msg.Err, agent.ErrLLMNotConfigured) {
				m.Messages = append(m.Messages, errStyle.Render(i18n.Tf(lang, i18n.KeyErrLLMNotConfigured, config.ConfigPath())))
			} else {
				m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyErrorPrefix)+msg.Err.Error()))
			}
		} else if msg.Reply != "" {
			m.Messages = append(m.Messages, i18n.T(lang, i18n.KeyAILabel)+msg.Reply)
		}
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil
	case SystemNotifyMsg:
		if msg.Text != "" {
			m.Messages = append(m.Messages, suggestStyle.Render(msg.Text))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
		}
		return m, nil

	case CommandExecutedMsg:
		lang := m.getLang()
		var tag string
		if msg.Direct {
			tag = i18n.T(lang, i18n.KeyRunTagDirect)
		} else if msg.Allowed {
			tag = i18n.T(lang, i18n.KeyRunTagAllowlist)
		} else {
			tag = i18n.T(lang, i18n.KeyRunTagApproved)
		}
		m.Messages = append(m.Messages, execStyle.Render(i18n.T(lang, i18n.KeyRunLabel)+msg.Command+" ("+tag+")"))
		if msg.Sensitive {
			m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyResultSensitive)))
		}
		if msg.Result != "" {
			m.Messages = append(m.Messages, resultStyle.Render(msg.Result))
		}
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// NewModel creates a Model with default input (slash commands and viewport scrolling).
// initialMessages if non-nil is used as existing conversation (e.g. after /sh return).
// initialSessionPath is the current session file path (excluded from /sessions list so first option is another session).
// allowlistAutoRunChangeChan and getAllowlistAutoRun are for runtime toggle and header/card options.
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
) Model {
	ti := textinput.New()
	ti.Placeholder = i18n.T("en", i18n.KeyPlaceholderInput)
	ti.Prompt = "> "
	ti.PromptStyle = inputPromptStyle
	ti.TextStyle = inputTextStyle
	ti.Cursor.Style = inputCursorStyle
	ti.CharLimit = 0
	ti.Width = 60
	ti.Focus()
	vp := viewport.New(defaultWidth, defaultHeight-4)
	vp.MouseWheelEnabled = true
	msgs := []string(nil)
	if len(initialMessages) > 0 {
		msgs = make([]string, len(initialMessages))
		copy(msgs, initialMessages)
	}
	return Model{
		Input:                     ti,
		Viewport:                  vp,
		Messages:                  msgs,
		SubmitChan:                submitChan,
		ExecDirectChan:            execDirectChan,
		ShellRequestedChan:        shellRequestedChan,
		CancelRequestChan:         cancelRequestChan,
		ConfigUpdatedChan:         configUpdatedChan,
		AllowlistAutoRunChangeChan: allowlistAutoRunChangeChan,
		SessionSwitchChan:          sessionSwitchChan,
		RemoteOnChan:               remoteOnChan,
		RemoteOffChan:              remoteOffChan,
		RemoteAuthRespChan:         remoteAuthRespChan,
		CurrentSessionPath:         initialSessionPath,
		GetAllowlistAutoRun:        getAllowlistAutoRun,
		Width:                     defaultWidth,
		Height:                    defaultHeight,
	}
}
