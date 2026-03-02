package history

import (
	"os"
	"path/filepath"
	"time"

	"delve-shell/internal/config"
)

// Prune 根据配置清理过期历史；MaxDays>0 时删除超过该天数的会话文件
func Prune(cfg *config.Config) error {
	if cfg.History.MaxDays <= 0 {
		return nil
	}
	dir := config.HistoryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-time.Duration(cfg.History.MaxDays) * 24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		}
	}
	return nil
}
