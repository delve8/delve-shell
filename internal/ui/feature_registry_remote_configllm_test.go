package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func registerTestTitleBarMirror() {
	RegisterTitleBarFragmentProvider(func(m Model) (string, bool) {
		if !m.Context.RemoteActive {
			return "", false
		}
		if m.Context.RemoteLabel != "" {
			return "Remote " + m.Context.RemoteLabel, true
		}
		return "Remote", true
	})
}

func registerTestExactOverlayMirrors() {
	RegisterSlashExact("/config add-remote", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return openTestAddRemoteOverlay(m, true, false), nil
		},
		ClearInput: true,
	})
	RegisterSlashExact("/remote on", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return openTestAddRemoteOverlay(m, false, true), nil
		},
		ClearInput: true,
	})
	RegisterSlashExact("/config llm", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return openTestConfigLLMOverlay(m), nil
		},
		ClearInput: true,
	})
	RegisterSlashExact("/config del-remote", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Input.SetValue("/config del-remote ")
			m.Input.CursorEnd()
			m.Interaction.SlashSuggestIndex = 0
			return m, nil
		},
		ClearInput: false,
	})
}

func registerTestStartupOverlayMirror() {
	RegisterStartupOverlayProvider(func(m Model) (Model, tea.Cmd, bool) {
		return openTestConfigLLMOverlay(m), nil, true
	})
}

func registerTestOverlayCloseResetMirror() {
	RegisterOverlayCloseHook(func(m Model) Model {
		return applyTestOverlayCloseFeatureResets(m)
	})
}

func openTestAddRemoteOverlay(m Model, save bool, connect bool) Model {
	m.Overlay.Active = true
	m.Overlay.Title = i18n.T(m.getLang(), i18n.KeyAddRemoteTitle)
	m.AddRemote.Active = true
	m.AddRemote.Error = ""
	m.AddRemote.OfferOverwrite = false
	m.AddRemote.Save = save
	m.AddRemote.Connect = connect
	m.PathCompletion.Candidates = nil
	m.PathCompletion.Index = -1
	m.AddRemote.FieldIndex = 0
	m.AddRemote.HostInput = textinput.New()
	m.AddRemote.HostInput.Placeholder = "host or host:22"
	m.AddRemote.HostInput.Focus()
	m.AddRemote.UserInput = textinput.New()
	m.AddRemote.UserInput.Placeholder = "e.g. root"
	m.AddRemote.UserInput.SetValue("root")
	m.AddRemote.NameInput = textinput.New()
	m.AddRemote.NameInput.Placeholder = "name (optional)"
	m.AddRemote.KeyInput = textinput.New()
	m.AddRemote.KeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
	return m
}

func openTestConfigLLMOverlay(m Model) Model {
	m.Overlay.Active = true
	m.Overlay.Title = i18n.T(m.getLang(), i18n.KeyConfigLLMTitle)
	m.ConfigLLM.Active = true
	m.ConfigLLM.Checking = false
	m.ConfigLLM.Error = ""
	m.ConfigLLM.FieldIndex = 0
	m.ConfigLLM.BaseURLInput = textinput.New()
	m.ConfigLLM.BaseURLInput.Focus()
	m.ConfigLLM.ApiKeyInput = textinput.New()
	m.ConfigLLM.ApiKeyInput.Blur()
	m.ConfigLLM.ModelInput = textinput.New()
	m.ConfigLLM.ModelInput.Blur()
	m.ConfigLLM.MaxMessagesInput = textinput.New()
	m.ConfigLLM.MaxMessagesInput.Blur()
	m.ConfigLLM.MaxCharsInput = textinput.New()
	m.ConfigLLM.MaxCharsInput.Blur()
	return m
}
