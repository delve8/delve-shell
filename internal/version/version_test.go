package version

import "testing"

func TestString_DefaultVersionOnly(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = "dev", "unknown", "unknown"
	defer func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldBuildDate
	}()

	if got := String(); got != "dev" {
		t.Fatalf("String() = %q want %q", got, "dev")
	}
}

func TestString_IncludesCommitAndBuildDate(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = "v0.3.0-2-gabcdef", "abcdef0", "2026-04-12T10:20:30Z"
	defer func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldBuildDate
	}()

	want := "v0.3.0-2-gabcdef (commit abcdef0, built 2026-04-12T10:20:30Z)"
	if got := String(); got != want {
		t.Fatalf("String() = %q want %q", got, want)
	}
}
