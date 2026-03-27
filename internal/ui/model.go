package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/uivm"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

// Model is the Bubble Tea session and approval UI.
type Model struct {
	Input               textarea.Model
	Viewport            viewport.Model
	messages            []string
	printedMessages     int
	ChoiceCard          ChoiceCardState
	CommandSender       CommandSender
	layout              LayoutState
	Interaction         InteractionState

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
	WaitingForAI      bool // when true only blocks submitting new messages (Enter); /xxx slash commands always allowed
}

// ChoiceCardState stores current pending choice card (approval or sensitive confirmation).
type ChoiceCardState struct {
	pending          *uivm.PendingApproval
	pendingSensitive *uivm.PendingSensitive
}

// LayoutState stores terminal layout dimensions for rendering.
type LayoutState struct {
	Width  int
	Height int
}

// Layout returns a copy of the current layout dimensions.
func (m Model) Layout() LayoutState {
	return m.layout
}

func (m Model) LayoutWidth() int {
	return m.layout.Width
}

func (m Model) LayoutHeight() int {
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
	Active bool
	Label  string
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Input.Cursor.BlinkCmd(), tea.WindowSize())
}

// getLang returns the UI language for i18n. Currently UI is English-only.
func (m Model) getLang() string {
	return "en"
}

// GetLang returns the UI language code for i18n (e.g. "en"). Callers outside package ui use this.
func (m Model) GetLang() string {
	return m.getLang()
}

// delveMsg prefixes msg with "Delve: " for tool/system messages (config, session, notify, etc.).
func (m Model) delveMsg(msg string) string {
	return i18n.T(m.getLang(), i18n.KeyDelveLabel) + " " + msg
}

// Update implements tea.Model.
//
// Routing (first match wins):
//   - stateEventProviderChain — feature-registered global state sync handlers.
//   - update_lifecycle.go — WindowSize, Blur, Focus, overlay open/close, mouse / viewport.
//   - update_overlay_key.go then update_keymsg.go, update_slash.go, update_approval.go — keyboard when overlay vs main input.
//   - update_approval.go, update_events.go — agent approval and transcript events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevOverlayActive := m.Overlay.Active
	m.syncInputPlaceholder()

	if m.Overlay.Active {
		if feature, ok := overlayFeatureByKey(m.Overlay.Key); ok && feature.Event != nil {
			if m2, cmd, handled := feature.Event(m, msg); handled {
				return m2, cmd
			}
		}
	}

	for _, p := range stateEventProviderChain.List() {
		if m2, cmd, handled := p(m, msg); handled {
			return m.finalizeUpdate(prevOverlayActive, m2, cmd)
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m2, cmd := m.handleWindowSizeMsg(msg)
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)

	case tea.BlurMsg:
		m2, cmd := m.handleBlurMsg()
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)
	case tea.FocusMsg:
		m2, cmd := m.handleFocusMsg()
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)
	case tea.KeyMsg:
		m2, cmd := m.handleKeyMsg(msg)
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)

	case tea.MouseMsg:
		m2, cmd := m.handleMouseMsg(msg)
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)

	case ChoiceCardShowMsg:
		m2, cmd := m.handleChoiceCardShowMsg(msg)
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)

	case TranscriptAppendMsg:
		m2, cmd := m.handleTranscriptAppendMsg(msg)
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)
	case TranscriptReplaceMsg:
		m2, cmd := m.handleTranscriptReplaceMsg(msg)
		return m.finalizeUpdate(prevOverlayActive, m2, cmd)

	case transcriptPrintedMsg:
		if msg.upTo > len(m.messages) {
			msg.upTo = len(m.messages)
		}
		if msg.upTo < 0 {
			msg.upTo = 0
		}
		m.printedMessages = msg.upTo
		return m.finalizeUpdate(prevOverlayActive, m, nil)

	}

	return m.finalizeUpdate(prevOverlayActive, m, nil)
}

// NewModel creates a Model with default input (slash commands and viewport scrolling).
// initialMessages if non-nil is used as existing conversation (e.g. after /sh return).
func NewModel(initialMessages []string, readModel ReadModel) Model {
	ti := textarea.New()
	ti.Placeholder = i18n.T("en", i18n.KeyPlaceholderInput)
	ti.Prompt = "> "
	ti.ShowLineNumbers = false
	ti.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+j"), key.WithHelp("ctrl+j", "new line"))
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
	vp := viewport.New(defaultWidth, defaultHeight-3)
	vp.MouseWheelEnabled = true
	msgs := []string(nil)
	if len(initialMessages) > 0 {
		msgs = make([]string, len(initialMessages))
		copy(msgs, initialMessages)
	}
	return Model{
		Input:     ti,
		Viewport:  vp,
		messages:  msgs,
		ReadModel: readModel,
		layout: LayoutState{
			Width:  defaultWidth,
			Height: defaultHeight,
		},
	}
}
