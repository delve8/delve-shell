package ui

import (
	"strings"

	"delve-shell/internal/version"
)

func uiVersionText() string {
	v := strings.TrimSpace(version.Version)
	if v == "" {
		return "unknown"
	}
	return v
}
