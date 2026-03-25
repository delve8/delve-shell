package interactive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"delve-shell/internal/config"
)

func TestNeedsConfigLLMOverlay_NilConfig(t *testing.T) {
	if !NeedsConfigLLMOverlay(nil) {
		t.Fatal("nil config should require LLM overlay")
	}
}

func TestNeedsConfigLLMOverlay_ModelPresent(t *testing.T) {
	cases := []string{
		"gpt-4o-mini",
		" qwen ",
		"a",
		"0",
		"model-with-dashes",
		"org/model",
	}
	for _, m := range cases {
		t.Run(m, func(t *testing.T) {
			cfg := &config.Config{LLM: config.LLMConfig{Model: m}}
			if NeedsConfigLLMOverlay(cfg) {
				t.Fatalf("model %q should not require overlay", m)
			}
		})
	}
}

func TestNeedsConfigLLMOverlay_EmptyModel(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"\t",
		"\n",
		" \t \n ",
	}
	for _, m := range cases {
		t.Run(escapeLabel(m), func(t *testing.T) {
			cfg := &config.Config{LLM: config.LLMConfig{Model: m}}
			if !NeedsConfigLLMOverlay(cfg) {
				t.Fatalf("model %q should require overlay", m)
			}
		})
	}
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	if s == "" {
		return "empty"
	}
	return s
}

func TestNeedsConfigLLMOverlay_Table(t *testing.T) {
	type row struct {
		model string
		want  bool
	}
	rows := []row{
		{"", true},
		{" ", true},
		{"x", false},
		{"ok", false},
		{"  m  ", false},
		{strings.Repeat(" ", 20), true},
		{strings.Repeat("x", 200), false},
	}
	for i, tc := range rows {
		cfg := &config.Config{LLM: config.LLMConfig{Model: tc.model}}
		got := NeedsConfigLLMOverlay(cfg)
		if got != tc.want {
			t.Fatalf("row %d model %q: want %v got %v", i, tc.model, tc.want, got)
		}
	}
}

func TestRunPreflight_CreatesSessionInTempRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)

	pf, err := RunPreflight()
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	if pf == nil {
		t.Fatal("nil preflight result")
	}
	t.Cleanup(func() { _ = pf.InitialSession.Close() })

	if pf.InitialSession == nil {
		t.Fatal("nil initial session")
	}
	path := pf.InitialSession.Path()
	if path == "" {
		t.Fatal("empty session path")
	}
	if !strings.HasPrefix(path, filepath.Join(root, "sessions")) {
		t.Fatalf("session path should live under temp sessions dir: %s", path)
	}
	// Session file is created lazily on first append; path must still be non-empty and under sessions/.
}

func TestRunPreflight_RulesDirExists(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)

	pf, err := RunPreflight()
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	t.Cleanup(func() { _ = pf.InitialSession.Close() })

	rulesDir := filepath.Join(root, "rules")
	st, err := os.Stat(rulesDir)
	if err != nil || !st.IsDir() {
		t.Fatalf("rules dir missing: %v", err)
	}
}

func TestRunPreflight_LoadsRulesWhenPresent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)

	if err := os.MkdirAll(filepath.Join(root, "rules"), 0o700); err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(root, "rules", "z_demo.md")
	if err := os.WriteFile(rulePath, []byte("hello rules"), 0o600); err != nil {
		t.Fatal(err)
	}

	pf, err := RunPreflight()
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	t.Cleanup(func() { _ = pf.InitialSession.Close() })

	if !strings.Contains(pf.RulesText, "hello rules") {
		t.Fatalf("rules text should include file body, got: %q", pf.RulesText)
	}
	if !strings.Contains(pf.RulesText, "z_demo.md") {
		t.Fatalf("rules text should mention filename, got: %q", pf.RulesText)
	}
}

func TestRunPreflight_NeedConfigLLMMatchesHelper(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)

	pf, err := RunPreflight()
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	t.Cleanup(func() { _ = pf.InitialSession.Close() })

	if pf.NeedConfigLLM != NeedsConfigLLMOverlay(pf.Config) {
		t.Fatalf("NeedConfigLLM field inconsistent: pf=%v overlay=%v cfg=%v",
			pf.NeedConfigLLM, NeedsConfigLLMOverlay(pf.Config), pf.Config)
	}
}

func TestRunPreflight_WithMinimalConfigYAML(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)

	cfgPath := filepath.Join(root, "config.yaml")
	content := "llm:\n  model: \"custom-model\"\n  base_url: \"\"\n  api_key: \"\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	pf, err := RunPreflight()
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	t.Cleanup(func() { _ = pf.InitialSession.Close() })

	if pf.Config == nil {
		t.Fatal("expected config to load")
	}
	if strings.TrimSpace(pf.Config.LLM.Model) != "custom-model" {
		t.Fatalf("model: %q", pf.Config.LLM.Model)
	}
	if pf.NeedConfigLLM {
		t.Fatal("should not need LLM overlay when model set")
	}
}
