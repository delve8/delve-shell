package config

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func countLeadingSpaces(line string) int {
	n := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

func TestAllowlistSyncWithDefaults_rewritesNonDefaultFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	path := AllowlistPath()
	stale := []byte("version: 2\ncommands:\n  pwd: {}\n")
	if err := os.WriteFile(path, stale, 0600); err != nil {
		t.Fatal(err)
	}
	updated, err := AllowlistSyncWithDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("want updated=true for stale file")
	}
	want, err := EncodeAllowlistYAML(DefaultLoadedAllowlist())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatal("file should match embedded default")
	}
	updated2, err := AllowlistSyncWithDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if updated2 {
		t.Fatal("want updated=false when file already matches default")
	}
}

func TestEncodeAllowlistYAML_twoSpaceIndent(t *testing.T) {
	ld := &LoadedAllowlist{
		Version: AllowlistSchemaVersion,
		Commands: map[string]ReadOnlyCLIPolicy{
			"pwd": {Name: "pwd"},
		},
	}
	b, err := EncodeAllowlistYAML(ld)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	var cmdDepth, pwdDepth int
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "commands:") {
			cmdDepth = countLeadingSpaces(ln)
		}
		if strings.HasPrefix(trim, "pwd:") {
			pwdDepth = countLeadingSpaces(ln)
		}
	}
	if pwdDepth == 0 {
		t.Fatalf("missing pwd key, output:\n%s", string(b))
	}
	if cmdDepth > pwdDepth {
		t.Fatalf("invalid indent (commands deeper than pwd), cmd=%d pwd=%d\n%s", cmdDepth, pwdDepth, string(b))
	}
	if pwdDepth-cmdDepth != 2 {
		t.Fatalf("want 2-space step from commands to pwd, got cmd=%d pwd=%d\n%s", cmdDepth, pwdDepth, string(b))
	}
}
