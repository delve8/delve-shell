package configllm

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

// Register wires config-LLM slash routes and overlay providers into the UI. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	ui.RegisterOverlayFeature(ui.OverlayFeature{
		KeyID: "config_llm",
		Open: func(m ui.Model, req ui.OverlayOpenRequest) (ui.Model, tea.Cmd, bool) {
			if req.Key != "config_llm" {
				return m, nil, false
			}
			return openOverlay(m), nil, true
		},
		Event: handleConfigLLMCheckDoneMessage,
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
