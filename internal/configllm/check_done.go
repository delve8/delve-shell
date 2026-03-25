package configllm

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func handleConfigLLMCheckDoneMessage(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	done, ok := msg.(ui.ConfigLLMCheckDoneMsg)
	if !ok {
		return m, nil, false
	}
	lang := "en"
	st := getOverlayState()
	st.Checking = false
	if done.Err != nil {
		st.Error = i18n.Tf(lang, i18n.KeyConfigLLMCheckFailed, done.Err)
		setOverlayState(st)
		return m.SetMainViewportContent(), nil, true
	}
	st.Error = ""
	setOverlayState(st)
	m.Messages = append(m.Messages, ui.SuggestStyleRender(delveLine(lang, i18n.T(lang, i18n.KeyConfigSavedLLM))))
	if done.CorrectedBaseURL != "" {
		m.Messages = append(m.Messages, ui.SuggestStyleRender(delveLine(lang, i18n.Tf(lang, i18n.KeyConfigLLMBaseURLAutoCorrected, done.CorrectedBaseURL))))
	}
	m.Messages = append(m.Messages, ui.SuggestStyleRender(delveLine(lang, i18n.T(lang, i18n.KeyConfigLLMCheckOK))))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	m = m.CloseOverlayVisual()
	st = getOverlayState()
	st.Active = false
	setOverlayState(st)
	m.Host.NotifyConfigUpdated()
	return m, nil, true
}

func delveLine(lang, msg string) string {
	return i18n.T(lang, i18n.KeyDelveLabel) + " " + msg
}
