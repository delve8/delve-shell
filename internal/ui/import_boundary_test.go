package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Feature modules register into ui via bootstrap; ui should not take a direct dependency on them.
var forbiddenUIDirectImports = []string{
	"delve-shell/internal/config/llm",
	"delve-shell/internal/remote",
	"delve-shell/internal/run",
	"delve-shell/internal/session",
	"delve-shell/internal/skill",
}

func TestUIDirectImportsDoNotIncludeFeaturePackages(t *testing.T) {
	t.Helper()
	root := findModuleRoot(t)
	cmd := exec.Command("go", "list", "-test=false", "-f", "{{range .Imports}}{{.}}\n{{end}}", "delve-shell/internal/ui")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, imp := range lines {
		if imp == "" {
			continue
		}
		for _, bad := range forbiddenUIDirectImports {
			if imp == bad {
				t.Fatalf("ui must not import %s directly; use Register* from feature packages instead", bad)
			}
		}
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}
