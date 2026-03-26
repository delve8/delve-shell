package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/slashdispatch"
	"delve-shell/internal/slashview"
)

func (m Model) slashRuntimeDeps() slashdispatch.ExecDeps[Model, tea.Cmd] {
	return slashdispatch.ExecDeps[Model, tea.Cmd]{
		Hooks: slashdispatch.Hooks[Model, tea.Cmd]{
			BeforeDispatch: func(line string) { m.requestSlashDispatchAction(line) },
			AfterDispatch:  func(line string) { m.traceSlashEnteredAction(line) },
			ClearInput:     func(mm Model) Model { return mm.clearSlashInput() },
		},
		SuggestionContext: func(input string) ([]int, []slashview.Option) {
			_, vis, viewOpts := m.slashSuggestionContext(input)
			return vis, viewOpts
		},
		SlashSuggestIndex: func(mm Model) int { return mm.Interaction.slashSuggestIndex },
		FillInput: func(mm Model, fill string) Model {
			mm.Input.SetValue(fill)
			mm.Input.CursorEnd()
			mm.Interaction.slashSuggestIndex = 0
			return mm
		},
		AppendSessionNone: func(mm Model) Model {
			mm = mm.AppendTranscriptLines(suggestStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeySessionNone))))
			mm = mm.RefreshViewport()
			return mm.clearSlashInput()
		},
		AppendDelRemoteNone: func(mm Model) Model {
			mm = mm.AppendTranscriptLines(suggestStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyDelRemoteNoHosts))))
			mm = mm.RefreshViewport()
			return mm.clearSlashInput()
		},
		AppendUnknownSlash: func(mm Model) Model {
			mm = mm.AppendTranscriptLines(errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUnknownCmd))))
			return mm.RefreshViewport()
		},
		EchoSubmitted: func(mm Model, text string) Model {
			return mm.appendUserSubmittedEcho(text)
		},
		EmitChat: func(mm Model, text string) Model {
			if mm.EmitChatSubmitIntent(text, inputlifecycletype.SourceMainEnter) {
				mm.Interaction.WaitingForAI = true
			}
			return mm
		},
		SessionNoneMsg:   i18n.T(m.getLang(), i18n.KeySessionNone),
		DelRemoteNoneMsg: i18n.T(m.getLang(), i18n.KeyDelRemoteNoHosts),
	}
}

// executeSlashSubmission runs one normalized slash submission against the shared slash runtime.
func (m Model) executeSlashSubmission(rawText string, selectedIndex int) (Model, tea.Cmd) {
	text := strings.TrimSpace(rawText)
	if text == "" {
		return m, nil
	}
	return slashRuntime.ExecuteSubmission(m, text, selectedIndex, m.slashRuntimeDeps())
}

// executeSlashEarlySubmission runs slash-mode Enter after lifecycle submission routing.
func (m Model) executeSlashEarlySubmission(inputLine string) (Model, tea.Cmd, bool) {
	return slashRuntime.ExecuteEarlySubmission(m, inputLine, m.slashRuntimeDeps())
}
