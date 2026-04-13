package config

import (
	"bytes"
	"os"
)

// AllowlistSyncWithDefaults overwrites allowlist.yaml with the embedded built-in default whenever
// on-disk content (after leading/trailing whitespace trim) differs. No merge: local edits that
// are not byte-identical to the canonical default encoding are replaced.
func AllowlistSyncWithDefaults() (updated bool, err error) {
	path := AllowlistPath()
	def := DefaultLoadedAllowlist()
	want, err := EncodeAllowlistYAML(def)
	if err != nil {
		return false, err
	}
	if err := EnsureRootDir(); err != nil {
		return false, err
	}
	if _, err := ensureCustomAllowlistFile(); err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		return true, os.WriteFile(path, want, 0600)
	}
	if bytes.Equal(bytes.TrimSpace(data), bytes.TrimSpace(want)) {
		return false, nil
	}
	return true, os.WriteFile(path, want, 0600)
}

func ensureCustomAllowlistFile() (updated bool, err error) {
	path := CustomAllowlistPath()
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	return true, WriteLoadedAllowlistToPath(path, EmptyCustomLoadedAllowlist())
}
