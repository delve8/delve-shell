package config

import (
	"os"
	"path/filepath"
)

// DefaultRootDir is the default config and data root directory name.
const DefaultRootDir = ".delve-shell"

// RootDir returns the delve-shell root directory (e.g. ~/.delve-shell), or default if not set.
func RootDir() string {
	if p := os.Getenv("DELVE_SHELL_ROOT"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, DefaultRootDir)
}

// ConfigPath returns the config file path.
func ConfigPath() string {
	return filepath.Join(RootDir(), "config.yaml")
}

// AllowlistPath returns the allowlist config path (separate from config.yaml).
func AllowlistPath() string {
	return filepath.Join(RootDir(), "allowlist.yaml")
}

// SensitivePatternsPath returns the sensitive-output patterns file path (regex rules; optional override).
func SensitivePatternsPath() string {
	return filepath.Join(RootDir(), "sensitive_patterns.yaml")
}

// RemotesPath returns the remotes config path (separate from config.yaml).
func RemotesPath() string {
	return filepath.Join(RootDir(), "remotes.yaml")
}

// RulesDir returns the rules directory path.
func RulesDir() string {
	return filepath.Join(RootDir(), "rules")
}

// HistoryDir returns the session files directory path (~/.delve-shell/sessions).
func HistoryDir() string {
	return filepath.Join(RootDir(), "sessions")
}
