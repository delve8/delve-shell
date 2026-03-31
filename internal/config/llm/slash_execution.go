package configllm

import (
	"strings"

	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		if text == "/config model" {
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputOverlayOpen,
				Overlay: &inputlifecycletype.OverlayPayload{
					Key: OverlayFeatureKey,
				},
			}), true, nil
		}
		return inputlifecycletype.ProcessResult{}, false, nil
	})
}
