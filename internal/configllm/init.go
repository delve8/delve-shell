package configllm

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

// Register wires config-LLM slash routes and overlay providers into the UI. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	openOverlay := func(m ui.Model) ui.Model {
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
		st.BaseURLInput.Placeholder = "https://api.openai.com/v1 (optional)"
		st.BaseURLInput.SetValue(cfg.LLM.BaseURL)
		st.BaseURLInput.Focus()
		st.ApiKeyInput = textinput.New()
		st.ApiKeyInput.Placeholder = "sk-... or $API_KEY"
		st.ApiKeyInput.EchoMode = textinput.EchoPassword
		st.ApiKeyInput.SetValue(cfg.LLM.APIKey)
		st.ApiKeyInput.Blur()
		st.ModelInput = textinput.New()
		st.ModelInput.Placeholder = "gpt-4o-mini (optional)"
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
		return m.OpenOverlayFeature("config_llm", i18n.T("en", i18n.KeyConfigLLMTitle), "")
	}
	ui.RegisterOverlayFeature(ui.OverlayFeature{
		KeyID: "config_llm",
		Open: func(m ui.Model, req ui.OverlayOpenRequest) (ui.Model, tea.Cmd, bool) {
			if req.Key != "config_llm" {
				return m, nil, false
			}
			return openOverlay(m), nil, true
		},
		Event: func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
			if m.Overlay.Key != "config_llm" {
				return m, nil, false
			}
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
			m.EmitConfigUpdatedIntent()
			return m, nil, true
		},
		Content: func(m ui.Model) (string, bool) {
			return buildConfigLLMOverlayContent()
		},
		Key: func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
			return handleOverlayKey(m, key, msg)
		},
		Startup: func(m ui.Model) (ui.Model, tea.Cmd, bool) {
			return openOverlay(m), nil, true
		},
		Close: func(m ui.Model, activeKey string) ui.Model {
			if activeKey != "config_llm" {
				return m
			}
			ResetOnOverlayClose()
			return m
		},
	})
}
