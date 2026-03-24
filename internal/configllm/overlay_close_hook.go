package configllm

import "delve-shell/internal/ui"

func registerOverlayCloseHook() {
	ui.RegisterOverlayCloseHook(func(m ui.Model) ui.Model {
		m.ConfigLLMActive = false
		m.ConfigLLMChecking = false
		m.ConfigLLMError = ""
		return m
	})
}
