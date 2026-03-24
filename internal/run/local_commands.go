package run

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	localRunOnce sync.Once
	localRunCmds []string
)

// loadLocalRunCommands returns a cached list of executable names found in PATH.
func loadLocalRunCommands() []string {
	localRunOnce.Do(func() {
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			localRunCmds = nil
			return
		}
		seen := make(map[string]struct{}, 4096)
		for _, dir := range filepath.SplitList(pathEnv) {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if name == "" || strings.Contains(name, " ") {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				if info.Mode()&0o111 == 0 {
					continue
				}
				seen[name] = struct{}{}
			}
		}
		out := make([]string, 0, len(seen))
		for k := range seen {
			out = append(out, k)
		}
		sort.Strings(out)
		localRunCmds = out
	})
	return localRunCmds
}
