package configllm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const configLLMFieldCount = 5

func handleOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	st := getOverlayState()
	if !st.Active {
		return m, nil, false
	}

	switch key {
	case "up", "down":
		dir := 1
		if key == "up" {
			dir = -1
		}
		st.FieldIndex = (st.FieldIndex + dir + configLLMFieldCount) % configLLMFieldCount
		st.BaseURLInput.Blur()
		st.ApiKeyInput.Blur()
		st.ModelInput.Blur()
		st.MaxMessagesInput.Blur()
		st.MaxCharsInput.Blur()
		switch st.FieldIndex {
		case 0:
			st.BaseURLInput.Focus()
		case 1:
			st.ApiKeyInput.Focus()
		case 2:
			st.ModelInput.Focus()
		case 3:
			st.MaxMessagesInput.Focus()
		case 4:
			st.MaxCharsInput.Focus()
		}
		setOverlayState(st)
		return m, nil, true
	case "enter":
		if st.Checking {
			return m, nil, true
		}
		baseURL := strings.TrimSpace(st.BaseURLInput.Value())
		apiKey := strings.TrimSpace(st.ApiKeyInput.Value())
		model := strings.TrimSpace(st.ModelInput.Value())
		maxMessagesStr := strings.TrimSpace(st.MaxMessagesInput.Value())
		maxCharsStr := strings.TrimSpace(st.MaxCharsInput.Value())
		if model == "" {
			st.Error = i18n.T("en", i18n.KeyConfigLLMModelRequired)
			setOverlayState(st)
			return m, nil, true
		}
		m = applyConfigLLMFromOverlayStart(m, baseURL, apiKey, model, maxMessagesStr, maxCharsStr)
		st = getOverlayState()
		if !st.Checking {
			return m, nil, true
		}
		return m, runConfigLLMCheckCmd(), true
	}

	var cmd tea.Cmd
	switch st.FieldIndex {
	case 0:
		st.BaseURLInput, cmd = st.BaseURLInput.Update(msg)
	case 1:
		st.ApiKeyInput, cmd = st.ApiKeyInput.Update(msg)
	case 2:
		st.ModelInput, cmd = st.ModelInput.Update(msg)
	case 3:
		st.MaxMessagesInput, cmd = st.MaxMessagesInput.Update(msg)
	case 4:
		st.MaxCharsInput, cmd = st.MaxCharsInput.Update(msg)
	}
	setOverlayState(st)
	return m, cmd, true
}
