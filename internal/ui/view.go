package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// choiceOption is one line in the choice menu (num 1-based, label for display).
type choiceOption struct {
	Num   int
	Label string
}

// choiceCount returns the number of options when in a choice state (approval 2 or 3, sensitive 3, or session list N).
func choiceCount(m Model) int {
	switch {
	case m.Pending != nil:
		if m.GetAllowlistAutoRun != nil && !m.GetAllowlistAutoRun() {
			return 3 // Run, Copy, Dismiss
		}
		return 2 // Run, Reject
	case m.PendingSensitive != nil:
		return 3
	default:
		return 0
	}
}

// getChoiceOptions returns the option list for the current choice state (approval 2 or 3 options / sensitive / session list).
func getChoiceOptions(m Model, lang string) []choiceOption {
	switch {
	case m.Pending != nil:
		if m.GetAllowlistAutoRun != nil && !m.GetAllowlistAutoRun() {
			return []choiceOption{
				{1, i18n.T(lang, i18n.KeyChoiceApprove)},
				{2, i18n.T(lang, i18n.KeyChoiceCopy)},
				{3, i18n.T(lang, i18n.KeyChoiceDismiss)},
			}
		}
		return []choiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceApprove)},
			{2, i18n.T(lang, i18n.KeyChoiceReject)},
		}
	case m.PendingSensitive != nil:
		return []choiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceRefuse)},
			{2, i18n.T(lang, i18n.KeyChoiceRunStore)},
			{3, i18n.T(lang, i18n.KeyChoiceRunNoStore)},
		}
	default:
		return nil
	}
}

// syncInputPlaceholder sets the input placeholder to selection hint (1/2 or 1/2/3) when waiting for choice, else normal placeholder.
func (m *Model) syncInputPlaceholder() {
	lang := m.getLang()
	switch {
	case m.Pending != nil:
		if m.GetAllowlistAutoRun != nil && !m.GetAllowlistAutoRun() {
			m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintApproveThree)
		} else {
			m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintApprove)
		}
	case m.PendingSensitive != nil:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintSensitive)
	default:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyPlaceholderInput)
	}
}

// statusKey returns the i18n key for current state: idle, running, or pending approval.
func (m Model) statusKey() string {
	if m.Pending != nil || m.PendingSensitive != nil {
		return i18n.KeyStatusPendingApproval
	}
	if m.WaitingForAI {
		return i18n.KeyStatusRunning
	}
	return i18n.KeyStatusIdle
}

