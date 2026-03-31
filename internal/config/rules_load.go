package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadRules reads all rule file contents from RulesDir(), sorts by filename, and concatenates
// for LLM system prompt or context.
func LoadRules() (string, error) {
	dir := RulesDir()
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
