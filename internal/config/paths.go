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

// CustomAllowlistPath returns the user-editable allowlist overlay path.
func CustomAllowlistPath() string {
	return filepath.Join(RootDir(), "allowlist_custom.yaml")
}

// SensitivePatternsPath returns the sensitive-output patterns file path (regex rules; optional override).
func SensitivePatternsPath() string {
	return filepath.Join(RootDir(), "sensitive_patterns.yaml")
}

// RemotesPath returns the remotes config path (separate from config.yaml).
func RemotesPath() string {
	return filepath.Join(RootDir(), "remotes.yaml")
}

// InputHistoryPath returns the persisted local input-history path.
func InputHistoryPath() string {
	return filepath.Join(RootDir(), "input_history.json")
}

// RulesDir returns the rules directory path.
func RulesDir() string {
	return filepath.Join(RootDir(), "rules")
}

// HistoryDir returns the session files directory path (~/.delve-shell/sessions).
func HistoryDir() string {
	return filepath.Join(RootDir(), "sessions")
}

// HostsDir returns the host memory directory path (~/.delve-shell/hosts).
func HostsDir() string {
	return filepath.Join(RootDir(), "hosts")
}

// SkillsDir returns the skills directory path (~/.delve-shell/skills). Each subdir is one skill (SKILL.md + optional scripts/).
func SkillsDir() string {
	return filepath.Join(RootDir(), "skills")
}

// SkillAuditPath returns the skill install/remove audit log path (~/.delve-shell/skill_audit.jsonl).
func SkillAuditPath() string {
	return filepath.Join(RootDir(), "skill_audit.jsonl")
}

// SkillsManifestPath returns the skills manifest path (~/.delve-shell/skills/manifest.json).
// Tracks which skill dir was installed from which git URL/ref for upgrade and display.
func SkillsManifestPath() string {
	return filepath.Join(RootDir(), "skills", "manifest.json")
}
