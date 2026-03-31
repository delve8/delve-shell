//go:build !windows

package run

import (
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func bashRootSlashOptions(lang string) []ui.SlashOption {
	return []ui.SlashOption{{Cmd: "/bash", Desc: i18n.T(i18n.KeyDescSh)}}
}
