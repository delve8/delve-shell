package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"delve-shell/internal/config"
)

// SessionSummary is a human-readable summary of one session for listing (time + snippet).
type SessionSummary struct {
	Path        string // absolute path to the session file
	ID          string // session id (filename without .jsonl)
	DisplayTime string // e.g. "2006-01-02 15:04"
	Snippet     string // first user_input text or first command, truncated
}

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
	r := bufio.NewReader(f)
	for {
		lineBytes, readErr := r.ReadBytes('\n')
		if len(lineBytes) > 0 {
			lineBytes = bytes.TrimRight(lineBytes, "\r\n")
			if len(lineBytes) > 0 {
				var ev Event
				if json.Unmarshal(lineBytes, &ev) == nil {
					events = append(events, ev)
					if maxLines > 0 && len(events) > maxLines {
						events = events[len(events)-maxLines:]
					}
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return events, readErr
		}
	}
	return events, nil
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

// ListSessionsWithSummary returns sessions sorted by mtime descending (newest first),
// with human-readable display time and content snippet for each.
func ListSessionsWithSummary(maxSessions int) ([]SessionSummary, error) {
	paths, err := ListSessions()
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	// Sort by ModTime desc: need file info
	type pathMtime struct {
		path  string
		mtime time.Time
	}
	var withMtime []pathMtime
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		withMtime = append(withMtime, pathMtime{p, info.ModTime()})
	}
	sort.Slice(withMtime, func(i, j int) bool { return withMtime[i].mtime.After(withMtime[j].mtime) })
	if maxSessions > 0 && len(withMtime) > maxSessions {
		withMtime = withMtime[:maxSessions]
	}
	var out []SessionSummary
	for _, pm := range withMtime {
		s := getSessionSummary(pm.path, pm.mtime)
		out = append(out, s)
	}
	return out, nil
}

const snippetMaxLen = 60

func getSessionSummary(path string, mtime time.Time) SessionSummary {
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	displayTime := mtime.Format("2006-01-02 15:04")
	snippet := ""
	events, err := ReadRecent(path, 20)
	if err != nil || len(events) == 0 {
		return SessionSummary{Path: path, ID: id, DisplayTime: displayTime, Snippet: snippet}
	}
	for _, ev := range events {
		switch ev.Type {
		case "user_input":
			var payload struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &payload) == nil && payload.Text != "" {
				snippet = payload.Text
				if len(snippet) > snippetMaxLen {
					snippet = snippet[:snippetMaxLen] + "..."
				}
				snippet = strings.TrimSpace(snippet)
				return SessionSummary{Path: path, ID: id, DisplayTime: displayTime, Snippet: snippet}
			}
		case "command":
			var payload struct {
				Command   string `json:"command"`
				Kind      string `json:"kind"`
				SkillName string `json:"skill_name"`
			}
			if json.Unmarshal(ev.Payload, &payload) == nil && payload.Command != "" {
				if payload.Kind == "skill" && strings.TrimSpace(payload.SkillName) != "" {
					snippet = "Skill: " + strings.TrimSpace(payload.SkillName)
				} else {
					snippet = payload.Command
				}
				if len(snippet) > snippetMaxLen {
					snippet = snippet[:snippetMaxLen] + "..."
				}
				snippet = strings.TrimSpace(snippet)
				return SessionSummary{Path: path, ID: id, DisplayTime: displayTime, Snippet: snippet}
			}
		}
	}
	return SessionSummary{Path: path, ID: id, DisplayTime: displayTime, Snippet: snippet}
}
