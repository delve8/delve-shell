package cli

import (
	"github.com/spf13/cobra"

	"delve-shell/internal/cli/interactive"
)

func Run(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	return interactive.Run()
}
