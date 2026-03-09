package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

// choiceOption is one line in the choice menu (num 1-based, label for display).
type choiceOption struct {
	Num   int
	Label string
}

// choiceCount returns the number of options when in a choice state (approval 2, sensitive 3, suggest 2, or session list N).
func choiceCount(m Model) int {
	switch {
	case m.Pending != nil:
		return 2
	case m.PendingSensitive != nil:
		return 3
	case m.PendingSuggested != nil:
		return 2
	default:
		return 0
	}
}

// getChoiceOptions returns the option list for the current choice state (approval / sensitive / suggest).
func getChoiceOptions(m Model, lang string) []choiceOption {
	switch {
	case m.Pending != nil:
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
	case m.PendingSuggested != nil:
		return []choiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceCopy)},
			{2, i18n.T(lang, i18n.KeyChoiceDismiss)},
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
		m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintApprove)
	case m.PendingSensitive != nil:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintSensitive)
	case m.PendingSuggested != nil:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintSuggest)
	default:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyPlaceholderInput)
	}
}

// statusKey returns the i18n key for current state: idle, running, pending approval, or suggest card.
func (m Model) statusKey() string {
	if m.Pending != nil || m.PendingSensitive != nil {
		return i18n.KeyStatusPendingApproval
	}
	if m.PendingSuggested != nil {
		return i18n.KeyStatusSuggest
	}
	if m.WaitingForAI {
		return i18n.KeyStatusRunning
	}
	return i18n.KeyStatusIdle
}

// titleLine returns the fixed title (mode + status) for display above the viewport; does not scroll.
// When pending, status and operation hint (1=approve/2=reject or 1/2/3) are rendered with pendingActionStyle on the same line.
func (m Model) titleLine() string {
	lang := m.getLang()
	modeStr := "run"
	if m.GetMode != nil {
		modeStr = m.GetMode()
	}
	modePart := i18n.T(lang, i18n.KeyModeLabel) + ": " + modeStr + " | "
	statusStr := i18n.T(lang, m.statusKey())
	if m.Pending != nil {
		hint := i18n.T(lang, i18n.KeyApproveYN)
		return titleStyle.Render(modePart) + pendingActionStyle.Render(statusStr+"  "+hint)
	}
	if m.PendingSensitive != nil {
		hint := i18n.T(lang, i18n.KeySensitivePressKey)
		return titleStyle.Render(modePart) + pendingActionStyle.Render(statusStr+"  "+hint)
	}
	if m.PendingSuggested != nil {
		hint := i18n.T(lang, i18n.KeySuggestedCardHint)
		return titleStyle.Render(modePart) + pendingActionStyle.Render(statusStr+"  "+hint)
	}
	// Idle and running: render status in a more prominent color
	switch m.statusKey() {
	case i18n.KeyStatusIdle:
		return titleStyle.Render(modePart) + statusIdleStyle.Render(statusStr)
	case i18n.KeyStatusRunning:
		return titleStyle.Render(modePart) + statusRunningStyle.Render(statusStr)
	default:
		return titleStyle.Render(modePart + statusStr)
	}
}

const maxSessionHistoryEvents = 500

// sessionEventsToMessages converts history events to the same display lines used in the live conversation (User:, AI:, Run:, result).
func sessionEventsToMessages(events []history.Event, lang string) []string {
	var out []string
	for _, ev := range events {
		switch ev.Type {
		case "user_input":
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Text != "" {
				out = append(out, i18n.T(lang, i18n.KeyUserLabel)+p.Text)
			}
		case "llm_response":
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Reply != "" {
				out = append(out, i18n.T(lang, i18n.KeyAILabel)+p.Reply)
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
			out = append(out, execStyle.Render(i18n.T(lang, i18n.KeyRunLabel)+p.Command+" ("+tag+")"))
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
				out = append(out, resultStyle.Render(result))
			}
		}
	}
	return out
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
		b.WriteString(execStyle.Render(m.PendingSensitive.Command) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice1)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice2)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice3)))
		return b.String()
	}
	if m.Pending != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)) + "\n")
		switch m.Pending.RiskLevel {
		case "read_only":
			b.WriteString(riskReadOnlyStyle.Render("["+i18n.T(lang, i18n.KeyRiskReadOnly)+"] ") + m.Pending.Command + "\n")
		case "low":
			b.WriteString(riskLowStyle.Render("["+i18n.T(lang, i18n.KeyRiskLow)+"] ") + m.Pending.Command + "\n")
		case "high":
			b.WriteString(riskHighStyle.Render("["+i18n.T(lang, i18n.KeyRiskHigh)+"] ") + m.Pending.Command + "\n")
		default:
			b.WriteString(m.Pending.Command + "\n")
		}
		if m.Pending.Reason != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason) + "\n")
		}
		return b.String()
	}
	if m.PendingSuggested != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySuggestedCardTitle)) + "\n")
		b.WriteString(execStyle.Render(*m.PendingSuggested))
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

	inChoice := m.Pending != nil || m.PendingSensitive != nil || m.PendingSuggested != nil
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
			for i := start; i < end; i++ {
				vi := vis[i]
				opt := opts[vi]
				line := opt.Cmd
				if opt.Path != "" {
					line = "/sessions " + line
				} else {
					line = fmt.Sprintf("%-14s  %s", opt.Cmd, opt.Desc)
				}
				if i == hiIdx {
					out += suggestHi.Render("   "+line) + "\n"
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
	return out
}

// appendSuggestedLine appends the run line and copy hint for a suggested command (when dismissing the card).
func (m *Model) appendSuggestedLine(command, lang string) {
	tag := i18n.T(lang, i18n.KeyRunTagSuggested)
	m.Messages = append(m.Messages, execStyle.Render(i18n.T(lang, i18n.KeyRunLabel)+command+" ("+tag+")"))
	m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopyHint)))
}
