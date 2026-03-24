package configllm

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/service/configsvc"
	"delve-shell/internal/ui"
)

func runConfigLLMCheckCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		corrected, err := configsvc.CheckLLMAndMaybeAutoCorrect(ctx, nil)
		if err != nil {
			return ui.ConfigLLMCheckDoneMsg{Err: err}
		}
		if corrected != "" {
			return ui.ConfigLLMCheckDoneMsg{CorrectedBaseURL: corrected}
		}
		return ui.ConfigLLMCheckDoneMsg{Err: nil}
	}
}
