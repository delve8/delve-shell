package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/config"
)

// ReadRecent 从会话文件中读取最近若干条事件，用于「查看上下文」等只读场景；maxLines<=0 表示不限制。
// 若文件不存在（会话尚未写入过）返回 nil, nil。
func ReadRecent(sessionPath string, maxLines int) ([]Event, error) {
	f, err := os.Open(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var events []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		events = append(events, ev)
		if maxLines > 0 && len(events) >= maxLines {
			// 保留最后 maxLines 条
			events = events[len(events)-maxLines:]
		}
	}
	if maxLines > 0 && len(events) > maxLines {
		events = events[len(events)-maxLines:]
	}
	return events, sc.Err()
}

// ListSessions 列出 HistoryDir 下所有会话文件（返回绝对路径）
func ListSessions() ([]string, error) {
	dir := config.HistoryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	return paths, nil
}
