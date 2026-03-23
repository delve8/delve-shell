package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func (m Model) handleConfigLLMCheckDoneMsg(msg ConfigLLMCheckDoneMsg) (Model, tea.Cmd) {
	m.ConfigLLMChecking = false
	lang := m.getLang()
	if msg.Err != nil {
		m.ConfigLLMError = i18n.Tf(lang, i18n.KeyConfigLLMCheckFailed, msg.Err)
		m.Viewport.SetContent(m.buildContent())
		return m, nil
	}
	m.ConfigLLMError = ""
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigSavedLLM))))
	if msg.CorrectedBaseURL != "" {
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigLLMBaseURLAutoCorrected, msg.CorrectedBaseURL))))
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigLLMCheckOK))))
	m.Messages = append(m.Messages, "")
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	m.OverlayActive = false
	m.ConfigLLMActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	if m.ConfigUpdatedChan != nil {
		select {
		case m.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m, nil
}

func (m Model) handleAddSkillRefsLoadedMsg(msg AddSkillRefsLoadedMsg) (Model, tea.Cmd) {
	if m.AddSkillActive {
		m.AddSkillRefsFullList = msg.Refs
		m.AddSkillRefCandidates = filterByPrefix(msg.Refs, m.AddSkillRefInput.Value())
		m.AddSkillRefIndex = 0
	}
	return m, nil
}

func (m Model) handleAddSkillPathsLoadedMsg(msg AddSkillPathsLoadedMsg) (Model, tea.Cmd) {
	if m.AddSkillActive {
		m.AddSkillPathsFullList = msg.Paths
		m = m.updateAddSkillPathCandidates()
	}
	return m, nil
}
