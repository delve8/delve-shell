package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui/uivm"
)

const (
	defaultWidth  = 80
	defaultHeight = 24

	// HistoryPreviewOverlayKey marks the /history read-only preview modal (Enter confirms switch, Esc cancels).
	HistoryPreviewOverlayKey = "history_preview"
)

// Model is the Bubble Tea session and approval UI.
type Model struct {
	Input           textarea.Model
	messages        []string
	printedMessages int
	// recenterStartupTitleOnce: first WindowSize replaces the default-width-centered title with contentWidth().
	recenterStartupTitleOnce bool
	ChoiceCard               ChoiceCardState
	CommandSender            CommandSender
	layout                   LayoutState
	Interaction              InteractionState

	// Overlay state: when Overlay.Active is true, a modal is rendered on top of the main UI.
	Overlay OverlayState

	// ReadModel is the injected read-only state provider for UI decisions/render.
	ReadModel ReadModel
	Remote    RemoteState
}

// InteractionState stores transient keyboard/interaction state.
type InteractionState struct {
	slashSuggestIndex int  // 0..len(visible)-1 when input starts with /
	ChoiceIndex       int  // 0-based selection when in Pending/PendingSensitive/PendingSuggested; Up/Down to move, Enter to confirm
	WaitingForAI      bool // when true blocks normal chat submit; slash still allowed unless CommandExecuting
	// CommandExecuting while a shell command runs (/exec or agent execute_command / run_skill); blocks input except Esc / Ctrl+C.
	CommandExecuting bool

	// inputHistory: recent submitted lines (non-slash single-line path + echoed slash lines); Up/Down recall when not in slash suggestion mode.
	inputHistory     []string
	inputHistIndex   int    // index into inputHistory while browsing, -1 when editing the tail draft
	inputHistScratch string // current buffer saved on first Up from the tail

	// pendingHistorySwitchID is the session id to pass to SessionSwitch when the user presses Enter in the history preview overlay.
	pendingHistorySwitchID string
}

// OfflinePasteState holds the paste textarea and callback for offline manual relay.
type OfflinePasteState struct {
	Command   string
	Reason    string
	RiskLevel string
	Paste     textarea.Model
	Respond   func(text string, cancelled bool)
	// copyFeedback is a transient line under the command after auto-copy on dialog open; cleared by offlinePasteCopyAckClearMsg.
	copyFeedback string
}

// ChoiceCardState stores current pending choice card (approval or sensitive confirmation).
type ChoiceCardState struct {
	pending          *uivm.PendingApproval
	pendingSensitive *uivm.PendingSensitive
	offlinePaste     *OfflinePasteState
}

// LayoutState stores terminal layout dimensions for rendering.
type LayoutState struct {
	Width  int
	Height int
}

// Layout returns a copy of the current layout dimensions.
func (m *Model) Layout() LayoutState {
	return m.layout
}

func (m *Model) LayoutWidth() int {
	return m.layout.Width
}

func (m *Model) LayoutHeight() int {
	return m.layout.Height
}

// OverlayState stores generic modal overlay state shared across features.
type OverlayState struct {
	Active   bool
	Key      string
	Title    string
	Content  string
	Viewport viewport.Model
}

type RemoteState struct {
	Active  bool
	Offline bool
	Label   string
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.Input.Cursor.BlinkCmd(), tea.WindowSize())
}

func languageFromConfig() string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return "en"
	}
	if s := strings.TrimSpace(cfg.Language); s != "" {
		return s
	}
	return "en"
}

// getLang returns the UI language for i18n (config language, default en).
func (m *Model) getLang() string {
	return languageFromConfig()
}

// GetLang returns the UI language code for i18n (e.g. "en"). Callers outside package ui use this.
func (m *Model) GetLang() string {
	return m.getLang()
}

// delveMsg prefixes msg with "Delve: " for tool/system messages (config, session, notify, etc.).
func (m *Model) delveMsg(msg string) string {
	return i18n.T(i18n.KeyDelveLabel) + " " + msg
}

