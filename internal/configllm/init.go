package configllm

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func init() {
	registerSlashExact()
	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildConfigLLMOverlayContent(m)
	})
	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		return handleOverlayKey(m, key, msg)
	})
}
