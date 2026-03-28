//go:build windows

package run

import "delve-shell/internal/ui"

func bashRootSlashOptions(_ string) []ui.SlashOption {
	return nil
}
