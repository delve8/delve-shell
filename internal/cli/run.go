package cli

import (
	"github.com/spf13/cobra"

	"delve-shell/internal/cli/interactive"
	_ "delve-shell/internal/configllm"
	_ "delve-shell/internal/skill"
)

func Run(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	return interactive.Run()
}
