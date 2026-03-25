package ui

import "delve-shell/internal/ui/widget"

// overlayBoxMaxWidth is the max width of the overlay box so hint lines (e.g. "Up/Down to move... Esc to cancel.") do not wrap.
const overlayBoxMaxWidth = widget.DefaultOverlayBoxMaxWidth

// renderOverlay draws a centered modal box over the base content.
func (m Model) renderOverlay(base string) string {
	w := m.Layout.Width
	h := m.Layout.Height
	if w < 20 || h < 6 {
		return base
	}

	var content string
	for _, p := range overlayContentProviderChain.List() {
		if c, handled := p(m); handled {
			content = c
			break
		}
	}
	if content == "" {
		content = m.Overlay.Viewport.View()
	}

	out := widget.RenderCenteredModal(w, h, overlayBoxMaxWidth, m.Overlay.Title, content)
	if out == "" {
		return base
	}
	return out
}
