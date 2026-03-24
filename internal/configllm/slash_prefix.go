package configllm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func registerSlashPrefix() {
	ui.RegisterSlashPrefix("/config llm base_url ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config llm base_url ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			m = applyConfigLLMField(m, "base_url", strings.TrimSpace(rest))
			return m, nil, true
		},
	})
	ui.RegisterSlashPrefix("/config llm api_key ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config llm api_key ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			m = applyConfigLLMField(m, "api_key", strings.TrimSpace(rest))
			return m, nil, true
		},
	})
	ui.RegisterSlashPrefix("/config llm model ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config llm model ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			m = applyConfigLLMField(m, "model", strings.TrimSpace(rest))
			return m, nil, true
		},
	})
}
