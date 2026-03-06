package ui

import (
	"context"
	"errors"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
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
	PendingSuggested    *string // suggested command card: press c to copy, Enter to dismiss
	SubmitChan          chan<- string
	ExecDirectChan      chan<- string
	ShellRequestedChan  chan<- []string // on /sh send current Messages to preserve after return
	CancelRequestChan   chan<- struct{}  // on /cancel request cancel of in-flight AI
	ConfigUpdatedChan   chan<- struct{}  // on /config save or /reload, invalidate runner so next message reloads config/allowlist
	ModeChangeChan      chan<- string    // on /mode send new mode (suggest|run), runtime only, not written to config
	GetMode             func() string   // current runtime mode for display
	Width               int
	Height              int
	SlashSuggestIndex   int  // 0..len(visible)-1 when input starts with /
	ChoiceIndex         int  // 0-based selection when in Pending/PendingSensitive/PendingSuggested; Up/Down to move, Enter to confirm
	WaitingForAI        bool // when true only blocks submitting new messages (Enter); /xxx slash commands always allowed
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Input.Cursor.BlinkCmd(), tea.WindowSize())
}

// getLang returns the current config language (for i18n); returns "en" on failure or when unset.
func (m Model) getLang() string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return "en"
	}
	if cfg.Language != "" {
		return cfg.Language
	}
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

	case tea.KeyMsg:
		key := msg.String()

		// Always allow ctrl+c to quit, even during pending approvals or sensitive prompts.
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		inChoice := m.Pending != nil || m.PendingSensitive != nil || m.PendingSuggested != nil
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

				m.Pending.ResponseCh <- true
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

				m.Pending.ResponseCh <- false
				m.Pending = nil
				m.WaitingForAI = false // after reject allow input immediately, no need to wait for agent
				return m, nil
			}
			return m, nil
		}
		if m.PendingSuggested != nil {
			lang := m.getLang()
			switch key {
			case "1":
				_ = clipboard.WriteAll(*m.PendingSuggested)
				m.appendSuggestedLine(*m.PendingSuggested, lang)
				m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopied)))
				m.PendingSuggested = nil
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			case "2":
				m.appendSuggestedLine(*m.PendingSuggested, lang)
				m.PendingSuggested = nil
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			}
			return m, nil
		}

		inputVal := m.Input.Value()
		inSlash := strings.HasPrefix(inputVal, "/")

		// scroll keys: Up/Down change selection in slash mode, else go to viewport with PgUp/PgDown
		if key == "up" || key == "down" || key == "pgup" || key == "pgdown" {
			if inSlash && (key == "up" || key == "down") {
				opts := getSlashOptionsForInput(inputVal, m.getLang())
				vis := visibleSlashOptions(inputVal, opts)
				if len(vis) > 0 {
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
			// first check "fill selection only": / then Up/Down then Enter only fills input, does not submit
			if strings.HasPrefix(text, "/") {
				opts := getSlashOptionsForInput(text, m.getLang())
				vis := visibleSlashOptions(text, opts)
				if len(vis) > 0 && m.SlashSuggestIndex < len(vis) {
					chosen := opts[vis[m.SlashSuggestIndex]].Cmd
					// chosen != text => fill selection only, do not execute or add to View
					if (chosen == text || strings.HasPrefix(chosen, text)) && chosen != text {
						m.Input.SetValue(slashChosenToInputValue(chosen))
						m.Input.CursorEnd()
						return m, nil
					}
				}
			}
			m.Messages = append(m.Messages, i18n.T(m.getLang(), i18n.KeyUserLabel)+text)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0

			switch {
			case text == "/exit":
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
				m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(m.getLang(), i18n.KeyHelpText)))
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
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
			case strings.HasPrefix(text, "/config language "):
				m = m.applyConfigLanguage(strings.TrimSpace(strings.TrimPrefix(text, "/config language ")))
				return m, nil
			case text == "/config show", text == "/config":
				m = m.showConfig()
				return m, nil
			case text == "/config allowlist update":
				m = m.applyConfigAllowlistUpdate()
				return m, nil
			case strings.HasPrefix(text, "/mode "), text == "/mode":
				arg := strings.TrimSpace(strings.TrimPrefix(text, "/mode"))
				m = m.applyModeSwitch(arg)
				return m, nil
			case strings.HasPrefix(text, "/config mode "):
				m = m.applyConfigMode(strings.TrimSpace(strings.TrimPrefix(text, "/config mode ")))
				return m, nil
			case text == "/reload":
				if m.ConfigUpdatedChan != nil {
					select {
					case m.ConfigUpdatedChan <- struct{}{}:
					default:
					}
				}
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
				opts := getSlashOptionsForInput(text, m.getLang())
				vis := visibleSlashOptions(text, opts)
				if len(vis) > 0 && m.SlashSuggestIndex < len(vis) {
					chosen := opts[vis[m.SlashSuggestIndex]].Cmd
					// input must match chosen command; skip when only "/". "Fill only" already returned above.
					if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && (chosen == text || strings.HasPrefix(chosen, text)) {
						// user input matches chosen (full input then Enter) => execute
						if chosen == "/exit" {
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
							m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(m.getLang(), i18n.KeyHelpText)))
							m.Viewport.SetContent(m.buildContent())
							m.Viewport.GotoBottom()
							return m, nil
						}
						if chosen == "/config show" {
							m = m.showConfig()
							return m, nil
						}
						if chosen == "/config allowlist update" {
							m = m.applyConfigAllowlistUpdate()
							return m, nil
						}
						if chosen == "/mode suggest" || chosen == "/mode run" {
							modeArg := strings.TrimSpace(strings.TrimPrefix(chosen, "/mode"))
							m = m.applyModeSwitch(modeArg)
							return m, nil
						}
						if chosen == "/config mode <suggest|run>" {
							m.Input.SetValue("/config mode ")
							m.Input.CursorEnd()
							return m, nil
						}
						if strings.HasPrefix(chosen, "/config mode") {
							m.Input.SetValue("/config mode ")
							m.Input.CursorEnd()
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
						if strings.HasPrefix(chosen, "/config language") {
							m.Input.SetValue("/config language ")
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
			m.SlashSuggestIndex = 0
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

	case CommandExecutedMsg:
		lang := m.getLang()
		if msg.Suggested {
			// Show a card and wait for user to press c (copy) or Enter (dismiss)
			if m.PendingSuggested != nil {
				m.appendSuggestedLine(*m.PendingSuggested, lang)
			}
			cmd := msg.Command
			m.PendingSuggested = &cmd
			m.ChoiceIndex = 0
			m.syncInputPlaceholder()
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		}
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
// modeChangeChan and getMode are for /mode (runtime switch); getMode returns current mode for title.
func NewModel(
	submitChan chan<- string,
	execDirectChan chan<- string,
	shellRequestedChan chan<- []string,
	cancelRequestChan chan<- struct{},
	configUpdatedChan chan<- struct{},
	modeChangeChan chan<- string,
	getMode func() string,
	initialMessages []string,
) Model {
	ti := textinput.New()
	lang := "en"
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.Language != "" {
		lang = cfg.Language
	}
	ti.Placeholder = i18n.T(lang, i18n.KeyPlaceholderInput)
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
		Input:               ti,
		Viewport:            vp,
		Messages:            msgs,
		SubmitChan:          submitChan,
		ExecDirectChan:      execDirectChan,
		ShellRequestedChan:  shellRequestedChan,
		CancelRequestChan:   cancelRequestChan,
		ConfigUpdatedChan:   configUpdatedChan,
		ModeChangeChan:      modeChangeChan,
		GetMode:             getMode,
		Width:               defaultWidth,
		Height:              defaultHeight,
	}
}
