package configllm

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

// Register wires config-LLM slash routes and overlay providers into the UI. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	ui.RegisterMessageProvider(handleConfigLLMCheckDoneMessage)
	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildConfigLLMOverlayContent()
	})
	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		return handleOverlayKey(m, key, msg)
	})
	ui.RegisterStartupOverlayProvider(func(m ui.Model) (ui.Model, tea.Cmd, bool) {
		return openOverlay(m), nil, true
	})
}
