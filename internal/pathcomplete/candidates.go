package pathcomplete

import (
	"os"
	"path/filepath"
	"strings"
)

const maxCandidates = 10

// Candidates returns path completions for the given input.
func Candidates(input string) []string {
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
		if len(out) >= maxCandidates {
			break
		}
	}
	return out
}
