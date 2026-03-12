package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"delve-shell/internal/config"
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
			}
			if json.Unmarshal(ev.Payload, &p) != nil || p.Command == "" {
				continue
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
				Command   string `json:"command"`
				Stdout    string `json:"stdout"`
				Stderr    string `json:"stderr"`
				ExitCode  int    `json:"exit_code"`
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
	vh := m.Height - 8
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
		opts := getSlashOptionsForInput(inputVal, lang, m.CurrentSessionPath)
		vis := visibleSlashOptions(inputVal, opts)
		if len(vis) > 0 {
			out += "\n"
			hiIdx := m.SlashSuggestIndex
			if hiIdx >= len(vis) {
				hiIdx = 0
			}
			const maxSlashVisible = 8
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
			for i := start; i < end; i++ {
				vi := vis[i]
				opt := opts[vi]
				var line string
				var truncated bool
				if opt.Path != "" {
					line = "/sessions " + opt.Cmd
				} else {
					line = fmt.Sprintf("%-*s  %s", cmdWidth, opt.Cmd, opt.Desc)
				}
				if maxLineLen > 3 {
					r := []rune(line)
					if len(r) > maxLineLen-3 {
						line = string(r[:maxLineLen-3]) + "..."
						truncated = true
					}
				}
				if i == hiIdx {
					out += suggestHi.Render("   "+line) + "\n"
					// When selected and description was truncated, show full description below (multi-line if needed)
					if opt.Path == "" && opt.Desc != "" && truncated {
						indent := "   " + strings.Repeat(" ", cmdWidth) + "  "
						descRunes := []rune(opt.Desc)
						indentLen := 3 + cmdWidth + 2
						descW := maxLineLen - indentLen
						if descW < 20 {
							descW = 40
						}
						if descW > len(descRunes) {
							descW = len(descRunes)
						}
						if descW < 1 {
							descW = 1
						}
						for j := 0; j < len(descRunes); j += descW {
							endJ := j + descW
							if endJ > len(descRunes) {
								endJ = len(descRunes)
							}
							out += suggestStyle.Render(indent+string(descRunes[j:endJ])) + "\n"
						}
					}
				} else {
					out += suggestStyle.Render("   "+line) + "\n"
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

	// Build box content.
	var content string
	if m.ConfigLLMActive {
		lang := m.getLang()
		var b strings.Builder
		if m.ConfigLLMChecking {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyConfigLLMChecking)) + "\n\n")
		} else if m.ConfigLLMError != "" {
			b.WriteString(errStyle.Render(m.ConfigLLMError) + "\n\n")
		}
		b.WriteString(i18n.T(lang, i18n.KeyConfigLLMBaseURLLabel) + "\n")
		b.WriteString(m.ConfigLLMBaseURLInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyConfigLLMApiKeyLabel) + "\n")
		b.WriteString(m.ConfigLLMApiKeyInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyConfigLLMModelLabel) + "\n")
		b.WriteString(m.ConfigLLMModelInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxMessagesLabel) + "\n")
		b.WriteString(m.ConfigLLMMaxMessagesInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxCharsLabel) + "\n")
		b.WriteString(m.ConfigLLMMaxCharsInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(lang, i18n.KeyConfigLLMHint))
		content = b.String()
	} else if m.AddRemoteActive {
		var b strings.Builder
		if m.AddRemoteError != "" {
			b.WriteString(errStyle.Render(m.AddRemoteError) + "\n\n")
			if m.AddRemoteOfferOverwrite {
				b.WriteString("Press y to overwrite, or change host/username and try again.\n\n")
			}
		}
		b.WriteString("Add remote\n\n")
		b.WriteString("Host (address or host:port):\n")
		b.WriteString(m.AddRemoteHostInput.View())
		b.WriteString("\n\n")
		b.WriteString("Username:\n")
		b.WriteString(m.AddRemoteUserInput.View())
		b.WriteString("\n\n")
		b.WriteString("Name (optional):\n")
		b.WriteString(m.AddRemoteNameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Key path (optional):\n")
		b.WriteString(m.AddRemoteKeyInput.View())
		if m.AddRemoteFieldIndex == 3 && len(m.PathCompletionCandidates) > 0 {
			b.WriteString("\n\n")
			b.WriteString("Path completion (Up/Down select, Enter or Tab to pick):\n")
			for i, c := range m.PathCompletionCandidates {
				line := "  " + c
				if i == m.PathCompletionIndex {
					b.WriteString(suggestHi.Render(line) + "\n")
				} else {
					b.WriteString(suggestStyle.Render(line) + "\n")
				}
			}
		}
		b.WriteString("\n\n")
		b.WriteString("Up/Down to move between fields, Enter to save, Esc to cancel.")
		content = b.String()
	} else if m.RemoteAuthStep == "username" {
		var b strings.Builder
		if m.RemoteAuthError != "" {
			b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
		}
		b.WriteString("SSH auth for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n\n")
		b.WriteString("Username:\n")
		b.WriteString(m.RemoteAuthUsernameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Press Enter to continue, Esc to cancel.")
		content = b.String()
	} else if m.RemoteAuthStep == "choose" {
		var b strings.Builder
		if m.RemoteAuthError != "" {
			b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
		}
		b.WriteString("Choose authentication method:\n")
		b.WriteString("  1. Password\n")
		b.WriteString("  2. Key file (identity file)\n\n")
		b.WriteString("Press 1 or 2 to select, Esc to cancel.")
		content = b.String()
	} else if m.RemoteAuthStep == "password" {
		var b strings.Builder
		b.WriteString(m.OverlayContent)
		b.WriteString("\n\n")
		b.WriteString(m.RemoteAuthInput.View())
		content = b.String()
	} else if m.RemoteAuthStep == "identity" {
		var b strings.Builder
		b.WriteString(m.OverlayContent)
		b.WriteString("\n\n")
		b.WriteString(m.RemoteAuthInput.View())
		if len(m.PathCompletionCandidates) > 0 {
			b.WriteString("\n\n")
			b.WriteString("Path completion (Up/Down select, Enter or Tab to pick):\n")
			for i, c := range m.PathCompletionCandidates {
				line := "  " + c
				if i == m.PathCompletionIndex {
					b.WriteString(suggestHi.Render(line) + "\n")
				} else {
					b.WriteString(suggestStyle.Render(line) + "\n")
				}
			}
		}
		content = b.String()
	} else {
		// Generic overlay: scrollable viewport.
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
