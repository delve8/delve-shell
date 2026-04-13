package config

import (
	"testing"
)

func TestLoadAllowlist_mergesCustomOverlay(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	if err := EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := WriteLoadedAllowlistToPath(CustomAllowlistPath(), &LoadedAllowlist{
		Version: AllowlistSchemaVersion,
		Commands: map[string]ReadOnlyCLIPolicy{
			"mycmd": {Name: "mycmd"},
			"docker": {
				Name: "docker",
				Root: &RootSpec{
					Flags:    NewFlagNone(),
					Operands: NewOperandsNone(),
					Subcommands: SubcommandMap{
						"version": {},
					},
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	ld, err := LoadAllowlist()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ld.Commands["mycmd"]; !ok {
		t.Fatal("custom command should be merged into effective allowlist")
	}
	docker, ok := ld.Commands["docker"]
	if !ok {
		t.Fatal("expected docker policy to exist")
	}
	if docker.Root == nil || len(docker.Root.Subcommands) != 1 {
		t.Fatalf("expected custom docker policy to override built-in, got %#v", docker.Root)
	}
	if _, ok := docker.Root.Subcommands["version"]; !ok {
		t.Fatal("expected custom docker override to win")
	}
}

func TestLoadCustomAllowlist_createsEmptyDefault(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	ld, err := LoadCustomAllowlist()
	if err != nil {
		t.Fatal(err)
	}
	if ld.Version != AllowlistSchemaVersion {
		t.Fatalf("version=%d want %d", ld.Version, AllowlistSchemaVersion)
	}
	if len(ld.Commands) != 0 {
		t.Fatalf("commands len=%d want 0", len(ld.Commands))
	}
}
