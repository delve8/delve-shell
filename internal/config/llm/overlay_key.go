package configllm

import (
	"context"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui"
)

const configModelFieldCount = 5

func applyOverlayFieldFocus(st *overlayState) {
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
}

func firstIncompleteOverlayField(st overlayState) (idx int, missing bool) {
	if strings.TrimSpace(st.ModelInput.Value()) == "" {
		return 2, true
	}
	return 0, false
}

func handleOverlayKey(m *ui.Model, key string, msg tea.KeyMsg) (*ui.Model, tea.Cmd, bool) {
	st := getOverlayState()
	if !st.Active {
		return m, nil, false
	}

	switch key {
	case teakey.Up, teakey.Down, teakey.Tab:
		dir := 1
		if key == teakey.Up {
			dir = -1
		}
		st.FieldIndex = (st.FieldIndex + dir + configModelFieldCount) % configModelFieldCount
		applyOverlayFieldFocus(&st)
		setOverlayState(st)
		return m, nil, true
	case teakey.Enter:
		if st.Checking {
			return m, nil, true
		}
		if st.FieldIndex != configModelFieldCount-1 {
			st.FieldIndex = (st.FieldIndex + 1 + configModelFieldCount) % configModelFieldCount
			applyOverlayFieldFocus(&st)
			setOverlayState(st)
			return m, nil, true
		}
		baseURL := strings.TrimSpace(st.BaseURLInput.Value())
		apiKey := strings.TrimSpace(st.ApiKeyInput.Value())
		model := strings.TrimSpace(st.ModelInput.Value())
		maxMessagesStr := strings.TrimSpace(st.MaxMessagesInput.Value())
		maxCharsStr := strings.TrimSpace(st.MaxCharsInput.Value())
		if missingIdx, missing := firstIncompleteOverlayField(st); missing {
			st.FieldIndex = missingIdx
			applyOverlayFieldFocus(&st)
			st.Error = i18n.T(i18n.KeyConfigModelModelRequired)
			setOverlayState(st)
			return m, nil, true
		}
		cfg, err := config.Load()
		if err != nil || cfg == nil {
			cfg = config.Default()
			if err := config.EnsureRootDir(); err != nil {
				m.AppendTranscriptLines(ui.ErrStyleRender(i18n.T(i18n.KeyConfigPrefix) + err.Error()))
				return m, nil, true
			}
		}

		cfg.LLM.BaseURL = baseURL
		cfg.LLM.APIKey = apiKey
		cfg.LLM.Model = model

		if s := strings.TrimSpace(maxMessagesStr); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n >= 0 {
				cfg.LLM.MaxContextMessages = n
			}
		} else {
			cfg.LLM.MaxContextMessages = 0
		}
		if s := strings.TrimSpace(maxCharsStr); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n >= 0 {
				cfg.LLM.MaxContextChars = n
			}
		} else {
			cfg.LLM.MaxContextChars = 0
		}
		if err := config.Write(cfg); err != nil {
			m.AppendTranscriptLines(ui.ErrStyleRender(i18n.T(i18n.KeyConfigPrefix) + err.Error()))
			return m, nil, true
		}
		st.Checking = true
		setOverlayState(st)
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			corrected, err := CheckLLMAndMaybeAutoCorrect(ctx)
			if err != nil {
				return CheckDoneMsg{ErrText: err.Error()}
			}
			if corrected != "" {
				return CheckDoneMsg{CorrectedBaseURL: corrected}
			}
			return CheckDoneMsg{}
		}, true
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
	st.Error = ""
	setOverlayState(st)
	return m, cmd, true
}
