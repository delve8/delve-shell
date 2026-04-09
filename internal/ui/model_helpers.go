package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
)

const (
	minInputLayoutWidth     = 4
	minContentWidthFallback = 80
	minOverlayLayoutHeight  = 6
	inputTextareaMinHeight  = 1
	inputTextareaMaxHeight  = 5
	// inputBelowStableRows: fixed lines below the input so the separator above the input and the footer
	// stay in a stable vertical band across idle / processing / slash-open (padded with blanks when needed).
	inputBelowStableRows   = 5
	maxInputHistoryEntries = 200
	// maxFullTranscriptReplayLines caps clear+replay work so overlay close does not become pathological
	// on very long histories. Older transcript stays in logical history; only replay is capped.
	maxFullTranscriptReplayLines = 100000
	// transcriptBulkPrintChunkLines groups multiple transcript rows into one tea.Println during replay.
	// Bubble Tea still splits by '\n', but this avoids one tea.Msg per logical transcript line.
	transcriptBulkPrintChunkLines = 512
	// shortSeparatorMaxWidth keeps transcript/chrome separators visually stable on resize instead of
	// drawing nearly full-width rules that can look messy near the terminal edge.
	shortSeparatorMaxWidth = 48
)

// ReadModel provides host-derived read-only state needed by UI rendering and local decisions.
type ReadModel interface {
	TakeOpenConfigModelOnFirstLayout() bool
	OfflineExecutionMode() bool
	// InitialRemoteFooter mirrors host Runtime when a new tea.Program starts (e.g. after /bash).
	// When inactive, label and offline are ignored by the UI footer.
	InitialRemoteFooter() (active bool, label string, offline bool)
}

func (m *Model) takeOpenConfigModelOnFirstLayout() bool {
	if m.ReadModel == nil {
		return false
	}
	return m.ReadModel.TakeOpenConfigModelOnFirstLayout()
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

// TranscriptLines returns a copy of the current transcript lines (printed via tea.Println + padding alignment).
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
		m.screenTranscriptStart = 0
		m.screenPrefixRows = 0
		m.recenterStartupTitleOnce = false
		return
	}
	out := make([]string, len(lines))
	copy(out, lines)
	m.messages = out
	m.screenTranscriptStart = 0
	m.screenPrefixRows = 0
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

func clearScrollbackCmd() tea.Cmd {
	// Best-effort: many terminals honor CSI 3J to clear scrollback, but some multiplexers/IDE
	// terminals ignore it. Follow with normal clear+replay so the current screen is deterministic.
	return tea.Printf("%s%s", ansi.EraseEntireDisplay, ansi.CursorHomePosition)
}

func (m *Model) printTranscriptCmd(clearFirst bool) tea.Cmd {
	return m.printTranscriptFromCmd(m.printedMessages, clearFirst)
}

func (m *Model) printTranscriptFromCmd(start int, clearFirst bool) tea.Cmd {
	return m.printTranscriptWithPrefixCmd(start, clearFirst, nil)
}

func (m *Model) printTranscriptWithPrefixCmd(start int, clearFirst bool, prefix []string) tea.Cmd {
	if m.Overlay.Active {
		return nil
	}
	end := len(m.messages)
	if start < 0 {
		start = 0
	}
	if start > end {
		start = end
	}
	if clearFirst {
		start = recentTranscriptReplayStart(start, end)
	}
	if !clearFirst && start >= end {
		return nil
	}
	cmds := make([]tea.Cmd, 0, (end-start)/max(1, transcriptBulkPrintChunkLines)+2)
	if clearFirst {
		cmds = append(cmds, teaCmdForMsg(tea.ClearScreen()))
		m.screenTranscriptStart = start
		m.screenPrefixRows = renderedLineRows(prefix, m.contentWidth())
	} else if len(prefix) > 0 {
		m.screenPrefixRows += renderedLineRows(prefix, m.contentWidth())
	}
	for _, line := range prefix {
		cmds = append(cmds, tea.Println(line))
	}
	for i := start; i < end; i += transcriptBulkPrintChunkLines {
		j := i + transcriptBulkPrintChunkLines
		if j > end {
			j = end
		}
		cmds = append(cmds, tea.Println(strings.Join(m.messages[i:j], "\n")))
	}
	// Sync printed count before async cmds run: a second WindowSize (or other Update) can arrive
	// after this return but before transcriptPrintedMsg; without this, the same lines are enqueued twice.
	m.printedMessages = end
	cmds = append(cmds, func() tea.Msg {
		return transcriptPrintedMsg{upTo: end}
	})
	return tea.Sequence(cmds...)
}

func recentTranscriptReplayStart(start, end int) int {
	if start < 0 {
		start = 0
	}
	if end < start {
		return start
	}
	if maxFullTranscriptReplayLines > 0 {
		floor := end - maxFullTranscriptReplayLines
		if floor > start {
			start = floor
		}
	}
	return start
}

func renderedLineRows(lines []string, width int) int {
	total := 0
	for _, line := range lines {
		total += terminalWrappedRows(line, width)
	}
	return total
}

