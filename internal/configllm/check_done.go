package configllm

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func handleConfigLLMCheckDoneMessage(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	done, ok := msg.(CheckDoneMsg)
	if !ok {
		return m, nil, false
	}
	lang := "en"
	st := getOverlayState()
	st.Checking = false
	if done.ErrText != "" {
		st.Error = i18n.Tf(lang, i18n.KeyConfigLLMCheckFailed, done.ErrText)
		setOverlayState(st)
		return m.SetMainViewportContent(), nil, true
	}
	st.Error = ""
	setOverlayState(st)
	mm := ui.TranscriptAppendMsg{Lines: []uivm.Line{
		{Kind: uivm.LineSystemSuggest, Text: i18n.T(lang, i18n.KeyConfigSavedLLM)},
	}}
	if done.CorrectedBaseURL != "" {
		mm.Lines = append(mm.Lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.Tf(lang, i18n.KeyConfigLLMBaseURLAutoCorrected, done.CorrectedBaseURL)})
	}
	mm.Lines = append(mm.Lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.T(lang, i18n.KeyConfigLLMCheckOK)})
	mm.Lines = append(mm.Lines, uivm.Line{Kind: uivm.LineBlank})
	next, _ := m.Update(mm)
	m = next.(ui.Model)
	m = m.CloseOverlayVisual()
	st = getOverlayState()
	st.Active = false
	setOverlayState(st)
	m.Host.NotifyConfigUpdated()
	return m, nil, true
}
