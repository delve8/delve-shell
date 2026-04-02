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
	TitleBarStatusExecuting
	TitleBarStatusRunning
	TitleBarStatusWaitingUserInput
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

// RenderFooterBar renders a fixed footer/status line: status first, optional middle segment (legacy: auto-run), then remote.
// When AutoRunFull, AutoRunShort are empty and AutoRunReserveWidth <= 0, only status and remote are shown (one separator).
// It uses lightweight spacing and center-truncates the remote segment when needed so both ends stay visible.
func RenderFooterBar(width int, parts FooterBarParts, st TitleBarStatus, s TitleLineStyles) string {
	if width < 1 {
		width = 1
	}

	statusText := strings.TrimSpace(parts.Status)
	autoFullText := strings.TrimSpace(parts.AutoRunFull)
	autoShortText := strings.TrimSpace(parts.AutoRunShort)
	remoteText := strings.TrimSpace(parts.Remote)
	omitAuto := autoFullText == "" && autoShortText == "" && parts.AutoRunReserveWidth <= 0

	statusReserveW := parts.StatusReserveWidth
	if statusReserveW < runewidth.StringWidth(statusText) {
		statusReserveW = runewidth.StringWidth(statusText)
	}

	autoText := autoFullText
	autoW := 0
	autoReserveW := 0
	if !omitAuto {
		autoReserveW = parts.AutoRunReserveWidth
		if autoReserveW < runewidth.StringWidth(autoText) {
			autoReserveW = runewidth.StringWidth(autoText)
		}
		autoW = autoReserveW
	}

	statusW := statusReserveW
	remoteW := runewidth.StringWidth(remoteText)

	statusStyle := statusStyleFor(st, s)
	const maxSepW = 8
	const minSepW = 2

	sepW := maxSepW
	if omitAuto {
		if total := statusW + sepW + remoteW; total > width {
			overflow := total - width
			sepW = max(minSepW, maxSepW-overflow)
		}
	} else {
		if total := statusW + sepW + autoW + sepW + remoteW; total > width {
			overflow := total - width
			sepW = max(minSepW, maxSepW-((overflow+1)/2))
		}
	}
	sep := s.Base.Render(strings.Repeat(" ", sepW))

	renderStatus := func(text string) string {
		return renderExactWidth(statusStyle, text, statusReserveW)
	}
	renderBase := func(text string) string {
		return s.Base.Render(text)
	}

	if !omitAuto && autoShortText != "" {
		shortW := runewidth.StringWidth(autoShortText)
		if statusW+sepW+autoW+sepW+remoteW > width && shortW < autoW {
			autoText = autoShortText
			autoW = shortW
		}
	}

	if omitAuto {
		if statusW+sepW+remoteW <= width {
			return renderStatus(statusText) + sep + renderBase(remoteText)
		}
		remoteAvail := width - statusW - sepW
		if remoteAvail >= 1 {
			remoteText = truncateMiddle(remoteText, remoteAvail)
			return renderStatus(statusText) + sep + renderBase(remoteText)
		}
		if width <= statusW {
			return renderStatus(truncateMiddle(statusText, width))
		}
		return renderStatus(statusText)
	}

	if statusW+sepW+autoW+sepW+remoteW <= width {
		return renderStatus(statusText) + sep + renderExactWidth(s.Base, autoText, autoW) + sep + renderBase(remoteText)
	}

	remoteAvail := width - statusW - sepW - autoW - sepW
	if remoteAvail >= 1 {
		remoteText = truncateMiddle(remoteText, remoteAvail)
		return renderStatus(statusText) + sep + renderExactWidth(s.Base, autoText, autoW) + sep + renderBase(remoteText)
	}

	if statusW+sepW+autoW <= width {
		return renderStatus(statusText) + sep + renderExactWidth(s.Base, autoText, autoW)
	}

	if width <= statusW {
		return renderStatus(truncateMiddle(statusText, width))
	}
	return renderStatus(statusText)
}

func statusStyleFor(st TitleBarStatus, s TitleLineStyles) lipgloss.Style {
	switch st {
	case TitleBarStatusIdle:
		return s.StatusIdle
	case TitleBarStatusExecuting:
		return s.StatusRunning
	case TitleBarStatusRunning:
		return s.StatusRunning
	case TitleBarStatusWaitingUserInput:
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
