package pathcomplete

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidates_FiltersByPrefix(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "alpha.txt")
	f2 := filepath.Join(dir, "beta.txt")
	if err := os.WriteFile(f1, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}

	out := Candidates(filepath.Join(dir, "a"))
	if len(out) == 0 {
		t.Fatalf("expected non-empty candidates")
	}
	foundAlpha := false
	for _, c := range out {
		if strings.HasSuffix(c, "alpha.txt") {
			foundAlpha = true
		}
		if strings.HasSuffix(c, "beta.txt") {
			t.Fatalf("unexpected beta candidate for prefix 'a': %q", c)
		}
	}
	if !foundAlpha {
		t.Fatalf("expected alpha candidate, got %#v", out)
	}
}

func TestCandidates_DirectoryHasTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	out := Candidates(filepath.Join(dir, "s"))
	if len(out) == 0 {
		t.Fatalf("expected at least one candidate")
	}
	if !strings.HasSuffix(out[0], "/") {
		t.Fatalf("expected directory candidate to end with '/', got %q", out[0])
	}
}
