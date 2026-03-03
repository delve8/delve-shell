package cli

import (
	"github.com/spf13/cobra"
)

// Execute runs the root command and returns an error (non-nil means exit 1).
// Runs TUI directly; no run/help/completion subcommands; help is /help inside TUI.
func Execute() error {
	root := &cobra.Command{
		Use:          "delve-shell",
		Short:        "AI-assisted ops: run commands after your approval",
		RunE:         runRun,
		SilenceUsage: true, // on error show only error message, not Usage/Flags
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(nil)
	return root.Execute()
}
