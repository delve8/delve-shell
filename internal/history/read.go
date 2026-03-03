package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/config"
)

// ReadRecent reads the most recent events from the session file for read-only use (e.g. view_context). maxLines<=0 means no limit.
// If the file does not exist (session never written), returns nil, nil.
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
			// keep last maxLines
			events = events[len(events)-maxLines:]
		}
	}
	if maxLines > 0 && len(events) > maxLines {
		events = events[len(events)-maxLines:]
	}
	return events, sc.Err()
}

// ListSessions lists all session files under HistoryDir (returns absolute paths).
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
