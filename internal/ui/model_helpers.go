package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/x/ansi"
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
	maxInputHistoryEntries   = 200
)

// ReadModel provides host-derived read-only state needed by UI rendering and local decisions.
type ReadModel interface {
	TakeOpenConfigLLMOnFirstLayout() bool
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
		m.recenterStartupTitleOnce = false
		return m
	}
	out := make([]string, len(lines))
	copy(out, lines)
	m.messages = out
	m.recenterStartupTitleOnce = false
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
	end := len(m.messages)
	for _, line := range m.messages[m.printedMessages:end] {
		cmds = append(cmds, tea.Println(line))
	}
	// Sync printed count before async cmds run: a second WindowSize (or other Update) can arrive
	// after this return but before transcriptPrintedMsg; without this, the same lines are enqueued twice.
	m.printedMessages = end
	cmds = append(cmds, func() tea.Msg {
		return transcriptPrintedMsg{upTo: end}
	})
	return m, tea.Sequence(cmds...)
}

func (m Model) withTranscriptReplaced(lines []string) Model {
	m = m.WithTranscriptLines(lines)
	m.printedMessages = 0
	return m
}

// RefreshViewport is kept as a compatibility shim for feature modules.
// In normal log-stream mode it is a no-op; in choice-card mode it syncs the controlled viewport.
func (m Model) RefreshViewport() Model {
	return m.syncChoiceViewport()
}

func (m Model) syncChoiceViewport() Model {
	if !m.hasPendingChoiceCard() {
		return m
	}
	m.Viewport.Width = m.layout.Width
	m.Viewport.Height = m.mainViewportHeight()
	m.Viewport.SetContent(m.pendingChoiceContent())
	m.Viewport.GotoBottom()
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
		m = m.syncChoiceViewport()
		return m.Viewport.View()
	}
	return ""
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
		total += terminalWrappedRows(line, m.contentWidth())
	}
	return total
}

func (m Model) normalModeTopPaddingLines(bottomBlock string) int {
	if m.layout.Height <= 0 {
		return 0
	}
	bottomLines := terminalWrappedRows(bottomBlock, m.contentWidth())
	visiblePrinted := m.printedTranscriptLineCount()
	if visiblePrinted > m.layout.Height {
		visiblePrinted = m.layout.Height
	}
	pad := m.layout.Height - visiblePrinted - bottomLines
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

// renderSeparator returns a horizontal separator with provided width.
func renderSeparator(width int) string {
	if width < 1 {
		width = 1
	}
	// Avoid drawing exactly to the terminal edge: many terminals will soft-wrap
	// a full-width line, which breaks our bottom-block line accounting.
	if width > 1 {
		width--
	}
	return separatorStyle.Render(strings.Repeat("─", width))
}

// terminalWrappedRows returns how many terminal rows a string occupies at the given width,
// matching soft-wrap behavior used when Bubble Tea prints transcript lines (see ansi.StringWidth
// in the standard renderer). Using runewidth on ANSI-stripped text was slightly off for styled
// transcript lines and caused the bottom block to drift upward over time.
func terminalWrappedRows(text string, width int) int {
	if width < 1 {
		width = 1
	}
	// Do not short-circuit text=="" : strings.Split("", "\n") is []string{""}, one blank display row,
	// matching tea.Println("") and the trailing "" appended by AppendUserInputLines after each user line.
	parts := strings.Split(text, "\n")
	total := 0
	for _, part := range parts {
		w := ansi.StringWidth(part)
		if w <= 0 {
			total++
			continue
		}
		total += (w + width - 1) / width
	}
	return total
}
