package widget

import "github.com/charmbracelet/lipgloss"

// DefaultOverlayBoxMaxWidth keeps centered overlays comfortably readable on wide terminals
// without wasting too much horizontal space.
const DefaultOverlayBoxMaxWidth = 120

// RenderCenteredModal draws a titled modal with border and centers it in layoutW×layoutH cells.
// For very small terminals it returns an empty string (caller should fall back to base content).
func RenderCenteredModal(layoutW, layoutH, boxWMax int, title, innerContent string) string {
	if layoutW < 20 || layoutH < 6 {
		return ""
	}
	boxW := layoutW - 8
	if boxW > boxWMax {
		boxW = boxWMax
	}
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
	boxContent := overlayBoxStyle.Render(innerContent)
	titleBar := overlayTitleBarStyle.Render(title)
	box := titleBar + "\n" + boxContent
	overlayStyle := lipgloss.NewStyle().
		Width(layoutW).
		Height(layoutH).
		Align(lipgloss.Center, lipgloss.Center)
	return overlayStyle.Render(box)
}
