package ui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
)

const (
	minInputLayoutWidth      = 4
	minContentWidthFallback  = 80
	minOverlayLayoutWidth    = 4
	minOverlayLayoutHeight   = 6
	maxOverlayViewportHeight = 20
	inputTextareaMinHeight   = 1
	inputTextareaMaxHeight   = 5
	inputBelowStableRows     = 5
)

var transcriptAnsiStrip = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[\[?][0-9;]*[a-zA-Z]?`)

// ReadModel provides host-derived read-only state needed by UI rendering and local decisions.
type ReadModel interface {
	AllowlistAutoRunEnabled() bool
	TakeOpenConfigLLMOnFirstLayout() bool
}

func (m Model) allowlistAutoRunEnabled() bool {
	if m.ReadModel == nil {
		return true
	}
	return m.ReadModel.AllowlistAutoRunEnabled()
}

func (m Model) takeOpenConfigLLMOnFirstLayout() bool {
	if m.ReadModel == nil {
		return false
	}
	return m.ReadModel.TakeOpenConfigLLMOnFirstLayout()
}

type uiState string

const (
	uiStateMainInput     uiState = "main_input"
	uiStateChoiceCard    uiState = "choice_card"
	uiStateChoiceCardAlt uiState = "choice_card_alt"
	uiStateOverlay       uiState = "overlay"
)

// currentUIState is a lightweight FSM view of current UI mode.
// Priority follows interactive exclusivity: pending > overlay > main.
func (m Model) currentUIState() uiState {
	if m.ChoiceCard.pendingSensitive != nil {
		return uiStateChoiceCardAlt
	}
	if m.ChoiceCard.pending != nil {
		return uiStateChoiceCard
	}
	if m.Overlay.Active {
		return uiStateOverlay
	}
	return uiStateMainInput
}

// TranscriptLines returns a copy of the current transcript lines shown in the main viewport.
func (m Model) TranscriptLines() []string {
	if len(m.messages) == 0 {
		return nil
	}
	out := make([]string, len(m.messages))
	copy(out, m.messages)
	return out
}

// WithTranscriptLines replaces the transcript with the provided lines (copied).
func (m Model) WithTranscriptLines(lines []string) Model {
	if len(lines) == 0 {
		m.messages = nil
		return m
	}
	out := make([]string, len(lines))
	copy(out, lines)
	m.messages = out
	return m
}

// AppendTranscriptLines appends rendered transcript lines.
func (m Model) AppendTranscriptLines(lines ...string) Model {
	if len(lines) == 0 {
		return m
	}
	m.messages = append(m.messages, lines...)
	return m
}

func teaCmdForMsg(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (m Model) printTranscriptCmd(clearFirst bool) (Model, tea.Cmd) {
	if m.Overlay.Active || m.printedMessages >= len(m.messages) {
		return m, nil
	}
	cmds := make([]tea.Cmd, 0, len(m.messages)-m.printedMessages+1)
	if clearFirst {
		cmds = append(cmds, teaCmdForMsg(tea.ClearScreen()))
	}
	for _, line := range m.messages[m.printedMessages:] {
		cmds = append(cmds, tea.Println(line))
	}
	m.printedMessages = len(m.messages)
	return m, tea.Sequence(cmds...)
}

func (m Model) withTranscriptReplaced(lines []string) Model {
	m = m.WithTranscriptLines(lines)
	m.printedMessages = 0
	return m
}

// RefreshViewport rebuilds the view content and scrolls to bottom.
// This is used by exact slash handlers that need immediate UI feedback.
func (m Model) RefreshViewport() Model {
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// SetMainViewportContent rebuilds the main transcript viewport without changing scroll position.
func (m Model) SetMainViewportContent() Model {
	m.Viewport.SetContent(m.buildContent())
	return m
}

// OpenOverlayFeature opens a feature-owned overlay and records its active key.
func (m Model) OpenOverlayFeature(key, title, content string) Model {
	m.Overlay.Active = true
	m.Overlay.Key = key
	m.Overlay.Title = title
	m.Overlay.Content = content
	return m
}

// CloseOverlayVisual closes overlay chrome only.
// Feature-specific flags are still owned by each feature package.
func (m Model) CloseOverlayVisual() Model {
	m.Overlay.Active = false
	m.Overlay.Key = ""
	m.Overlay.Title = ""
	m.Overlay.Content = ""
	return m
}

// InitOverlayViewport initializes the generic overlay viewport from current layout.
func (m Model) InitOverlayViewport() Model {
	m.Overlay.Viewport = viewport.New(m.layout.Width-minOverlayLayoutWidth, min(m.layout.Height-minOverlayLayoutHeight, maxOverlayViewportHeight))
	m.Overlay.Viewport.SetContent(m.Overlay.Content)
	return m
}

// hasPendingApproval reports whether the UI is in approval choice mode.
func (m Model) hasPendingChoiceCard() bool {
	return m.ChoiceCard.pending != nil || m.ChoiceCard.pendingSensitive != nil
}

// contentWidth returns a safe rendering width with fallback.
func (m Model) contentWidth() int {
	w := m.layout.Width
	if w <= 0 {
		return minContentWidthFallback
	}
	return w
}

// syncInputHeight keeps the textarea height in step with the current content.
func (m Model) syncInputHeight() Model {
	target := inputTextareaMinHeight
	if m.Input.LineCount() > 1 {
		target = inputTextareaMaxHeight
	}
	if m.Input.Height() != target {
		m.Input.SetHeight(target)
	}
	return m
}

// inputChromeHeight returns the total number of lines reserved below the transcript viewport.
func (m Model) inputChromeHeight() int {
	height := 1 // separator above input
	height += m.Input.Height()
	if m.Input.LineCount() > 1 {
		height += 1 // visual gap between multiline textarea and the below-input block
	}
	height += m.inputBelowHeight()
	height += 1 // footer
	return height
}

// inputBelowHeight returns the number of lines reserved below the input box.
func (m Model) inputBelowHeight() int {
	if m.hasPendingChoiceCard() {
		return inputBelowStableRows
	}
	if m.Input.LineCount() > 1 {
		return 1
	}
	if strings.HasPrefix(m.Input.Value(), "/") {
		_, vis, _ := m.slashSuggestionContext(m.Input.Value())
		if len(vis) > 0 {
			return inputBelowStableRows
		}
	}
	return inputBelowStableRows
}

// mainViewportHeight returns the viewport height used by main content.
func (m Model) mainViewportHeight() int {
	vh := m.layout.Height - m.inputChromeHeight()
	if vh < 1 {
		return 1
	}
	return vh
}

func (m Model) mainBodyView() string {
	if m.hasPendingChoiceCard() {
		m.Viewport.Width = m.layout.Width
		m.Viewport.Height = m.mainViewportHeight()
		m.Viewport.SetContent(m.pendingChoiceContent())
		m.Viewport.GotoBottom()
		return m.Viewport.View()
	}
	padLines := m.mainTopPaddingLines()
	if padLines <= 0 {
		return ""
	}
	return strings.Repeat("\n", padLines)
}

func (m Model) pendingChoiceContent() string {
	var b strings.Builder
	m.appendApprovalViewportContent(&b)
	return b.String()
}

func (m Model) printedTranscriptLineCount() int {
	if m.printedMessages <= 0 || len(m.messages) == 0 {
		return 0
	}
	limit := m.printedMessages
	if limit > len(m.messages) {
		limit = len(m.messages)
	}
	total := 0
	for _, line := range m.messages[:limit] {
		total += strings.Count(line, "\n") + 1
	}
	return total
}

func (m Model) mainTopPaddingLines() int {
	if m.layout.Height <= 0 {
		return 0
	}
	available := m.layout.Height - m.inputChromeHeight()
	if available <= 0 {
		return 0
	}
	visiblePrinted := m.printedTranscriptLineCount()
	if visiblePrinted > available {
		visiblePrinted = available
	}
	pad := available - visiblePrinted
	if pad < 0 {
		return 0
	}
	return pad
}

func (m Model) finalizeUpdate(prevOverlayActive bool, next Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if !prevOverlayActive && next.Overlay.Active {
		return next, tea.Sequence(
			teaCmdForMsg(tea.EnterAltScreen()),
			teaCmdForMsg(tea.ClearScreen()),
			cmd,
		)
	}
	if prevOverlayActive && !next.Overlay.Active {
		next, printCmd := next.printTranscriptCmd(false)
		return next, tea.Sequence(
			teaCmdForMsg(tea.ExitAltScreen()),
			cmd,
			printCmd,
		)
	}
	return next, cmd
}

// visibleScreenLines returns the rendered lines for the current on-screen UI, excluding any selection styling.
func (m Model) visibleScreenLines() []string {
	return m.visibleScreenBuffer().Lines
}

func (m Model) visibleScreenBuffer() ScreenBuffer {
	return newScreenBuffer(m.renderScreenSnapshot())
}

// visibleScreenText returns the current visible screen as plain text with ANSI stripped.
func (m Model) visibleScreenText() string {
	lines := m.visibleScreenLines()
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for i, line := range lines {
		plain := transcriptAnsiStrip.ReplaceAllString(line, "")
		plain = strings.TrimRight(plain, " ")
		b.WriteString(plain)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// visiblePointForMouse maps a mouse coordinate to a visible screen point.
func (m Model) visiblePointForMouse(y, x int) (ScreenPoint, bool) {
	buf := m.visibleScreenBuffer()
	pt, ok := buf.clampPoint(y, x)
	if !ok {
		return ScreenPoint{}, false
	}
	return pt, true
}

func (m Model) screenSelectionBounds() (ScreenPoint, ScreenPoint, bool) {
	return m.visibleScreenBuffer().selectionBounds(m.TranscriptSelection)
}

// visibleLineForMouseY maps a mouse Y coordinate to a visible screen line index.
func (m Model) visibleLineForMouseY(y int) (int, bool) {
	lines := m.visibleScreenLines()
	if len(lines) == 0 {
		return 0, false
	}
	if y < 0 {
		y = 0
	}
	if y >= len(lines) {
		y = len(lines) - 1
	}
	return y, true
}

func (s ScreenSelectionState) bounds() (start, end ScreenPoint, ok bool) {
	if !s.Active {
		return ScreenPoint{}, ScreenPoint{}, false
	}
	start, end = s.Anchor, s.Focus
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	return start, end, true
}

func (m Model) transcriptSelectionText() (string, bool) {
	return m.visibleScreenBuffer().selectionText(m.TranscriptSelection)
}

// selectedOrVisibleScreenText returns the active selection text if present, else the current visible screen text.
func (m Model) selectedOrVisibleScreenText() string {
	if text, ok := m.transcriptSelectionText(); ok && text != "" {
		return text
	}
	return m.visibleScreenText()
}

func (m Model) withTranscriptSelection(start, end ScreenPoint) Model {
	m.TranscriptSelection.Active = true
	m.TranscriptSelection.Anchor = start
	m.TranscriptSelection.Focus = end
	return m
}

func (m Model) clearTranscriptSelection() Model {
	m.TranscriptSelection = TranscriptSelectionState{}
	return m
}

// renderSeparator returns a horizontal separator with provided width.
func renderSeparator(width int) string {
	if width < 1 {
		width = 1
	}
	return separatorStyle.Render(strings.Repeat("─", width))
}
