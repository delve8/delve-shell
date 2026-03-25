package widget

import "github.com/charmbracelet/lipgloss"

// TitleBarStatus selects which lipgloss style applies to the trailing status segment of the title line.
type TitleBarStatus int

const (
	TitleBarStatusIdle TitleBarStatus = iota
	TitleBarStatusRunning
	TitleBarStatusPendingApproval
	TitleBarStatusSuggest
	// TitleBarStatusOther renders autoRunPart and statusStr with Base only (default / unknown key).
	TitleBarStatusOther
)

// TitleLineStyles is injected from the ui package so colors stay in one place (styles.go).
type TitleLineStyles struct {
	Base          lipgloss.Style
	StatusIdle    lipgloss.Style
	StatusRunning lipgloss.Style
	StatusPending lipgloss.Style
	StatusSuggest lipgloss.Style
}

// RenderTitleLine renders the fixed header line: mode | auto-run | status, with status styled by st.
func RenderTitleLine(autoRunPart, statusStr string, st TitleBarStatus, s TitleLineStyles) string {
	base := s.Base.Render(autoRunPart)
	switch st {
	case TitleBarStatusIdle:
		return base + s.StatusIdle.Render(statusStr)
	case TitleBarStatusRunning:
		return base + s.StatusRunning.Render(statusStr)
	case TitleBarStatusPendingApproval:
		return base + s.StatusPending.Render(statusStr)
	case TitleBarStatusSuggest:
		return base + s.StatusSuggest.Render(statusStr)
	default:
		return s.Base.Render(autoRunPart + statusStr)
	}
}
