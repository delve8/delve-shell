package main

import (
	"delve-shell/internal/cli/interactive"
	"delve-shell/internal/version"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:     "delve-shell",
		Short:   "AI-assisted shell: run commands after your approval",
		Long:    "AI-assisted shell with human-in-the-loop command execution.\n\nStart the TUI with `delve-shell`, then use `/help` inside the app for the full interactive help and slash-command reference.",
		Version: version.String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd
			_ = args
			return interactive.Run()
		},
		SilenceUsage: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(nil)
	root.SetHelpTemplate(`{{with (or .Long .Short)}}{{.}}{{end}}

Usage:
  {{.UseLine}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
`)
	if err := root.Execute(); err != nil {
		slog.Error("Failed to execute root command", "error", err)
		os.Exit(1)
	}
}
