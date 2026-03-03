package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSensitivePatterns(t *testing.T) {
	p := DefaultSensitivePatterns()
	if len(p) == 0 {
		t.Fatal("DefaultSensitivePatterns should not be empty")
	}
	hasShadow := false
	for _, s := range p {
		if s == `/etc/shadow\b` {
			hasShadow = true
			break
		}
	}
	if !hasShadow {
		t.Error("DefaultSensitivePatterns should contain /etc/shadow\\b")
	}
}

func TestWriteSensitivePatterns_LoadSensitivePatterns_roundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	want := []string{`/etc/shadow\b`, `\.env\b`, `custom`}
	if err := WriteSensitivePatterns(want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSensitivePatterns()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Errorf("LoadSensitivePatterns: len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Errorf("LoadSensitivePatterns: [%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadSensitivePatterns_missingFile_writesDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	path := SensitivePatternsPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	got, err := LoadSensitivePatterns()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Error("LoadSensitivePatterns on missing file should write and return default")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("sensitive_patterns.yaml should be created: %v", err)
	}
}

func TestSensitivePatternsUpdateWithDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	// Start with only one custom pattern (no defaults)
	if err := WriteSensitivePatterns([]string{`custom_only`}); err != nil {
		t.Fatal(err)
	}
	added, err := SensitivePatternsUpdateWithDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if added == 0 {
		t.Error("SensitivePatternsUpdateWithDefaults should add missing default patterns")
	}
	got, err := LoadSensitivePatterns()
	if err != nil {
		t.Fatal(err)
	}
	hasCustom := false
	hasShadow := false
	for _, p := range got {
		if p == "custom_only" {
			hasCustom = true
		}
		if p == `/etc/shadow\b` {
			hasShadow = true
		}
	}
	if !hasCustom {
		t.Error("merged file should keep existing custom_only")
	}
	if !hasShadow {
		t.Error("merged file should contain default /etc/shadow\\b")
	}
}

func TestSensitivePatternsUpdateWithDefaults_noChangeWhenComplete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	def := DefaultSensitivePatterns()
	if err := WriteSensitivePatterns(def); err != nil {
		t.Fatal(err)
	}
	added, err := SensitivePatternsUpdateWithDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if added != 0 {
		t.Errorf("when file already has all defaults, added should be 0, got %d", added)
	}
	got, err := LoadSensitivePatterns()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(def) {
		t.Errorf("length unchanged: got %d, want %d", len(got), len(def))
	}
}

func TestSensitivePatternsUpdateWithDefaults_missingFile_createsDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	path := SensitivePatternsPath()
	_ = os.Remove(path)
	added, err := SensitivePatternsUpdateWithDefaults()
	if err != nil {
		t.Fatal(err)
	}
	def := DefaultSensitivePatterns()
	if added != len(def) {
		t.Errorf("when file missing, added = %d, want %d", added, len(def))
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should be created: %v", err)
	}
}

func TestSensitivePatternsPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")
	got := SensitivePatternsPath()
	want := filepath.Join(dir, "sensitive_patterns.yaml")
	if got != want {
		t.Errorf("SensitivePatternsPath() = %q, want %q", got, want)
	}
}
