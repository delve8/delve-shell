package configllm

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
)

// Register wires config-LLM slash routes and overlay providers into the UI. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	openOverlay := func(m *ui.Model) {
		cfg, err := config.Load()
		if err != nil || cfg == nil {
			cfg = config.Default()
		}
		var st overlayState
		st.Active = true
		st.Checking = false
		st.Error = ""
		st.FieldIndex = 0
		st.BaseURLInput = textinput.New()
		st.BaseURLInput.Placeholder = i18n.T(i18n.KeyConfigLLMBaseURLPlaceholder)
		st.BaseURLInput.SetValue(cfg.LLM.BaseURL)
		st.BaseURLInput.Focus()
		st.ApiKeyInput = textinput.New()
		st.ApiKeyInput.Placeholder = i18n.T(i18n.KeyConfigLLMApiKeyPlaceholder)
		st.ApiKeyInput.EchoMode = textinput.EchoPassword
		st.ApiKeyInput.SetValue(cfg.LLM.APIKey)
		st.ApiKeyInput.Blur()
		st.ModelInput = textinput.New()
		st.ModelInput.SetValue(cfg.LLM.Model)
		st.ModelInput.Blur()
		st.MaxMessagesInput = textinput.New()
		st.MaxMessagesInput.Placeholder = ""
		if cfg.LLM.MaxContextMessages > 0 {
			st.MaxMessagesInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextMessages))
		}
		st.MaxMessagesInput.Blur()
		st.MaxCharsInput = textinput.New()
		st.MaxCharsInput.Placeholder = ""
		if cfg.LLM.MaxContextChars > 0 {
			st.MaxCharsInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextChars))
		}
		st.MaxCharsInput.Blur()
		setOverlayState(st)
		m.OpenOverlayFeature(OverlayFeatureKey, i18n.T(i18n.KeyConfigLLMTitle), "")
	}
	ui.RegisterOverlayFeature(ui.OverlayFeature{
		KeyID: OverlayFeatureKey,
		Open: func(m *ui.Model, req ui.OverlayOpenRequest) (*ui.Model, tea.Cmd, bool) {
			if req.Key != OverlayFeatureKey {
				return m, nil, false
			}
			openOverlay(m)
			return m, nil, true
		},
		Event: func(m *ui.Model, msg tea.Msg) (*ui.Model, tea.Cmd, bool) {
			if m.Overlay.Key != OverlayFeatureKey {
				return m, nil, false
			}
			done, ok := msg.(CheckDoneMsg)
			if !ok {
				return m, nil, false
			}
			st := getOverlayState()
			st.Checking = false
			if done.ErrText != "" {
				st.Error = i18n.Tf(i18n.KeyConfigLLMCheckFailed, done.ErrText)
				setOverlayState(st)
				return m, nil, true
			}
			st.Error = ""
			setOverlayState(st)
			mm := ui.TranscriptAppendMsg{Lines: []uivm.Line{
				{Kind: uivm.LineSystemSuggest, Text: i18n.T(i18n.KeyConfigSavedLLM)},
			}}
			if done.CorrectedBaseURL != "" {
				mm.Lines = append(mm.Lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.Tf(i18n.KeyConfigLLMBaseURLAutoCorrected, done.CorrectedBaseURL)})
			}
			mm.Lines = append(mm.Lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.T(i18n.KeyConfigLLMCheckOK)})
			mm.Lines = append(mm.Lines, uivm.Line{Kind: uivm.LineBlank})
			next, _ := m.Update(mm)
			m = next.(*ui.Model)
			m.CloseOverlayVisual()
			st = getOverlayState()
			st.Active = false
			setOverlayState(st)
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.ConfigUpdated{})
			}
			return m, nil, true
		},
		Content: func(m *ui.Model) (string, bool) {
			return buildConfigLLMOverlayContent()
		},
		Key: func(m *ui.Model, key string, msg tea.KeyMsg) (*ui.Model, tea.Cmd, bool) {
			return handleOverlayKey(m, key, msg)
		},
		Startup: func(m *ui.Model) (*ui.Model, tea.Cmd, bool) {
			openOverlay(m)
			return m, nil, true
		},
		Close: func(m *ui.Model, activeKey string) {
			if activeKey != OverlayFeatureKey {
				return
			}
			ResetOnOverlayClose()
		},
	})
}
