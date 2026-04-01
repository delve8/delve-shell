package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	// inputBelowStableRows: fixed lines below the input so the separator above the input and the footer
	// stay in a stable vertical band across idle / processing / slash-open (padded with blanks when needed).
	inputBelowStableRows     = 5
	maxInputHistoryEntries   = 200
)

// ReadModel provides host-derived read-only state needed by UI rendering and local decisions.
type ReadModel interface {
	TakeOpenConfigLLMOnFirstLayout() bool
	OfflineExecutionMode() bool
}

func (m *Model) takeOpenConfigLLMOnFirstLayout() bool {
	if m.ReadModel == nil {
		return false
	}
	return m.ReadModel.TakeOpenConfigLLMOnFirstLayout()
}

func (m *Model) offlineExecutionMode() bool {
	if m.ReadModel == nil {
		return false
	}
	return m.ReadModel.OfflineExecutionMode()
}

type uiState string

const (
	uiStateMainInput     uiState = "main_input"
	uiStateChoiceCard    uiState = "choice_card"
	uiStateChoiceCardAlt uiState = "choice_card_alt"
	uiStateOfflinePaste  uiState = "offline_paste"
	uiStateOverlay       uiState = "overlay"
)

// currentUIState is a lightweight FSM view of current UI mode.
// Priority follows interactive exclusivity: pending > overlay > main.
func (m *Model) currentUIState() uiState {
	if m.ChoiceCard.offlinePaste != nil {
		return uiStateOfflinePaste
	}
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
func (m *Model) TranscriptLines() []string {
	if len(m.messages) == 0 {
		return nil
	}
	out := make([]string, len(m.messages))
	copy(out, m.messages)
	return out
}

// WithTranscriptLines replaces the transcript with the provided lines (copied).
func (m *Model) WithTranscriptLines(lines []string) {
	if len(lines) == 0 {
		m.messages = nil
		m.recenterStartupTitleOnce = false
		return
	}
	out := make([]string, len(lines))
	copy(out, lines)
	m.messages = out
	m.recenterStartupTitleOnce = false
}

// AppendTranscriptLines appends rendered transcript lines.
func (m *Model) AppendTranscriptLines(lines ...string) {
	if len(lines) == 0 {
		return
	}
	m.messages = append(m.messages, lines...)
}

func teaCmdForMsg(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (m *Model) printTranscriptCmd(clearFirst bool) tea.Cmd {
	if m.Overlay.Active || m.printedMessages >= len(m.messages) {
		return nil
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
	return tea.Sequence(cmds...)
}

func (m *Model) withTranscriptReplaced(lines []string) {
	m.WithTranscriptLines(lines)
	m.printedMessages = 0
}

func (m *Model) syncChoiceViewport() {
	if !m.hasPendingChoiceCard() {
		return
	}
	m.Viewport.Width = m.layout.Width
	m.Viewport.Height = m.mainViewportHeight()
	m.Viewport.SetContent(m.pendingChoiceContent())
	m.Viewport.GotoBottom()
}

// OpenOverlayFeature opens a feature-owned overlay and records its active key.
func (m *Model) OpenOverlayFeature(key, title, content string) {
	m.Overlay.Active = true
	m.Overlay.Key = key
	m.Overlay.Title = title
	m.Overlay.Content = content
}

// CloseOverlayVisual closes overlay chrome only.
// Feature-specific flags are still owned by each feature package.
func (m *Model) CloseOverlayVisual() {
	m.Overlay.Active = false
	m.Overlay.Key = ""
	m.Overlay.Title = ""
	m.Overlay.Content = ""
}

// InitOverlayViewport initializes the generic overlay viewport from current layout.
func (m *Model) InitOverlayViewport() {
	m.Overlay.Viewport = viewport.New(m.layout.Width-minOverlayLayoutWidth, min(m.layout.Height-minOverlayLayoutHeight, maxOverlayViewportHeight))
	m.Overlay.Viewport.SetContent(m.Overlay.Content)
}

// hasPendingChoiceCard reports whether the UI is in approval, sensitive confirmation, or offline paste mode.
func (m *Model) hasPendingChoiceCard() bool {
	return m.ChoiceCard.pending != nil || m.ChoiceCard.pendingSensitive != nil || m.ChoiceCard.offlinePaste != nil
}

// contentWidth returns a safe rendering width with fallback.
func (m *Model) contentWidth() int {
	w := m.layout.Width
	if w <= 0 {
		return minContentWidthFallback
	}
	return w
}

// syncInputHeight keeps the textarea height in step with the current content.
func (m *Model) syncInputHeight() {
	target := inputTextareaMinHeight
	if m.Input.LineCount() > 1 {
		target = inputTextareaMaxHeight
	}
	if m.Input.Height() != target {
		m.Input.SetHeight(target)
	}
}

// inputChromeHeight returns the total number of lines reserved below the transcript viewport.
func (m *Model) inputChromeHeight() int {
	height := 1 // separator above input
	height += m.primaryInputHeight()
	if m.ChoiceCard.offlinePaste != nil {
		if m.ChoiceCard.offlinePaste.Paste.LineCount() > 1 {
			height += 1
		}
	} else if m.Input.LineCount() > 1 {
		height += 1 // visual gap between multiline textarea and the below-input block
	}
	height += m.inputBelowHeight()
	height += 1 // footer
	return height
}

// inputBelowHeight returns the number of lines reserved below the input box.
func (m *Model) inputBelowHeight() int {
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
func (m *Model) mainViewportHeight() int {
	vh := m.layout.Height - m.inputChromeHeight()
	if vh < 1 {
		return 1
	}
	return vh
}

func (m *Model) mainBodyView() string {
	if m.hasPendingChoiceCard() {
		m.syncChoiceViewport()
		return m.Viewport.View()
	}
	return ""
}

// primaryInputHeight returns the height of the active bottom text area (main input or offline paste).
func (m *Model) primaryInputHeight() int {
	if m.ChoiceCard.offlinePaste != nil {
		return m.ChoiceCard.offlinePaste.Paste.Height()
	}
	return m.Input.Height()
}

// primaryInputView renders the active bottom text area (main input or offline paste box).
func (m *Model) primaryInputView() string {
	if m.ChoiceCard.offlinePaste != nil {
		return m.ChoiceCard.offlinePaste.Paste.View()
	}
	return m.Input.View()
}

// primaryInputLineCount returns the line count of the active bottom text area.
func (m *Model) primaryInputLineCount() int {
	if m.ChoiceCard.offlinePaste != nil {
		return m.ChoiceCard.offlinePaste.Paste.LineCount()
	}
	return m.Input.LineCount()
}

func (m *Model) pendingChoiceContent() string {
	var b strings.Builder
	if m.ChoiceCard.offlinePaste != nil {
		m.appendOfflinePasteViewportContent(&b)
		return b.String()
	}
	m.appendApprovalViewportContent(&b)
	return b.String()
}

func (m *Model) printedTranscriptLineCount() int {
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

// bottomChromeReserveRows is the vertical space reserved for separator + input + below-input + footer.
// terminalWrappedRows(bottomBlock) can underestimate lipgloss/textarea layout vs the real terminal;
// inputChromeHeight() is the layout budget that must stay aligned with choice-card viewport math.
// Using the max avoids extra leading newlines above the chrome after long transcript (input appears too high).
func (m *Model) bottomChromeReserveRows(bottomBlock string) int {
	wrapped := terminalWrappedRows(bottomBlock, m.contentWidth())
	chrome := m.inputChromeHeight()
	if wrapped > chrome {
		return wrapped
	}
	return chrome
}

func (m *Model) normalModeTopPaddingLines(bottomBlock string) int {
	if m.layout.Height <= 0 {
		return 0
	}
	bottomReserve := m.bottomChromeReserveRows(bottomBlock)
	visiblePrinted := m.printedTranscriptLineCount()
	if visiblePrinted > m.layout.Height {
		visiblePrinted = m.layout.Height
	}
	// When transcript already uses (height - bottomReserve) or more rows, pad is zero: scrollback fills
	// the band above the chrome and the bottom block sits directly under it without spacer rows.
	pad := m.layout.Height - visiblePrinted - bottomReserve
	if pad < 0 {
		return 0
	}
	return pad
}

func finalizeUpdate(prevOverlayActive bool, m *Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.syncInputHeight()
	if !prevOverlayActive && m.Overlay.Active {
		return m, tea.Sequence(
			teaCmdForMsg(tea.EnterAltScreen()),
			teaCmdForMsg(tea.ClearScreen()),
			cmd,
		)
	}
	if prevOverlayActive && !m.Overlay.Active {
		printCmd := m.printTranscriptCmd(false)
		return m, tea.Sequence(
			teaCmdForMsg(tea.ExitAltScreen()),
			cmd,
			printCmd,
		)
	}
	return m, cmd
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