func (m *Model) replayTruncatedNoticeLines(start int) []string {
	if start <= 0 {
		return nil
	}
	msg := hintStyle.Render(textwrap.WrapString(
		i18n.Tf(i18n.KeyTranscriptReplayTruncatedNotice, maxFullTranscriptReplayLines),
		m.contentWidth(),
	))
	return []string{
		renderShortSeparator(m.contentWidth()),
		msg,
		renderShortSeparator(m.contentWidth()),
	}
}

func (m *Model) withTranscriptReplaced(lines []string) {
	m.WithTranscriptLines(lines)
	m.printedMessages = 0
}

// OpenOverlayFeature opens a feature-owned overlay and records its active key.
func (m *Model) OpenOverlayFeature(key, title, content string) {
	m.openOverlayFeature(key, title, content, "")
}

// openMarkdownScrollOverlay opens the help-style overlay: Content is rendered from Markdown on each layout refresh.
func (m *Model) openMarkdownScrollOverlay(title, markdownSource, footer string) {
	m.Overlay.Active = true
	m.Overlay.Key = ""
	m.Overlay.Title = title
	m.Overlay.MarkdownSource = markdownSource
	m.Overlay.Footer = footer
	m.InitOverlayViewport()
}

// openOverlayFeature sets optional Footer: fixed hint lines below the scroll viewport (not part of scrolled text).
func (m *Model) openOverlayFeature(key, title, content, footer string) {
	m.Overlay.Active = true
	m.Overlay.Key = key
	m.Overlay.Title = title
	m.Overlay.Content = content
	m.Overlay.Footer = footer
	m.Overlay.MarkdownSource = ""
}

// CloseOverlayVisual closes overlay chrome only.
// Feature-specific flags are still owned by each feature package.
func (m *Model) CloseOverlayVisual() {
	m.Overlay.Active = false
	m.Overlay.Key = ""
	m.Overlay.Title = ""
	m.Overlay.Content = ""
	m.Overlay.Footer = ""
	m.Overlay.MarkdownSource = ""
}

// overlayFixedBelowViewportLineCount is rows below the scroll area: dim separator + footer hint (plain line count).
func overlayFixedBelowViewportLineCount(footer string) int {
	if footer == "" {
		return 0
	}
	return 1 + strings.Count(footer, "\n") + 1
}

func overlayViewportHeight(layoutH int, footer string) int {
	baseH := layoutH - minOverlayLayoutHeight
	if n := overlayFixedBelowViewportLineCount(footer); n > 0 {
		baseH -= n
	}
	if baseH < 3 {
		return 3
	}
	return baseH
}

// InitOverlayViewport initializes the generic overlay viewport from current layout.
func (m *Model) InitOverlayViewport() {
	if m.Overlay.MarkdownSource != "" {
		inner := overlayInnerWidth(m.layout.Width)
		m.Overlay.Content = RenderHelpMarkdown(m.Overlay.MarkdownSource, inner)
	}
	m.Overlay.Viewport = viewport.New(overlayInnerWidth(m.layout.Width), overlayViewportHeight(m.layout.Height, m.Overlay.Footer))
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

// inputChromeHeight returns the total number of lines in the bottom chrome (separator through footer).
func (m *Model) inputChromeHeight() int {
	height := m.execStreamPreviewReserveRows()
	height += 1 // separator above input
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

func (m *Model) printedTranscriptLineCount() int {
	if m.printedMessages <= 0 || len(m.messages) == 0 {
		return m.screenPrefixRows
	}
	limit := m.printedMessages
	if limit > len(m.messages) {
		limit = len(m.messages)
	}
	start := m.screenTranscriptStart
	if start < 0 {
		start = 0
	}
	if start > limit {
		start = limit
	}
	total := m.screenPrefixRows
	for _, line := range m.messages[start:limit] {
		total += terminalWrappedRows(line, m.contentWidth())
	}
	return total
}

// bottomChromeReserveRows is the vertical space reserved for separator + input + below-input + footer.
// terminalWrappedRows(bottomBlock) can underestimate lipgloss/textarea layout vs the real terminal;
// inputChromeHeight() is the layout budget that must stay aligned with bottom-chrome line accounting.
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
		// The main screen restored by ExitAltScreen can be stale relative to the current model.
		// Replay the recent transcript tail from the top; older history remains in logical transcript
		// but is intentionally not redrawn to keep replay work bounded.
		start := recentTranscriptReplayStart(0, len(m.messages))
		printCmd := m.printTranscriptWithPrefixCmd(start, true, m.replayTruncatedNoticeLines(start))
		return m, tea.Sequence(
			teaCmdForMsg(tea.ExitAltScreen()),
			// Reset terminal mouse tracking after leaving the alt-screen overlay (viewport may have
			// left modes like SGR mouse enabled on some emulators after auth / multi-field dialogs).
			teaCmdForMsg(tea.DisableMouse()),
			clearScrollbackCmd(),
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

func renderShortSeparator(width int) string {
	if width < 1 {
		width = 1
	}
	if width > 1 {
		width--
	}
	half := width / 2
	if half < 1 {
		half = 1
	}
	if width > half {
		width = half
	}
	if width > shortSeparatorMaxWidth {
		width = shortSeparatorMaxWidth
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
	// matching tea.Println("") and the trailing "" appended after each user transcript block.
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
