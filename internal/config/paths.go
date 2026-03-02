package config

import (
	"os"
	"path/filepath"
)

// DefaultRootDir 默认配置与数据根目录
const DefaultRootDir = ".delve-shell"

// RootDir 返回 delve-shell 根目录（如 ~/.delve-shell），若未设置则用默认
func RootDir() string {
	if p := os.Getenv("DELVE_SHELL_ROOT"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, DefaultRootDir)
}

// ConfigPath 配置文件路径
func ConfigPath() string {
	return filepath.Join(RootDir(), "config.yaml")
}

// AllowlistPath 允许列表配置文件路径（独立于 config.yaml）
func AllowlistPath() string {
	return filepath.Join(RootDir(), "allowlist.yaml")
}

// RulesDir rules 目录路径
func RulesDir() string {
	return filepath.Join(RootDir(), "rules")
}

// HistoryDir 会话历史目录路径
func HistoryDir() string {
	return filepath.Join(RootDir(), "history")
}
