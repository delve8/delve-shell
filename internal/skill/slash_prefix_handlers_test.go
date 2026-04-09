package skill

import (
	"os"
	"path/filepath"
	"testing"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycletype"
)

func TestHandleSlashConfigDelSkillPrefix_AppendsSuccessWithTrailingBlank(t *testing.T) {
	i18n.SetLang("en")
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(config.SkillsDir(), "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := handleSlashConfigDelSkillPrefix("demo")
	if len(res.Outputs) != 1 || res.Outputs[0].Transcript == nil {
		t.Fatalf("unexpected outputs: %#v", res.Outputs)
	}
	lines := res.Outputs[0].Transcript.Lines
	if len(lines) != 2 {
		t.Fatalf("transcript line count=%d want 2", len(lines))
	}
	if lines[0].Kind != inputlifecycletype.TranscriptLineSystemSuggest {
		t.Fatalf("line0 kind=%v want system suggest", lines[0].Kind)
	}
	if lines[1].Kind != inputlifecycletype.TranscriptLineBlank {
		t.Fatalf("line1 kind=%v want blank", lines[1].Kind)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected skill dir removed, stat err=%v", err)
	}
}
