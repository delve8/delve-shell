package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/atotto/clipboard"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle                  = lipgloss.NewStyle().Bold(true)
	errStyle                    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	execStyle                   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true)
	resultStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2)
	suggestStyle                = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	suggestHi                   = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	riskReadOnlyStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)  // green
	riskLowStyle                = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)  // yellow
	riskHighStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)  // red
	approvalHeaderStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // cyan, for HIL approval/sensitive headers
	approvalDecisionApprovedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true) // green
	approvalDecisionRejectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // red
	hintStyle                   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true) // dim, italic for hint lines (copy hint, "Copied to clipboard")
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

type slashOption struct{ Cmd, Desc string }

// getSlashOptions returns top-level slash commands (shown when input starts with "/"); order: help, cancel, config, mode, reload, run, sh, exit.
func getSlashOptions(lang string) []slashOption {
	return []slashOption{
		{"/help", i18n.T(lang, i18n.KeyDescHelp)},
		{"/cancel", i18n.T(lang, i18n.KeyDescCancel)},
		{"/config", i18n.T(lang, i18n.KeyDescConfig)},
		{"/mode suggest", i18n.T(lang, i18n.KeyDescModeSuggest)},
		{"/mode run", i18n.T(lang, i18n.KeyDescModeRun)},
		{"/reload", i18n.T(lang, i18n.KeyDescReload)},
		{"/run <cmd>", i18n.T(lang, i18n.KeyDescRun)},
		{"/sh", i18n.T(lang, i18n.KeyDescSh)},
		{"/exit", i18n.T(lang, i18n.KeyDescExit)},
	}
}

// getConfigSubOptions returns /config sub-options (shown when input starts with "/config"), not /exit, /sh, etc.
func getConfigSubOptions(lang string) []slashOption {
	return []slashOption{
		{"/config show", i18n.T(lang, i18n.KeyDescConfigShow)},
		{"/config mode <suggest|run>", i18n.T(lang, i18n.KeyDescConfigMode)},
		{"/config allowlist update", i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate)},
		{"/config llm base_url <url>", i18n.T(lang, i18n.KeyDescConfigLLMBaseURL)},
		{"/config llm api_key <key>", i18n.T(lang, i18n.KeyDescConfigLLMApiKey)},
		{"/config llm model <name>", i18n.T(lang, i18n.KeyDescConfigLLMModel)},
		{"/config language <en|zh>", i18n.T(lang, i18n.KeyDescConfigLanguage)},
	}
}

// getSlashOptionsForInput returns slash options to show: when input is "/config" or "/config xxx" returns only /config sub-options; when "/mode" or "/mode x" returns mode sub-options; else top-level commands.
func getSlashOptionsForInput(inputVal string, lang string) []slashOption {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	if normalized == "config" || strings.HasPrefix(normalized, "config ") {
		return getConfigSubOptions(lang)
	}
	if normalized == "mode" || strings.HasPrefix(normalized, "mode ") {
		return getSlashOptions(lang) // /mode suggest, /mode run
	}
	return getSlashOptions(lang)
}

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

// visibleSlashOptions filters and returns indices of visible slash options for the current input.
func visibleSlashOptions(input string, opts []slashOption) []int {
	input = strings.TrimPrefix(input, "/")
	input = strings.ToLower(input)
	var out []int
	for i, opt := range opts {
		base := strings.Split(opt.Cmd, " ")[0]
		base = strings.TrimPrefix(base, "/")
		if input == "" || strings.HasPrefix(base, input) || strings.HasPrefix(opt.Cmd, "/"+input) {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		for i := range opts {
			out = append(out, i)
		}
	}
	return out
}

// slashChosenToInputValue converts the chosen slash command to the string to put in the input (strips <placeholder> and adds space).
func slashChosenToInputValue(chosen string) string {
	// replace <...> placeholder with "prefix " so user can continue typing
	if strings.Contains(chosen, " <") {
		if i := strings.Index(chosen, " <"); i > 0 {
			return chosen[:i] + " "
		}
	}
	return chosen
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Height > 4 {
			vh := m.Height - 4
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

		if m.PendingSensitive != nil {
			lang := m.getLang()
			switch msg.String() {
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
			switch msg.String() {
			case "1", "y", "Y":
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
			case "2", "n", "N":
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
			case "1", "c", "C":
				_ = clipboard.WriteAll(*m.PendingSuggested)
				m.appendSuggestedLine(*m.PendingSuggested, lang)
				m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopied)))
				m.PendingSuggested = nil
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
				return m, nil
			case "2", "enter":
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
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil

	case SensitiveConfirmationRequestMsg:
		// Same as approval: ensure the sensitive confirmation card is visible.
		m.PendingSensitive = msg
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

func (m Model) buildContent() string {
	lang := m.getLang()
	var b strings.Builder
	modeStr := "run"
	if m.GetMode != nil {
		modeStr = m.GetMode()
	}
	title := i18n.T(lang, i18n.KeyTitleHeader) + " | " + i18n.T(lang, i18n.KeyModeLabel) + ": " + modeStr
	b.WriteString(titleStyle.Render(title) + "\n\n")
	for _, line := range m.Messages {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.PendingSensitive != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)) + "\n")
		b.WriteString(execStyle.Render(m.PendingSensitive.Command) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice1)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice2)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice3)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitivePressKey)))
		return b.String()
	}
	if m.Pending != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)) + "\n")
		switch m.Pending.RiskLevel {
		case "read_only":
			b.WriteString(riskReadOnlyStyle.Render("["+i18n.T(lang, i18n.KeyRiskReadOnly)+"] ") + m.Pending.Command + "\n")
		case "low":
			b.WriteString(riskLowStyle.Render("["+i18n.T(lang, i18n.KeyRiskLow)+"] ") + m.Pending.Command + "\n")
		case "high":
			b.WriteString(riskHighStyle.Render("["+i18n.T(lang, i18n.KeyRiskHigh)+"] ") + m.Pending.Command + "\n")
		default:
			b.WriteString(m.Pending.Command + "\n")
		}
		if m.Pending.Reason != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason) + "\n")
		}
		b.WriteString(i18n.T(lang, i18n.KeyApproveYN))
		return b.String()
	}
	if m.PendingSuggested != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySuggestedCardTitle)) + "\n")
		b.WriteString(execStyle.Render(*m.PendingSuggested) + "\n")
		b.WriteString(hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCardHint)))
		return b.String()
	}
	return b.String()
}

