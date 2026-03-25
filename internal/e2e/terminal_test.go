package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestTerminalE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e in short mode (PTY/TUI tests are slow and environment-sensitive)")
	}
	bin := buildBinary(t)
	root := t.TempDir()
	writeMinimalConfig(t, root)
	env := append(os.Environ(),
		"DELVE_SHELL_ROOT="+root,
	)

	for _, c := range TerminalCases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			if c.Skip != "" && !(c.Name == "TUI_approval_flow" && os.Getenv("E2E_LLM") == "1") {
				t.Skip(c.Skip)
			}
			runCase(t, bin, env, c)
		})
	}
}

// writeMinimalConfig writes a minimal config.yaml with llm.model set so the TUI starts without opening the Config LLM overlay.
func writeMinimalConfig(t *testing.T, root string) {
	t.Helper()
	cfgPath := filepath.Join(root, "config.yaml")
	body := "language: en\nllm:\n  model: gpt-4o-mini\n"
	if os.Getenv("E2E_LLM") == "1" {
		body = "language: en\nallowlist_auto_run: false\nllm:\n  base_url: ${LLM_BASE_URL}\n  api_key: ${LLM_API_KEY}\n  model: ${LLM_MODEL}\n"
	}
	if err := os.WriteFile(cfgPath, []byte(body), 0600); err != nil {
		t.Fatalf("write minimal config: %v", err)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "delve-shell")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/delve-shell")
	cmd.Dir = findModuleRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, out)
	}
	return bin
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func runCase(t *testing.T, binaryPath string, env []string, c Case) {
	t.Helper()
	ptmx, cmd, err := Spawn(binaryPath, env)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	stepTimeout := c.Timeout
	if stepTimeout == 0 {
		stepTimeout = DefaultStepTimeout
	}

	for i, step := range c.Steps {
		if step.Input != "" {
			if err := WriteLine(ptmx, step.Input); err != nil {
				t.Fatalf("step %d write: %v", i, err)
			}
			time.Sleep(400 * time.Millisecond)
		}
		if len(step.Expect) == 0 {
			continue
		}
		to := step.Timeout
		if to == 0 {
			to = stepTimeout
		}
		got, idx, err := ReadUntilAny(ptmx, step.Expect, to)
		if err != nil {
			t.Fatalf("step %d read: %v", i, err)
		}
		if idx < 0 {
			t.Fatalf("step %d: timeout waiting for any of %v; last output (len=%d):\n%s", i, step.Expect, len(got), truncate(got, 1200))
		}
		t.Logf("step %d: matched %q", i, step.Expect[idx])
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:] + "...[truncated]"
}
