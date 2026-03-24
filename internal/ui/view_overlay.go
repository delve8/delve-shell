package ui

import "github.com/charmbracelet/lipgloss"

// overlayBoxMaxWidth is the max width of the overlay box so hint lines (e.g. "Up/Down to move... Esc to cancel.") do not wrap.
const overlayBoxMaxWidth = 70

// renderOverlay draws a centered modal box over the base content.
func (m Model) renderOverlay(base string) string {
	w := m.Width
	h := m.Height
	if w < 20 || h < 6 {
		return base
	}

	// Box dimensions (smaller, centered).
	boxW := w - 8
	if boxW > overlayBoxMaxWidth {
		boxW = overlayBoxMaxWidth
	}

	// Build box content: feature packages register overlay body builders.
	var content string
	for _, p := range overlayContentProviders {
		if c, handled := p(m); handled {
			content = c
			break
		}
	}
	if content == "" {
		// Generic overlay: scrollable viewport (e.g. /help).
		content = m.Overlay.Viewport.View()
	}

	// Border styles.
	overlayBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1).
		Width(boxW - 2)
	overlayTitleBarStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("12")).
		Padding(0, 1).
		Width(boxW - 2).
		Align(lipgloss.Center)

	// Compose box with title.
	boxContent := overlayBoxStyle.Render(content)
	titleBar := overlayTitleBarStyle.Render(m.Overlay.Title)
	box := titleBar + "\n" + boxContent

	// Use lipgloss.Place to center the overlay on a blank background.
	overlayStyle := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(box)
}
