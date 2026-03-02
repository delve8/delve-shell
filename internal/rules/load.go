package rules

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"delve-shell/internal/config"
)

// Load 从 config.RulesDir() 读取所有规则文件内容，按文件名排序后拼接为一段文本，供注入 LLM system prompt 或上下文
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
