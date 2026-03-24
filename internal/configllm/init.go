package configllm

import "delve-shell/internal/ui"

func init() {
	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildConfigLLMOverlayContent(m)
	})
}
