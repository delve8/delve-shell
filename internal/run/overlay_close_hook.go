package run

import (
	"delve-shell/internal/configllm"
	"delve-shell/internal/ui"
)

func applyOverlayCloseFeatureResets(m ui.Model) ui.Model {
	configllm.ResetOnOverlayClose()
	return m
}

func registerOverlayCloseHookRun() {
	ui.RegisterOverlayCloseHook(applyOverlayCloseFeatureResets)
}
