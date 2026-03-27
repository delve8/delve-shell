package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

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

type FooterBarParts struct {
	Remote              string
	AutoRunFull         string
	AutoRunShort        string
	AutoRunReserveWidth int
	Status              string
	StatusReserveWidth  int
}

// RenderFooterBar renders a fixed footer/status line, preferring status first, then auto-run,
// then remote. It uses lightweight spacing between segments and center-truncates the remote
// segment when needed so both ends stay visible.
func RenderFooterBar(width int, parts FooterBarParts, st TitleBarStatus, s TitleLineStyles) string {
	if width < 1 {
		width = 1
	}

	statusText := strings.TrimSpace(parts.Status)
	autoText := strings.TrimSpace(parts.AutoRunFull)
	remoteText := strings.TrimSpace(parts.Remote)
	autoReserveW := parts.AutoRunReserveWidth
	if autoReserveW < runewidth.StringWidth(autoText) {
		autoReserveW = runewidth.StringWidth(autoText)
	}
	statusReserveW := parts.StatusReserveWidth
	if statusReserveW < runewidth.StringWidth(statusText) {
		statusReserveW = runewidth.StringWidth(statusText)
	}

	statusStyle := statusStyleFor(st, s)
	sep := s.Base.Render("        ")
	sepW := 8

	statusW := statusReserveW
	autoW := autoReserveW
	remoteW := runewidth.StringWidth(remoteText)

	renderStatus := func(text string) string {
		return renderExactWidth(statusStyle, text, statusReserveW)
	}
	renderAuto := func(text string) string {
		return renderExactWidth(s.Base, text, autoReserveW)
	}
	renderBase := func(text string) string {
		return s.Base.Render(text)
	}

	if statusW+sepW+autoW+sepW+remoteW <= width {
		return renderStatus(statusText) + sep + renderAuto(autoText) + sep + renderBase(remoteText)
	}

	remoteAvail := width - statusW - sepW - autoW - sepW
	if remoteAvail >= 1 {
		remoteText = truncateMiddle(remoteText, remoteAvail)
		return renderStatus(statusText) + sep + renderAuto(autoText) + sep + renderBase(remoteText)
	}

	if statusW+sepW+autoW <= width {
		return renderStatus(statusText) + sep + renderAuto(autoText)
	}

	if width <= statusW {
		return renderStatus(truncateMiddle(statusText, width))
	}
	return renderStatus(statusText)
}

// RenderTitleLine is kept as a thin compatibility wrapper for the older two-segment footer shape.
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

func statusStyleFor(st TitleBarStatus, s TitleLineStyles) lipgloss.Style {
	switch st {
	case TitleBarStatusIdle:
		return s.StatusIdle
	case TitleBarStatusRunning:
		return s.StatusRunning
	case TitleBarStatusPendingApproval:
		return s.StatusPending
	case TitleBarStatusSuggest:
		return s.StatusSuggest
	default:
		return s.Base
	}
}

func truncateMiddle(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth == 1 {
		return "…"
	}
	if maxWidth == 2 {
		return "…"
	}
	leftWidth := (maxWidth - 1) / 2
	rightWidth := maxWidth - 1 - leftWidth
	left := takePrefixByWidth(s, leftWidth)
	right := takeSuffixByWidth(s, rightWidth)
	return left + "…" + right
}

func renderExactWidth(style lipgloss.Style, text string, width int) string {
	if width <= 0 {
		return ""
	}
	textW := runewidth.StringWidth(text)
	if textW > width {
		text = truncateMiddle(text, width)
	} else if textW < width {
		text += strings.Repeat(" ", width-textW)
	}
	return style.Render(text)
}

func takePrefixByWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

func takeSuffixByWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	var out []rune
	w := 0
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if w+rw > maxWidth {
			break
		}
		out = append(out, runes[i])
		w += rw
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
