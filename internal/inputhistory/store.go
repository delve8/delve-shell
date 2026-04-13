package inputhistory

import (
	"encoding/json"
	"fmt"
	"os"

	"delve-shell/internal/config"
)

const (
	// MaxEntries matches the in-UI input recall cap.
	MaxEntries = 1000

	storeSchemaVersion = 1
)

type fileData struct {
	Version int      `json:"version"`
	Entries []string `json:"entries"`
}

// Load returns persisted input history. Missing file is treated as empty history.
func Load() ([]string, error) {
	data, err := os.ReadFile(config.InputHistoryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, err
	}
	if fd.Version != storeSchemaVersion {
		return nil, fmt.Errorf("input history version %d, want %d", fd.Version, storeSchemaVersion)
	}
	return normalize(fd.Entries), nil
}

// Save persists input history entries atomically under the config root.
func Save(entries []string) error {
	if err := config.EnsureRootDir(); err != nil {
		return err
	}
	fd := fileData{
		Version: storeSchemaVersion,
		Entries: normalize(entries),
	}
	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := config.InputHistoryPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func normalize(entries []string) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		entry = trimInputHistoryEntry(entry)
		if entry == "" {
			continue
		}
		out = append(out, entry)
	}
	if len(out) > MaxEntries {
		out = out[len(out)-MaxEntries:]
	}
	return out
}

func trimInputHistoryEntry(entry string) string {
	start, end := 0, len(entry)
	for start < end {
		switch entry[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			goto leftDone
		}
	}
leftDone:
	for start < end {
		switch entry[end-1] {
		case ' ', '\t', '\n', '\r':
			end--
		default:
			goto rightDone
		}
	}
rightDone:
	return entry[start:end]
}
