package ui

import (
	"os"
	"path/filepath"
	"strings"
)

const pathCompletionMax = 10

// PathCandidates returns path completions for the given input (e.g. ~/.ssh/ -> ~/.ssh/id_rsa, ...).
// Uses ~ in results when input starts with ~ (no expansion to absolute path in display).
// When input is a directory (or ends with /), lists that directory (auto-expand).
// Shared by any path input with dropdown (auth identity key path, add-remote key path).
func PathCandidates(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	useTilde := strings.HasPrefix(input, "~")
	expanded := input
	if strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		expanded = filepath.Join(home, input[2:])
	} else if input == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		expanded = home
	}
	expanded = filepath.Clean(expanded)
	dir := expanded
	prefix := ""
	if !strings.HasSuffix(input, "/") && !strings.HasSuffix(expanded, string(filepath.Separator)) {
		dir = filepath.Dir(expanded)
		prefix = filepath.Base(expanded)
		// When the path exists and is a directory, list its contents (auto-expand).
		if prefix != "" && prefix != "." && prefix != ".." {
			childPath := filepath.Join(dir, prefix)
			if info, err := os.Stat(childPath); err == nil && info.IsDir() {
				dir = childPath
				prefix = ""
			}
		}
	}
	if dir == "" || dir == "." {
		wd, _ := os.Getwd()
		dir = wd
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	homePrefix := ""
	if useTilde {
		home, _ := os.UserHomeDir()
		homePrefix = filepath.Clean(home) + string(filepath.Separator)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		full := filepath.Join(dir, name)
		if useTilde && strings.HasPrefix(filepath.Clean(full), homePrefix) {
			rest := full[len(homePrefix):]
			full = "~/" + strings.ReplaceAll(rest, string(filepath.Separator), "/")
		}
		if e.IsDir() {
			full += "/"
		}
		out = append(out, full)
		if len(out) >= pathCompletionMax {
			break
		}
	}
	return out
}
