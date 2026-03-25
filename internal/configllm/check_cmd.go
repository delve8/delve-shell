package configllm

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/service/configsvc"
)

func runConfigLLMCheckCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		corrected, err := configsvc.CheckLLMAndMaybeAutoCorrect(ctx, nil)
		if err != nil {
			return CheckDoneMsg{ErrText: err.Error()}
		}
		if corrected != "" {
			return CheckDoneMsg{CorrectedBaseURL: corrected}
		}
		return CheckDoneMsg{}
	}
}
