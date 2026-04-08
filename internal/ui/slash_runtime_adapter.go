package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/slash/dispatch"
	"delve-shell/internal/slash/view"
)

func (m *Model) slashRuntimeDeps() slashdispatch.ExecDeps[*Model, tea.Cmd] {
	return slashdispatch.ExecDeps[*Model, tea.Cmd]{
		Hooks: slashdispatch.Hooks[*Model, tea.Cmd]{
			ClearInput: func(mm *Model) *Model {
				mm.clearSlashInput()
				return mm
			},
		},
		SuggestionContext: func(input string) ([]int, []slashview.Option) {
			_, vis, viewOpts := m.slashSuggestionContext(input)
			return vis, viewOpts
		},
		SlashSuggestIndex: func(mm *Model) int { return mm.Interaction.slashSuggestIndex },
		FillInput: func(mm *Model, fill string) *Model {
			mm.Input.SetValue(fill)
			mm.Input.CursorEnd()
			mm.Interaction.slashSuggestIndex = 0
			return mm
		},
		AppendSessionNone: func(mm *Model) *Model {
			mm.AppendTranscriptLines(infoStyle.Render(mm.infoMsg(i18n.T(i18n.KeySessionNone))))
			mm.clearSlashInput()
			return mm
		},
		AppendDelRemoteNone: func(mm *Model) *Model {
			mm.AppendTranscriptLines(infoStyle.Render(mm.infoMsg(i18n.T(i18n.KeyDelRemoteNoHosts))))
			mm.clearSlashInput()
			return mm
		},
		AppendUnknownSlash: func(mm *Model) *Model {
			mm.AppendTranscriptLines(errStyle.Render(i18n.T(i18n.KeyUnknownCmd)))
			return mm
		},
		EchoSubmitted: func(mm *Model, text string) *Model {
			mm.appendUserSubmittedEcho(text)
			return mm
		},
		EmitChat: func(mm *Model, text string) *Model {
			if mm.CommandSender != nil && mm.CommandSender.Send(hostcmd.Submission{
				Submission: inputlifecycletype.InputSubmission{
					Kind:    inputlifecycletype.SubmissionChat,
					Source:  inputlifecycletype.SourceMainEnter,
					RawText: text,
				},
			}) {
				mm.Interaction.WaitingForAI = true
			}
			return mm
		},
		SessionNoneMsg:   i18n.T(i18n.KeySessionNone),
		DelRemoteNoneMsg: i18n.T(i18n.KeyDelRemoteNoHosts),
	}
}

// executeSlashSubmission runs one normalized slash submission against the shared slash runtime.
func (m *Model) executeSlashSubmission(rawText string, selectedIndex int) (*Model, tea.Cmd) {
	text := strings.TrimSpace(rawText)
	if text == "" {
		return m, nil
	}
	if m2, cmd, ok := trySlashBashQuit(m, text); ok {
		return m2, cmd
	}
	m2, cmd := slashRuntime.ExecuteSubmission(m, text, selectedIndex, m.slashRuntimeDeps())
	printCmd := m2.printTranscriptCmd(false)
	if printCmd != nil {
		if cmd != nil {
			return m2, tea.Sequence(cmd, printCmd)
		}
		return m2, printCmd
	}
	return m2, cmd
}

// executeSlashEarlySubmission runs slash-mode Enter after lifecycle submission routing.
func (m *Model) executeSlashEarlySubmission(inputLine string) (*Model, tea.Cmd, bool) {
	return slashRuntime.ExecuteEarlySubmission(m, inputLine, m.slashRuntimeDeps())
}
