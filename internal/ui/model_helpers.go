package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
)

const (
	minInputLayoutWidth      = 4
	minContentWidthFallback  = 80
	mainViewportPadding      = 10
	minOverlayLayoutWidth    = 4
	minOverlayLayoutHeight   = 6
	maxOverlayViewportHeight = 20
)

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

// mainViewportHeight returns the viewport height used by main content.
func (m Model) mainViewportHeight() int {
	vh := m.layout.Height - mainViewportPadding
	if vh < 1 {
		return 1
	}
	return vh
}

// renderSeparator returns a horizontal separator with provided width.
func renderSeparator(width int) string {
	if width < 1 {
		width = 1
	}
	return separatorStyle.Render(strings.Repeat("─", width))
}