// appendSuggestedLine appends the run line and copy hint for a suggested command (when dismissing the card).
func (m *Model) appendSuggestedLine(command, lang string) {
	tag := i18n.T(lang, i18n.KeyRunTagSuggested)
	m.Messages = append(m.Messages, execStyle.Render(i18n.T(lang, i18n.KeyRunLabel)+command+" ("+tag+")"))
	m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopyHint)))
}

// applyConfigLLM sets one llm field in config.yaml and writes back; value supports $VAR env expansion.
func (m Model) applyConfigLLM(field, value string) Model {
	value = strings.TrimSpace(value)
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	switch field {
	case "base_url":
		cfg.LLM.BaseURL = value
	case "api_key":
		cfg.LLM.APIKey = value
	case "model":
		cfg.LLM.Model = value
	default:
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigUnknownField)+field))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigSaved, field)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigLanguage sets config.yaml language and writes back.
func (m Model) applyConfigLanguage(value string) Model {
	value = strings.TrimSpace(value)
	if value == "" {
		lang := m.getLang()
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigLanguageRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	cfg.Language = value
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	// update current input placeholder to reflect new language immediately
	m.Input.Placeholder = i18n.T(value, i18n.KeyPlaceholderInput)
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigSavedLanguage, value)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// showConfig displays current config path and LLM summary (api_key masked) in the conversation area.
func (m Model) showConfig() Model {
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(config.ConfigPath()+"\n"+cfg.LLMSummary()))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// applyModeSwitch sets runtime mode to the given value (suggest or run) and sends to ModeChangeChan; does not write config.
func (m Model) applyModeSwitch(modeArg string) Model {
	lang := m.getLang()
	modeArg = strings.TrimSpace(strings.ToLower(modeArg))
	if modeArg != "suggest" && modeArg != "run" {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyModeRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	if m.ModeChangeChan != nil {
		select {
		case m.ModeChangeChan <- modeArg:
		default:
		}
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyModeSetTo, modeArg)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// applyConfigMode sets default mode in config and writes config; next startup will use this mode.
func (m Model) applyConfigMode(value string) Model {
	value = strings.TrimSpace(strings.ToLower(value))
	if value != "suggest" && value != "run" {
		lang := m.getLang()
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+i18n.T(lang, i18n.KeyConfigModeRequired)))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	lang := m.getLang()
	cfg, err := config.Load()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	cfg.Mode = value
	if err := config.Write(cfg); err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyConfigSavedMode, value)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// applyConfigAllowlistUpdate merges built-in default allowlist into current allowlist.yaml, appending only missing patterns.
func (m Model) applyConfigAllowlistUpdate() Model {
	lang := m.getLang()
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		m.Messages = append(m.Messages, errStyle.Render(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m
	}
	m.Messages = append(m.Messages, suggestStyle.Render(i18n.Tf(lang, i18n.KeyAllowlistUpdateDone, added)))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

// View implements tea.Model.
func (m Model) View() string {
	lang := m.getLang()
	if m.Height <= 4 {
		out := m.buildContent() + "\n" + m.Input.View()
		if m.WaitingForAI {
			out += "\n" + suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel))
		}
		return out
	}
	vh := m.Height - 4
	if vh < 1 {
		vh = 1
	}
	m.Viewport.Width = m.Width
	m.Viewport.Height = vh
	// do not SetContent in View() to avoid resetting scroll every frame (would break Up/Down/PgUp/PgDown); set only in Update() when content changes
	out := m.Viewport.View()
	out += "\n"
	out += m.Input.View()
	inputVal := m.Input.Value()
	if strings.HasPrefix(inputVal, "/") {
		opts := getSlashOptionsForInput(inputVal, lang)
		vis := visibleSlashOptions(inputVal, opts)
		if len(vis) > 0 {
			out += "\n"
			for i, vi := range vis {
				opt := opts[vi]
				line := fmt.Sprintf("%-14s  %s", opt.Cmd, opt.Desc)
				if i == m.SlashSuggestIndex {
					out += suggestHi.Render(" "+line) + "\n"
				} else {
					out += suggestStyle.Render(" "+line) + "\n"
				}
			}
		}
	}
	if m.WaitingForAI {
		out += "\n"
		out += suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel))
	}
	return out
}

// NewModel creates a Model with default input (slash commands and viewport scrolling).
// initialMessages if non-nil is used as existing conversation (e.g. after /sh return).
// modeChangeChan and getMode are for /mode (runtime switch); getMode returns current mode for title.
func NewModel(submitChan chan<- string, execDirectChan chan<- string, shellRequestedChan chan<- []string, cancelRequestChan chan<- struct{}, configUpdatedChan chan<- struct{}, modeChangeChan chan<- string, getMode func() string, initialMessages []string) Model {
	ti := textinput.New()
	lang := "en"
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.Language != "" {
		lang = cfg.Language
	}
	ti.Placeholder = i18n.T(lang, i18n.KeyPlaceholderInput)
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	ti.TextStyle = lipgloss.NewStyle()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
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
