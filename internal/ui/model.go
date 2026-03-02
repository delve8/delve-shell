package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	execStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true)
	resultStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2)
	suggestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	suggestHi    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

type slashOption struct{ Cmd, Desc string }

// getSlashOptions 返回顶层斜杠命令（输入 "/" 时显示）；顺序：help, cancel, config, reload, run, sh, exit
func getSlashOptions(lang string) []slashOption {
	return []slashOption{
		{"/help", i18n.T(lang, i18n.KeyDescHelp)},
		{"/cancel", i18n.T(lang, i18n.KeyDescCancel)},
		{"/config", i18n.T(lang, i18n.KeyDescConfig)},
		{"/reload", i18n.T(lang, i18n.KeyDescReload)},
		{"/run <cmd>", i18n.T(lang, i18n.KeyDescRun)},
		{"/sh", i18n.T(lang, i18n.KeyDescSh)},
		{"/exit", i18n.T(lang, i18n.KeyDescExit)},
	}
}

// getConfigSubOptions 返回 /config 子项（仅当输入以 "/config" 开头时显示），不含 /exit、/sh 等。
func getConfigSubOptions(lang string) []slashOption {
	return []slashOption{
		{"/config show", i18n.T(lang, i18n.KeyDescConfigShow)},
		{"/config allowlist update", i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate)},
		{"/config llm base_url <url>", i18n.T(lang, i18n.KeyDescConfigLLMBaseURL)},
		{"/config llm api_key <key>", i18n.T(lang, i18n.KeyDescConfigLLMApiKey)},
		{"/config llm model <name>", i18n.T(lang, i18n.KeyDescConfigLLMModel)},
		{"/config language <en|zh>", i18n.T(lang, i18n.KeyDescConfigLanguage)},
	}
}

// getSlashOptionsForInput 根据当前输入返回应展示的斜杠选项：输入 "/config" 或 "/config xxx" 时只返回 /config 子项，否则返回顶层命令。
func getSlashOptionsForInput(inputVal string, lang string) []slashOption {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	if normalized == "config" || strings.HasPrefix(normalized, "config ") {
		return getConfigSubOptions(lang)
	}
	return getSlashOptions(lang)
}

// Model 为 Bubble Tea 的会话与审批 UI
type Model struct {
	Input               textinput.Model
	Viewport            viewport.Model
	Messages            []string
	Pending             *agent.ApprovalRequest
	SubmitChan          chan<- string
	ExecDirectChan      chan<- string
	ShellRequestedChan  chan<- []string // on /sh send current Messages to preserve after return
	CancelRequestChan   chan<- struct{}  // on /cancel request cancel of in-flight AI
	ConfigUpdatedChan   chan<- struct{}  // on /config save or /reload, invalidate runner so next message reloads config/allowlist
	Width               int
	Height              int
	SlashSuggestIndex   int  // 0..len(visible)-1 when input starts with /
	WaitingForAI       bool // true 时仅禁止再提交新问题（Enter 发消息）；/xxx 斜杠命令任何时候都可用
}

// Init 实现 tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Input.Cursor.BlinkCmd(), tea.WindowSize())
}

// getLang 返回当前配置的语言（用于 i18n），失败或未设置时返回 "en"
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

// visibleSlashOptions 根据当前输入过滤并返回可选斜杠命令的下标
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

// slashChosenToInputValue 将选中的斜杠命令转为填入输入框的字符串（带占位时去掉占位并加空格，便于用户继续输入）
func slashChosenToInputValue(chosen string) string {
	// 带 <...> 占位符的改为 "前缀 " 方便用户接着输入
	if strings.Contains(chosen, " <") {
		if i := strings.Index(chosen, " <"); i > 0 {
			return chosen[:i] + " "
		}
	}
	return chosen
}

// Update 实现 tea.Model
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
		if m.Pending != nil {
			switch msg.String() {
			case "y", "Y":
				m.Pending.ResponseCh <- true
				m.Pending = nil
				return m, nil
			case "n", "N":
				m.Pending.ResponseCh <- false
				m.Pending = nil
				m.WaitingForAI = false // 拒绝后立即允许继续输入，不必等 agent 返回
				return m, nil
			}
			return m, nil
		}

		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		inputVal := m.Input.Value()
		inSlash := strings.HasPrefix(inputVal, "/")

		// 滚动键：Up/Down 在斜杠模式下改选中项，否则与 PgUp/PgDown 一起交给 viewport
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
			// WaitingForAI 只限制「提交新问题」：以 / 开头的斜杠命令任何时候都可执行
			if m.WaitingForAI && !strings.HasPrefix(text, "/") {
				return m, nil
			}
			// 先判断是否为「仅填入选中项」：/ 后 Up/Down 选中再回车，只填输入区，不写入对话区、不执行
			if strings.HasPrefix(text, "/") {
				opts := getSlashOptionsForInput(text, m.getLang())
				vis := visibleSlashOptions(text, opts)
				if len(vis) > 0 && m.SlashSuggestIndex < len(vis) {
					chosen := opts[vis[m.SlashSuggestIndex]].Cmd
					// 选中项与当前输入不一致 ⇒ 视为「选中后填入」，不执行、不写入 View
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
					// 输入须对应所选命令；仅 "/" 时不处理。「仅填入」已在 Enter 开头提前 return
					if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && (chosen == text || strings.HasPrefix(chosen, text)) {
						// 用户输入与选中一致（完整输入后回车）→ 执行
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
		m.Pending = msg
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
	b.WriteString(titleStyle.Render(i18n.T(lang, i18n.KeyTitleHeader)) + "\n\n")
	for _, line := range m.Messages {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.Pending != nil {
		b.WriteString("\n")
		b.WriteString(titleStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)) + "\n")
		b.WriteString(m.Pending.Command)
		b.WriteString("\n")
		b.WriteString(i18n.T(lang, i18n.KeyApproveYN))
	}
	return b.String()
}

// applyConfigLLM 设置 config.yaml 中 llm 的某一项并写回；value 支持 $VAR 引用环境变量
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

// applyConfigLanguage 设置 config.yaml 的 language 并写回
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

// showConfig 在对话区显示当前 config 路径与 LLM 摘要（api_key 脱敏）
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

// applyConfigAllowlistUpdate 将内置默认允许列表合并到当前 allowlist.yaml，仅追加缺失的 pattern
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

// View 实现 tea.Model
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
	// 不在 View() 中 SetContent，避免每帧重置滚动导致 Up/Down/PgUp/PgDown 无效；内容仅在 Update() 中变更时设置
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

// NewModel 创建带默认输入框的 Model（支持斜杠命令与 viewport 滚动）。
// initialMessages 非 nil 时用作已有对话内容（如 /sh 返回后恢复）。
func NewModel(submitChan chan<- string, execDirectChan chan<- string, shellRequestedChan chan<- []string, cancelRequestChan chan<- struct{}, configUpdatedChan chan<- struct{}, initialMessages []string) Model {
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
		Width:               defaultWidth,
		Height:              defaultHeight,
	}
}