// titleLine returns the fixed title (Remote + Auto-run + status) for display above the viewport; does not scroll.
func (m Model) titleLine() string {
	lang := m.getLang()
	remotePart := "Local"
	if m.RemoteActive {
		if m.RemoteLabel != "" {
			remotePart = "Remote " + m.RemoteLabel
		} else {
			remotePart = "Remote"
		}
	}
	autoRunStr := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if m.GetAllowlistAutoRun != nil && !m.GetAllowlistAutoRun() {
		autoRunStr = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	autoRunPart := remotePart + " | " + i18n.T(lang, i18n.KeyAutoRunLabel) + autoRunStr + " | "
	statusStr := i18n.T(lang, m.statusKey())
	// Render status with different colors for idle, running, pending, suggest.
	switch m.statusKey() {
	case i18n.KeyStatusIdle:
		return titleStyle.Render(autoRunPart) + statusIdleStyle.Render(statusStr)
	case i18n.KeyStatusRunning:
		return titleStyle.Render(autoRunPart) + statusRunningStyle.Render(statusStr)
	case i18n.KeyStatusPendingApproval:
		return titleStyle.Render(autoRunPart) + pendingActionStyle.Render(statusStr)
	case i18n.KeyStatusSuggest:
		return titleStyle.Render(autoRunPart) + suggestStyle.Render(statusStr)
	default:
		return titleStyle.Render(autoRunPart + statusStr)
	}
}

const maxSessionHistoryEvents = 500

// sessionEventsToMessages converts history events to the same display lines used in the live conversation (User:, AI:, Run:, result).
// width is used to soft-wrap long command lines; if <= 0, no wrapping is applied.
func sessionEventsToMessages(events []history.Event, lang string, width int) []string {
	var out []string
	for _, ev := range events {
		switch ev.Type {
		case "user_input":
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Text != "" {
				line := i18n.T(lang, i18n.KeyUserLabel) + p.Text
				if width > 0 {
					line = wrapString(line, width)
				}
				out = append(out, line)
				out = append(out, "") // blank line before command or AI reply
			}
		case "llm_response":
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Reply != "" {
				line := i18n.T(lang, i18n.KeyAILabel) + p.Reply
				if width > 0 {
					line = wrapString(line, width)
				}
				out = append(out, line)
				sepW := width
				if sepW <= 0 {
					sepW = 40
				}
				out = append(out, separatorStyle.Render(strings.Repeat("─", sepW)))
			}
		case "command":
			var p struct {
				Command   string `json:"command"`
				Approved  bool   `json:"approved"`
				Suggested bool   `json:"suggested"`
				Kind      string `json:"kind"`
				SkillName string `json:"skill_name"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil || p.Command == "" {
				continue
			}
			if p.Kind == "skill" && strings.TrimSpace(p.SkillName) != "" {
				skillLine := i18n.Tf(lang, i18n.KeySkillLine, strings.TrimSpace(p.SkillName))
				if width > 0 {
					skillLine = wrapString(skillLine, width)
				}
				out = append(out, suggestStyle.Render(skillLine))
			}
			tag := i18n.T(lang, i18n.KeyRunTagApproved)
			if p.Suggested {
				tag = i18n.T(lang, i18n.KeyRunTagSuggested)
			}
			line := i18n.T(lang, i18n.KeyRunLabel) + p.Command + " (" + tag + ")"
			if width > 0 {
				line = wrapString(line, width)
			}
			out = append(out, execStyle.Render(line))
		case "command_result":
			var p struct {
				Command  string `json:"command"`
				Stdout   string `json:"stdout"`
				Stderr   string `json:"stderr"`
				ExitCode int    `json:"exit_code"`
			}
			if json.Unmarshal(ev.Payload, &p) != nil {
				continue
			}
			result := strings.TrimSpace(p.Stdout)
			if p.Stderr != "" {
				if result != "" {
					result += "\n"
				}
				result += strings.TrimSpace(p.Stderr)
			}
			if result != "" {
				if width > 0 {
					result = wrapString(result, width)
				}
				out = append(out, resultStyle.Render(result))
			}
			out = append(out, "") // blank line after command output
		}
	}
	return out
}

// wrapString breaks s into lines of at most maxWidth terminal cells (soft wrap). Prefers breaking at spaces. If maxWidth <= 0, returns s unchanged.
func wrapString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	start := 0
	cellWidth := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			b.WriteString(string(runes[start : i+1]))
			start = i + 1
			cellWidth = 0
			continue
		}
		w := runewidth.RuneWidth(r)
		if cellWidth+w > maxWidth && cellWidth > 0 {
			// Prefer break at last space in this segment
			breakAt := i
			for j := i - 1; j >= start; j-- {
				if unicode.IsSpace(runes[j]) {
					breakAt = j + 1
					break
				}
			}
			b.WriteString(string(runes[start:breakAt]))
			b.WriteByte('\n')
			start = breakAt
			for start < len(runes) && unicode.IsSpace(runes[start]) {
				start++
			}
			cellWidth = 0
			i = start - 1
			continue
		}
		cellWidth += w
	}
	if start < len(runes) {
		b.WriteString(string(runes[start:]))
	}
	return b.String()
}

// buildContent returns the scrollable viewport content (messages + pending/suggest cards); title is rendered in View().
func (m Model) buildContent() string {
	lang := m.getLang()
	var b strings.Builder
	for _, line := range m.Messages {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.PendingSensitive != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)) + "\n")
		w := m.Width
		if w <= 0 {
			w = 80
		}
		b.WriteString(execStyle.Render(wrapString(m.PendingSensitive.Command, w)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice1)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice2)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice3)))
		return b.String()
	}
	if m.Pending != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)) + "\n")
		w := m.Width
		if w <= 0 {
			w = 80
		}
		if sn := strings.TrimSpace(m.Pending.SkillName); sn != "" {
			line := i18n.Tf(lang, i18n.KeySkillLine, sn)
			b.WriteString(suggestStyle.Render(wrapString(line, w)) + "\n")
		}
		switch m.Pending.RiskLevel {
		case "read_only":
			line := "[" + i18n.T(lang, i18n.KeyRiskReadOnly) + "] " + m.Pending.Command
			b.WriteString(riskReadOnlyStyle.Render(wrapString(line, w)) + "\n")
		case "low":
			line := "[" + i18n.T(lang, i18n.KeyRiskLow) + "] " + m.Pending.Command
			b.WriteString(riskLowStyle.Render(wrapString(line, w)) + "\n")
		case "high":
			line := "[" + i18n.T(lang, i18n.KeyRiskHigh) + "] " + m.Pending.Command
			b.WriteString(riskHighStyle.Render(wrapString(line, w)) + "\n")
		default:
			b.WriteString(execStyle.Render(wrapString(m.Pending.Command, w)) + "\n")
		}
		if m.Pending.Summary != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalSummary)+" "+m.Pending.Summary) + "\n")
		}
		if m.Pending.Reason != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason) + "\n")
		}
		return b.String()
	}
	return b.String()
}

// View implements tea.Model.
func (m Model) View() string {
	lang := m.getLang()
	sepW := m.Width
	if sepW <= 0 {
		sepW = 40
	}
	sepLine := separatorStyle.Render(strings.Repeat("─", sepW))
	header := m.titleLine() + "\n" + sepLine + "\n"

	inChoice := m.Pending != nil || m.PendingSensitive != nil
	if m.Height <= 4 {
		out := header + m.buildContent() + "\n" + m.Input.View()
		if m.WaitingForAI && !inChoice {
			out += "\n" + suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel))
		}
		return out
	}
	// Base viewport height: leave room for header, separator, input line, and slash/choice dropdown (the two lines at bottom are for input + suggestions).
	vh := m.Height - 10
	if vh < 1 {
		vh = 1
	}
	m.Viewport.Width = m.Width
	m.Viewport.Height = vh
	out := header
	out += m.Viewport.View()
	out += "\n" + sepLine + "\n"
	out += m.Input.View()
	inputVal := m.Input.Value()
	if inChoice {
		opts := getChoiceOptions(m, lang)
		if len(opts) > 0 {
			out += "\n"
			for i, o := range opts {
				line := fmt.Sprintf("%d  %s", o.Num, o.Label)
				if i == m.ChoiceIndex {
					out += suggestHi.Render(" "+line) + "\n"
				} else {
					out += suggestStyle.Render(" "+line) + "\n"
				}
			}
		}
	} else if strings.HasPrefix(inputVal, "/") {
		opts := getSlashOptionsForInput(inputVal, lang, m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
		vis := visibleSlashOptions(inputVal, opts)
		if len(vis) > 0 {
			out += "\n"
			hiIdx := m.SlashSuggestIndex
			if hiIdx >= len(vis) {
				hiIdx = 0
			}
			// Limit number of visible slash suggestions so the fixed header/title
			// line is not pushed out of the terminal viewport when the list is long.
			const maxSlashVisible = 4
			start := 0
			if len(vis) > maxSlashVisible {
				start = hiIdx - maxSlashVisible/2
				if start < 0 {
					start = 0
				}
				if start+maxSlashVisible > len(vis) {
					start = len(vis) - maxSlashVisible
				}
			}
			end := start + maxSlashVisible
			if end > len(vis) {
				end = len(vis)
			}
			// Align descriptions: use same command column width for all visible Cmd+Desc rows.
			cmdWidth := 0
			for i := start; i < end; i++ {
				o := opts[vis[i]]
				if o.Path == "" && len(o.Cmd) > cmdWidth {
					cmdWidth = len(o.Cmd)
				}
			}
			const minCmdWidth = 12
			if cmdWidth < minCmdWidth {
				cmdWidth = minCmdWidth
			}
			maxLineLen := 0
			if m.Width > 4 {
				maxLineLen = m.Width - 4 // leave margin for prefix and avoid wrap
			}
			// remainingLines is the total number of lines we can use for the dropdown (including wrapped descriptions).
			remainingLines := maxSlashVisible
			for i := start; i < end && remainingLines > 0; i++ {
				vi := vis[i]
				opt := opts[vi]

				cmdText := opt.Cmd
				if opt.Path != "" {
					cmdText = "/sessions " + opt.Cmd
				}

				// When there is a description and width budget, show Cmd + first chunk of Desc on the first line,
				// then wrap the remaining Desc onto following lines within remainingLines.
				if opt.Path == "" && opt.Desc != "" && maxLineLen > 0 {
					descRunes := []rune(opt.Desc)
					// Visible width for the first line: "   " + cmdWidth + "  " + descFirst
					prefixRunes := 3 + cmdWidth + 2
					descFirstW := maxLineLen - prefixRunes
					if descFirstW < 8 {
						// Not enough room for inline desc; fall back to command-only line and use wrapped lines below.
						line := fmt.Sprintf("%-*s", cmdWidth, cmdText)
						if i == hiIdx {
							out += suggestHi.Render("   "+line) + "\n"
						} else {
							out += suggestStyle.Render("   "+line) + "\n"
						}
						remainingLines--
						if remainingLines <= 0 {
							break
						}
						// Now wrap full desc on following lines.
						indent := "   " + strings.Repeat(" ", cmdWidth) + "  "
						indentLen := len([]rune(indent))
						descW := maxLineLen - indentLen
						if descW < 10 {
							descW = 10
						}
						if descW > len(descRunes) {
							descW = len(descRunes)
						}
						if descW < 1 {
							descW = 1
						}
						for j := 0; j < len(descRunes) && remainingLines > 0; j += descW {
							endJ := j + descW
							if endJ > len(descRunes) {
								endJ = len(descRunes)
							}
							out += suggestStyle.Render(indent+string(descRunes[j:endJ])) + "\n"
							remainingLines--
						}
						continue
					}

					if descFirstW > len(descRunes) {
						descFirstW = len(descRunes)
					}
					firstChunk := string(descRunes[:descFirstW])
					line := fmt.Sprintf("%-*s  %s", cmdWidth, cmdText, firstChunk)
					if i == hiIdx {
						out += suggestHi.Render("   "+line) + "\n"
					} else {
						out += suggestStyle.Render("   "+line) + "\n"
					}
					remainingLines--
					if remainingLines <= 0 {
						break
					}

					// Wrap remaining desc onto following lines.
					rest := descRunes[descFirstW:]
					if len(rest) > 0 {
						indent := "   " + strings.Repeat(" ", cmdWidth) + "  "
						indentLen := len([]rune(indent))
						descW := maxLineLen - indentLen
						if descW < 10 {
							descW = 10
						}
						if descW > len(rest) {
							descW = len(rest)
						}
						if descW < 1 {
							descW = 1
						}
						for j := 0; j < len(rest) && remainingLines > 0; j += descW {
							endJ := j + descW
							if endJ > len(rest) {
								endJ = len(rest)
							}
							out += suggestStyle.Render(indent+string(rest[j:endJ])) + "\n"
							remainingLines--
						}
					}
				} else {
					// No description or no width budget: show command only.
					line := fmt.Sprintf("%-*s", cmdWidth, cmdText)
					if i == hiIdx {
						out += suggestHi.Render("   "+line) + "\n"
					} else {
						out += suggestStyle.Render("   "+line) + "\n"
					}
					remainingLines--
				}
			}
		}
	}
	if m.WaitingForAI && !inChoice {
		out += "\n"
		out += suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel))
	}

	// Render overlay on top if active.
	if m.OverlayActive {
		out = m.renderOverlay(out)
	}
	return out
}

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
	boxH := 10
	if boxH > h-4 {
		boxH = h - 4
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
		content = m.OverlayViewport.View()
	}

	// Border styles.
	overlayBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1).
		Width(boxW - 2)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("12")).
		Padding(0, 1).
		Width(boxW - 2).
		Align(lipgloss.Center)

	// Compose box with title.
	boxContent := overlayBoxStyle.Render(content)
	titleBar := titleStyle.Render(m.OverlayTitle)
	box := titleBar + "\n" + boxContent

	// Use lipgloss.Place to center the overlay on a blank background.
	overlayStyle := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center)

	// Clear the overlay area and place the box in center.
	return overlayStyle.Render(box)
}

// appendSuggestedLine appends the run line and copy hint for a suggested command (when dismissing the card).
func (m *Model) appendSuggestedLine(command, lang string) {
	tag := i18n.T(lang, i18n.KeyRunTagSuggested)
	line := i18n.T(lang, i18n.KeyRunLabel) + command + " (" + tag + ")"
	w := m.Width
	if w <= 0 {
		w = 80
	}
	m.Messages = append(m.Messages, execStyle.Render(wrapString(line, w)))
	m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopyHint)))
}
