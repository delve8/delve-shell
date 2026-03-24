package ui

import (
	"fmt"
	"strings"

	"delve-shell/internal/i18n"
)

// slashDropdownBelowInput returns extra lines to show under the input when in slash mode (not in approval/sensitive choice).
func (m Model) slashDropdownBelowInput(lang string) string {
	inputVal := m.Input.Value()
	if !strings.HasPrefix(inputVal, "/") {
		return ""
	}
	opts := getSlashOptionsForInput(inputVal, lang, m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	if len(vis) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	hiIdx := m.Interaction.SlashSuggestIndex
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
	if m.Layout.Width > 4 {
		maxLineLen = m.Layout.Width - 4 // leave margin for prefix and avoid wrap
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
					out.WriteString(suggestHi.Render("   "+line) + "\n")
				} else {
					out.WriteString(suggestStyle.Render("   "+line) + "\n")
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
					out.WriteString(suggestStyle.Render(indent+string(descRunes[j:endJ])) + "\n")
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
				out.WriteString(suggestHi.Render("   "+line) + "\n")
			} else {
				out.WriteString(suggestStyle.Render("   "+line) + "\n")
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
					out.WriteString(suggestStyle.Render(indent+string(rest[j:endJ])) + "\n")
					remainingLines--
				}
			}
		} else {
			// No description or no width budget: show command only.
			line := fmt.Sprintf("%-*s", cmdWidth, cmdText)
			if i == hiIdx {
				out.WriteString(suggestHi.Render("   "+line) + "\n")
			} else {
				out.WriteString(suggestStyle.Render("   "+line) + "\n")
			}
			remainingLines--
		}
	}
	return out.String()
}

// choiceLinesBelowInput returns extra lines for numeric choice menu under the input.
func (m Model) choiceLinesBelowInput(lang string) string {
	opts := getChoiceOptions(m, lang)
	if len(opts) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	for i, o := range opts {
		line := fmt.Sprintf("%d  %s", o.Num, o.Label)
		if i == m.Interaction.ChoiceIndex {
			out.WriteString(suggestHi.Render(" "+line) + "\n")
		} else {
			out.WriteString(suggestStyle.Render(" "+line) + "\n")
		}
	}
	return out.String()
}

// waitingLineBelowInput returns the "wait or /cancel" hint when AI is running (empty if not applicable).
func (m Model) waitingLineBelowInput(lang string) string {
	inChoice := m.Pending != nil || m.PendingSensitive != nil
	if m.Interaction.WaitingForAI && !inChoice {
		return "\n" + suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel))
	}
	return ""
}