// Update implements tea.Model.
//
// Routing (first match wins):
//   - stateEventProviderChain — feature-registered global state sync handlers.
//   - update_lifecycle.go — WindowSize, Blur, Focus, overlay open/close, mouse / viewport.
//   - update_overlay_key.go then update_keymsg.go, update_slash.go, update_approval.go — keyboard when overlay vs main input.
//   - update_approval.go, update_events.go — agent approval and transcript events.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	i18n.SetLang(m.getLang())
	prevOverlayActive := m.Overlay.Active
	m.syncInputPlaceholder()

	if m.Overlay.Active {
		if feature, ok := overlayFeatureByKey(m.Overlay.Key); ok && feature.Event != nil {
			if m2, cmd, handled := feature.Event(m, msg); handled {
				// Must match other overlay transitions: ExitAltScreen / print batch run in finalizeUpdate.
				return finalizeUpdate(prevOverlayActive, m2, cmd)
			}
		}
	}

	for _, p := range stateEventProviderChain.List() {
		if m2, cmd, handled := p(m, msg); handled {
			return finalizeUpdate(prevOverlayActive, m2, cmd)
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m2, cmd := m.handleWindowSizeMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case tea.BlurMsg:
		m2, cmd := m.handleBlurMsg()
		return finalizeUpdate(prevOverlayActive, m2, cmd)
	case tea.FocusMsg:
		m2, cmd := m.handleFocusMsg()
		return finalizeUpdate(prevOverlayActive, m2, cmd)
	case tea.KeyMsg:
		m2, cmd := m.handleKeyMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case tea.MouseMsg:
		m2, cmd := m.handleMouseMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case ChoiceCardShowMsg:
		m2, cmd := m.handleChoiceCardShowMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case OfflinePasteShowMsg:
		m2, cmd := m.handleOfflinePasteShowMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case TranscriptAppendMsg:
		m2, cmd := m.handleTranscriptAppendMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)
	case TranscriptReplaceMsg:
		m2, cmd := m.handleTranscriptReplaceMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case OverlayShowMsg:
		m2, cmd := m.handleOverlayShowMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case HistoryPreviewOverlayMsg:
		m2, cmd := m.handleHistoryPreviewOverlayMsg(msg)
		return finalizeUpdate(prevOverlayActive, m2, cmd)

	case transcriptPrintedMsg:
		if msg.upTo > len(m.messages) {
			msg.upTo = len(m.messages)
		}
		if msg.upTo < 0 {
			msg.upTo = 0
		}
		m.printedMessages = msg.upTo
		return finalizeUpdate(prevOverlayActive, m, nil)

	case offlinePasteCopyAckClearMsg:
		if m.ChoiceCard.offlinePaste != nil {
			m.ChoiceCard.offlinePaste.copyFeedback = ""
		}
		return finalizeUpdate(prevOverlayActive, m, nil)

	case CommandExecutionStateMsg:
		m.Interaction.CommandExecuting = msg.Active
		return finalizeUpdate(prevOverlayActive, m, nil)

	}

	return finalizeUpdate(prevOverlayActive, m, nil)
}

// NewModel creates a Model with default input (slash commands and transcript-aligned bottom chrome).
// initialMessages if non-nil is used as existing conversation (e.g. after /bash return).
func NewModel(initialMessages []string, readModel ReadModel) *Model {
	i18n.SetLang(languageFromConfig())
	ti := textarea.New()
	ti.Placeholder = i18n.T(i18n.KeyPlaceholderInput)
	ti.Prompt = "> "
	ti.ShowLineNumbers = false
	// InsertNewline: shift+enter when the TTY distinguishes it; alt+enter often works on macOS/Linux
	// (\e\r); ctrl+j is always distinct. Plain Enter and Shift+Enter are the same on many terminals.
	ti.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys(teakey.ShiftEnter, teakey.AltEnter, teakey.CtrlJ),
		key.WithHelp(teakey.InsertNewlineBindingHelp, "new line"),
	)
	ti.CharLimit = 0
	ti.SetHeight(inputTextareaMinHeight)
	ti.SetWidth(defaultWidth - 4) // will be updated on first WindowSizeMsg to match terminal
	ti.FocusedStyle.Prompt = inputPromptStyle
	ti.FocusedStyle.Text = inputTextStyle
	ti.FocusedStyle.Placeholder = inputPlaceholderStyle
	ti.BlurredStyle.Prompt = inputPromptStyle
	ti.BlurredStyle.Text = inputTextStyle
	ti.BlurredStyle.Placeholder = inputPlaceholderStyle
	ti.Cursor.Style = inputCursorStyle
	ti.Focus()
	msgs := []string(nil)
	if len(initialMessages) > 0 {
		msgs = make([]string, len(initialMessages))
		copy(msgs, initialMessages)
	} else {
		msgs = []string{startupTitleLine(defaultWidth)}
	}
	recenter := len(initialMessages) == 0
	remote := RemoteState{}
	if readModel != nil {
		a, lbl, off := readModel.InitialRemoteFooter()
		remote = RemoteState{Active: a, Label: lbl, Offline: off}
	}
	return &Model{
		Input:                    ti,
		messages:                 msgs,
		recenterStartupTitleOnce: recenter,
		ReadModel:                readModel,
		Remote:                   remote,
		Interaction: InteractionState{
			inputHistIndex: -1,
		},
		layout: LayoutState{
			Width:  defaultWidth,
			Height: defaultHeight,
		},
	}
}
