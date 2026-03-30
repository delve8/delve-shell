package main

import (
	"delve-shell/internal/cli/interactive"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "delve-shell",
		Short: "AI-assisted shell: run commands after your approval",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd
			_ = args
			return interactive.Run()
		},
		SilenceUsage: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(nil)
	if err := root.Execute(); err != nil {
		slog.Error("Failed to execute root command", "error", err)
		os.Exit(1)
	}
}
