package inputhistory

import (
	"os"
	"strings"
	"testing"

	"delve-shell/internal/config"
)

func TestLoad_missingFileReturnsEmpty(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len=%d want 0", len(got))
	}
}

func TestSaveAndLoad_roundTripAndNormalize(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	var entries []string
	entries = append(entries, "", "  alpha  ", "\n\n", "beta")
	for i := 0; i < MaxEntries+10; i++ {
		entries = append(entries, " item "+strings.Repeat("x", i%3))
	}
	if err := Save(entries); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxEntries {
		t.Fatalf("len=%d want %d", len(got), MaxEntries)
	}
	if got[0] != "item x" {
		t.Fatalf("first kept entry=%q want %q", got[0], "item x")
	}
	if got[len(got)-1] != "item x" {
		t.Fatalf("last entry=%q want %q", got[len(got)-1], "item x")
	}
}

func TestLoad_invalidVersionFails(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.InputHistoryPath(), []byte(`{"version":99,"entries":["x"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected version error")
	}
}
