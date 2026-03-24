package ui

import "github.com/charmbracelet/bubbles/viewport"

// ClearSlashInput resets slash input-related UI state.
// It is exported so feature packages can implement exact slash handlers
// without depending on unexported ui internals.
func (m Model) ClearSlashInput() Model {
	return m.clearSlashInput()
}

// RefreshViewport rebuilds the view content and scrolls to bottom.
// This is used by exact slash handlers that need immediate UI feedback.
func (m Model) RefreshViewport() Model {
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// OpenOverlay opens a generic overlay and sets title/content.
func (m Model) OpenOverlay(title, content string) Model {
	m.Overlay.Active = true
	m.Overlay.Title = title
	m.Overlay.Content = content
	return m
}

// CloseOverlayVisual closes overlay chrome only.
// Feature-specific flags are still owned by each feature package.
func (m Model) CloseOverlayVisual() Model {
	m.Overlay.Active = false
	m.Overlay.Title = ""
	m.Overlay.Content = ""
	return m
}

// InitOverlayViewport initializes the generic overlay viewport from current layout.
func (m Model) InitOverlayViewport() Model {
	m.Overlay.Viewport = viewport.New(m.Layout.Width-minOverlayLayoutWidth, min(m.Layout.Height-minOverlayLayoutHeight, maxOverlayViewportHeight))
	m.Overlay.Viewport.SetContent(m.Overlay.Content)
	return m
}
