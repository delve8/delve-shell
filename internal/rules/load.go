package rules

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"delve-shell/internal/config"
)

// Load reads all rule file contents from config.RulesDir(), sorts by filename, and concatenates for LLM system prompt or context.
func Load() (string, error) {
	dir := config.RulesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("--- ")
		b.WriteString(name)
		b.WriteString(" ---\n")
		b.Write(data)
	}
	return b.String(), nil
}
